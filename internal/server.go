// 内部包，实现服务端模式功能
package internal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/NodePassProject/logs"
	"github.com/NodePassProject/nph2"
	"github.com/NodePassProject/npws"
	"github.com/NodePassProject/pool"
	"github.com/NodePassProject/quic"
)

// Server 实现服务端模式功能
type Server struct{ Common }

// NewServer 创建新的服务端实例
func NewServer(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *logs.Logger) (*Server, error) {
	server := &Server{
		Common: Common{
			parsedURL:  parsedURL,
			tlsCode:    tlsCode,
			tlsConfig:  tlsConfig,
			logger:     logger,
			signalChan: make(chan Signal, semaphoreLimit),
			writeChan:  make(chan []byte, semaphoreLimit),
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
		s.logger.Info("%v: server://%v@%v/%v?dns=%v&max=%v&mode=%v&type=%v&dial=%v&read=%v&rate=%v&slot=%v&proxy=%v&block=%v&notcp=%v&noudp=%v",
			prefix, s.tunnelKey, s.tunnelTCPAddr, s.getTargetAddrsString(), s.dnsCacheTTL, s.maxPoolCapacity,
			s.runMode, s.poolType, s.dialerIP, s.readTimeout, s.rateLimit/125000, s.slotLimit,
			s.proxyProtocol, s.blockProtocol, s.disableTCP, s.disableUDP)
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

	// 关闭UDP监听器
	if s.tunnelUDPConn != nil {
		s.tunnelUDPConn.Close()
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

	// 接受隧道握手
	s.logger.Info("Pending tunnel handshake...")
	s.handshakeStart = time.Now()
	if err := s.tunnelHandshake(); err != nil {
		return fmt.Errorf("start: tunnelHandshake failed: %w", err)
	}

	// 初始化连接池
	if err := s.initTunnelPool(); err != nil {
		return fmt.Errorf("start: initTunnelPool failed: %w", err)
	}

	// 设置控制连接
	s.logger.Info("Getting tunnel pool ready...")
	if err := s.setControlConn(); err != nil {
		return fmt.Errorf("start: setControlConn failed: %w", err)
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

// initTunnelPool 初始化隧道连接池
func (s *Server) initTunnelPool() error {
	switch s.poolType {
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
		quicPool := quic.NewServerPool(
			s.maxPoolCapacity,
			s.clientIP,
			s.tlsConfig,
			s.tunnelUDPAddr.String(),
			reportInterval)
		go quicPool.ServerManager()
		s.tunnelPool = quicPool
	case "2":
		websocketPool := npws.NewServerPool(
			s.maxPoolCapacity,
			"",
			s.tlsConfig,
			s.tunnelListener,
			reportInterval)
		go websocketPool.ServerManager()
		s.tunnelPool = websocketPool
	case "3":
		http2Pool := nph2.NewServerPool(
			s.maxPoolCapacity,
			s.clientIP,
			s.tlsConfig,
			s.tunnelListener,
			reportInterval)
		go http2Pool.ServerManager()
		s.tunnelPool = http2Pool
	default:
		return fmt.Errorf("initTunnelPool: unknown pool type: %s", s.poolType)
	}
	return nil
}

// tunnelHandshake 与客户端进行HTTP握手
func (s *Server) tunnelHandshake() error {
	if s.tlsCode == "1" || s.tlsCode == "2" {
		s.verifyChan = make(chan struct{})
	}

	var clientIP string
	done := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")

		// 验证请求
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// 验证路径
		if r.URL.Path != "/" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// 验证令牌
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || !s.verifyAuthToken(strings.TrimPrefix(auth, "Bearer ")) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 记录客户端地址
		clientIP = r.RemoteAddr
		if host, _, err := net.SplitHostPort(clientIP); err == nil {
			clientIP = host
		}

		// 发送配置
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"flow": s.dataFlow,
			"max":  s.maxPoolCapacity,
			"tls":  s.tlsCode,
			"type": s.poolType,
		})

		s.logger.Info("Sending tunnel config: FLOW=%v|MAX=%v|TLS=%v|TYPE=%v",
			s.dataFlow, s.maxPoolCapacity, s.tlsCode, s.poolType)

		close(done)
	})

	server := &http.Server{Handler: handler}
	go server.Serve(s.tunnelListener)

	select {
	case <-done:
		server.Close()
		s.clientIP = clientIP
		s.tunnelListener, _ = net.ListenTCP("tcp", s.tunnelTCPAddr)
		return nil
	case <-s.ctx.Done():
		server.Close()
		return fmt.Errorf("tunnelHandshake: context canceled")
	}
}
