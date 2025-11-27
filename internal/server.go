// 内部包，实现服务端模式功能
package internal

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/NodePassProject/logs"
	"github.com/NodePassProject/pool"
	"github.com/NodePassProject/quic"
)

// Server 实现服务端模式功能
type Server struct {
	Common          // 继承共享功能
	clientIP string // 客户端IP
}

// NewServer 创建新的服务端实例
func NewServer(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *logs.Logger) (*Server, error) {
	server := &Server{
		Common: Common{
			parsedURL:  parsedURL,
			tlsCode:    tlsCode,
			tlsConfig:  tlsConfig,
			logger:     logger,
			signalChan: make(chan string, semaphoreLimit),
			tcpBufferPool: &sync.Pool{
				New: func() any {
					buf := make([]byte, tcpDataBufSize)
					return &buf
				},
			},
			udpBufferPool: &sync.Pool{
				New: func() any {
					buf := make([]byte, udpDataBufSize)
					return &buf
				},
			},
			flushURL: &url.URL{Scheme: "np", Fragment: "f"},
			pingURL:  &url.URL{Scheme: "np", Fragment: "i"},
			pongURL:  &url.URL{Scheme: "np", Fragment: "o"},
		},
	}
	if err := server.initConfig(); err != nil {
		return nil, fmt.Errorf("newServer: initConfig failed: %w", err)
	}
	server.initRateLimiter()
	return server, nil
}

