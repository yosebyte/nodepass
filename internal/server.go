// 内部包，实现服务端模式功能
package internal

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NodePassProject/logs"
	"github.com/NodePassProject/pool"
)

// Server 实现服务端模式功能
type Server struct {
	Common                // 继承共享功能
	tlsConfig *tls.Config // TLS配置
	clientIP  string      // 客户端IP
}

// NewServer 创建新的服务端实例
func NewServer(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *logs.Logger) *Server {
	server := &Server{
		Common: Common{
			tlsCode:    tlsCode,
			dataFlow:   "+",
			logger:     logger,
			semaphore:  make(chan struct{}, semaphoreLimit),
			signalChan: make(chan string, semaphoreLimit),
		},
		tlsConfig: tlsConfig,
	}
	server.getAddress(parsedURL)
	return server
}

// Run 管理服务端生命周期
func (s *Server) Run() {
	s.logger.Info("Server started: %v/%v", s.tunnelAddr, s.targetTCPAddr)

	// 启动服务端并处理重启
	go func() {
		for {
			if err := s.start(); err != nil {
				s.logger.Error("Server error: %v", err)
				time.Sleep(serviceCooldown)
				s.stop()
				s.logger.Info("Server restarted")
			}
		}
	}()

	// 监听系统信号以优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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
	s.initContext()

	// 初始化隧道监听器
	if err := s.initTunnelListener(); err != nil {
		return err
	}

	// 通过目标地址判断数据流向
	if s.isLocalAddress(s.targetTCPAddr.IP) {
		if err := s.initTargetListener(); err == nil {
			s.dataFlow = "-"
		}
	}

	// 与客户端进行握手
	if err := s.tunnelHandshake(); err != nil {
		return err
	}

	// 握手之后把UDP监听关掉
	if s.tunnelUDPConn != nil {
		s.tunnelUDPConn.Close()
	}

	// 初始化隧道连接池
	s.tunnelPool = pool.NewServerPool(
		s.clientIP,
		s.tlsConfig,
		s.tunnelListener,
		reportInterval)

	go s.tunnelPool.ServerManager()

	switch s.dataFlow {
	case "-":
		go s.commonLoop()
	case "+":
		go s.commonOnce()
		go s.commonQueue()
	}
	return s.healthCheck()
}

// tunnelHandshake 与客户端进行握手
func (s *Server) tunnelHandshake() error {
	// 接受隧道连接
	tunnelTCPConn, err := s.tunnelListener.Accept()
	if err != nil {
		return err
	}
	s.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
	s.bufReader = bufio.NewReader(s.tunnelTCPConn)
	s.tunnelTCPConn.SetKeepAlive(true)
	s.tunnelTCPConn.SetKeepAlivePeriod(reportInterval)

	// 记录客户端IP
	s.clientIP = s.tunnelTCPConn.RemoteAddr().(*net.TCPAddr).IP.String()

	// 构建并发送隧道URL到客户端
	tunnelURL := &url.URL{
		Host:     s.dataFlow,
		Fragment: s.tlsCode,
	}

	start := time.Now()
	_, err = s.tunnelTCPConn.Write(append(xor([]byte(tunnelURL.String())), '\n'))
	if err != nil {
		return err
	}
	s.logger.Debug("Tunnel signal -> : %v -> %v", tunnelURL.String(), s.tunnelTCPConn.RemoteAddr())

	// 等待客户端确认
	_, err = s.tunnelTCPConn.Read(make([]byte, 1))
	if err != nil {
		return err
	}
	s.logger.Event("Tunnel handshaked: %v <-> %v in %vms",
		s.tunnelTCPConn.LocalAddr(), s.tunnelTCPConn.RemoteAddr(), time.Since(start).Milliseconds())

	// 发送客户端确认握手完成
	if _, err := s.tunnelTCPConn.Write([]byte{'\n'}); err != nil {
		return err
	}
	return nil
}