// Run 管理服务端生命周期
func (s *Server) Run() {
	logInfo := func(prefix string) {
		s.logger.Info("%v: server://%v@%v/%v?dns=%v&max=%v&mode=%v&quic=%v&dial=%v&read=%v&rate=%v&slot=%v&proxy=%v&notcp=%v&noudp=%v",
			prefix, s.tunnelKey, s.tunnelTCPAddr, s.getTargetAddrsString(), s.dnsCacheTTL, s.maxPoolCapacity,
			s.runMode, s.quicMode, s.dialerIP, s.readTimeout, s.rateLimit/125000, s.slotLimit,
			s.proxyProtocol, s.disableTCP, s.disableUDP)
	}
	logInfo("Server started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// 启动服务端并处理重启
	go func() {
		for ctx.Err() == nil {
			// 启动服务端
			if err := s.start(); err != nil && err != io.EOF {
				s.logger.Error("Server error: %v", err)
				// 重启服务端
				s.stop()
				select {
				case <-ctx.Done():
					return
				case <-time.After(serviceCooldown):
				}
				logInfo("Server restart")
			}
		}
	}()

	// 监听系统信号以优雅关闭
	<-ctx.Done()
	stop()

	// 执行关闭过程
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.shutdown(shutdownCtx, s.stop); err != nil {
		s.logger.Error("Server shutdown error: %v", err)
	} else {
		s.logger.Info("Server shutdown complete")
	}
}

// start 启动服务端
func (s *Server) start() error {
	// 初始化上下文
	s.initContext()

	// 初始化隧道监听器
	if err := s.initTunnelListener(); err != nil {
		return fmt.Errorf("start: initTunnelListener failed: %w", err)
	}

	// 运行模式判断
	switch s.runMode {
	case "1": // 反向模式
		if err := s.initTargetListener(); err != nil {
			return fmt.Errorf("start: initTargetListener failed: %w", err)
		}
		s.dataFlow = "-"
	case "2": // 正向模式
		s.dataFlow = "+"
	default: // 自动判断
		if err := s.initTargetListener(); err == nil {
			s.runMode = "1"
			s.dataFlow = "-"
		} else {
			s.runMode = "2"
			s.dataFlow = "+"
		}
	}

	// 与客户端进行握手
	if err := s.tunnelHandshake(); err != nil {
		return fmt.Errorf("start: tunnelHandshake failed: %w", err)
	}

	// 握手之后把UDP监听关掉
	if s.tunnelUDPConn != nil {
		s.tunnelUDPConn.Close()
	}

	// 初始化隧道连接池
	switch s.quicMode {
	case "0":
		tcpPool := pool.NewServerPool(
			s.maxPoolCapacity,
			s.clientIP,
			s.tlsConfig,
			s.tunnelListener,
			reportInterval)
		go tcpPool.ServerManager()
		s.tunnelPool = tcpPool
	case "1":
		udpPool := quic.NewServerPool(
			s.maxPoolCapacity,
			s.clientIP,
			s.tlsConfig,
			s.tunnelUDPAddr.String(),
			reportInterval)
		go udpPool.ServerManager()
		s.tunnelPool = udpPool
	default:
		return fmt.Errorf("start: unknown quic mode: %s", s.quicMode)
	}

	// 判断数据流向
	if s.dataFlow == "-" {
		go s.commonLoop()
	}

	// 启动共用控制
	if err := s.commonControl(); err != nil {
		return fmt.Errorf("start: commonControl failed: %w", err)
	}
	return nil
}

// tunnelHandshake 与客户端进行握手
func (s *Server) tunnelHandshake() error {
	type handshakeResult struct {
		netConn  net.Conn
		clientIP string
	}

	successChan := make(chan handshakeResult, 1)
	closeChan := make(chan struct{})
	var wg sync.WaitGroup

	go func() {
		for {
			select {
			case <-closeChan:
				return
			default:
			}

			// 接受隧道连接
			rawConn, err := s.tunnelListener.Accept()
			if err != nil {
				select {
				case <-closeChan:
					return
				default:
					continue
				}
			}

			// 并发处理握手
			wg.Add(1)
			go func(rawConn net.Conn) {
				defer wg.Done()

				select {
				case <-closeChan:
					rawConn.Close()
					return
				default:
				}

				bufReader := bufio.NewReader(rawConn)
				peek, err := bufReader.Peek(4)
				if err == nil && len(peek) == 4 && peek[3] == ' ' {
					clientIP := rawConn.RemoteAddr().(*net.TCPAddr).IP.String() + "\n"
					if peek[0] == 'G' && peek[1] == 'E' && peek[2] == 'T' {
						fmt.Fprintf(rawConn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(clientIP), clientIP)
					} else {
						fmt.Fprint(rawConn, "HTTP/1.1 405 Method Not Allowed\r\nAllow: GET\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
					}
					rawConn.Close()
					return
				}

				// 读取隧道密钥
				rawConn.SetReadDeadline(time.Now().Add(handshakeTimeout))
				rawTunnelKey, err := bufReader.ReadBytes('\n')
				if err != nil {
					s.logger.Warn("tunnelHandshake: read timeout: %v", rawConn.RemoteAddr())
					rawConn.Close()
					return
				}
				rawConn.SetReadDeadline(time.Time{})

				// 解码隧道密钥
				tunnelKeyData, err := s.decode(rawTunnelKey)
				if err != nil {
					s.logger.Warn("tunnelHandshake: decode failed: %v", rawConn.RemoteAddr())
					rawConn.Close()
					return
				}

				// 验证隧道密钥
				if string(tunnelKeyData) != s.tunnelKey {
					s.logger.Warn("tunnelHandshake: access denied: %v", rawConn.RemoteAddr())
					rawConn.Close()
					return
				}

				// 返回握手结果
				select {
				case successChan <- handshakeResult{
					netConn:  rawConn,
					clientIP: rawConn.RemoteAddr().(*net.TCPAddr).IP.String(),
				}:
					close(closeChan)
				case <-closeChan:
					rawConn.Close()
				}
			}(rawConn)
		}
	}()

	// 阻塞等待握手结果
	var result handshakeResult
	select {
	case result = <-successChan:
		wg.Wait()
	case <-s.ctx.Done():
		close(closeChan)
		wg.Wait()
		return fmt.Errorf("tunnelHandshake: context error: %w", s.ctx.Err())
	}

	// 保存握手结果
	s.controlConn = result.netConn
	s.clientIP = result.clientIP

	// 构建隧道配置信息
	tunnelURL := &url.URL{
		Scheme:   "np",
		User:     url.User(s.quicMode),
		Host:     strconv.Itoa(s.maxPoolCapacity),
		Path:     s.dataFlow,
		Fragment: s.tlsCode,
	}

	// 发送隧道配置信息
	_, err := s.controlConn.Write(s.encode([]byte(tunnelURL.String())))
	if err != nil {
		return fmt.Errorf("tunnelHandshake: write tunnel config failed: %w", err)
	}

	s.logger.Info("Tunnel signal -> : %v -> %v", tunnelURL.String(), s.controlConn.RemoteAddr())
	s.logger.Info("Tunnel handshaked: %v <-> %v", s.controlConn.LocalAddr(), s.controlConn.RemoteAddr())
	return nil
}
