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

	// 检查目标地址是否为本机接口地址之一
	if s.isLocalAddress(s.targetTCPAddr.IP) {
		// 初始化目标监听器
		if err := s.initTargetListener(); err != nil {
			return err
		}
		s.dataFlow = "-"
	} else {
		s.dataFlow = "+"
	}

	// 与客户端进行握手
	if err := s.tunnelHandshake(); err != nil {
		return err
	}

	// 初始化隧道连接池
	s.tunnelPool = pool.NewServerPool(s.clientIP, s.tlsConfig, s.tunnelListener)

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

	// 记录客户端IP
	s.clientIP = s.tunnelTCPConn.RemoteAddr().(*net.TCPAddr).IP.String()

	// 构建并发送隧道URL到客户端
	tunnelURL := &url.URL{
		Host:     s.dataFlow,
		Fragment: s.tlsCode,
	}
	_, err = s.tunnelTCPConn.Write([]byte(tunnelURL.String() + "\n"))
	if err != nil {
		return err
	}

	s.logger.Debug("Tunnel signal -> : %v -> %v", tunnelURL.String(), s.tunnelTCPConn.RemoteAddr())
	s.logger.Debug("Tunnel handshaked: %v <-> %v", s.tunnelTCPConn.LocalAddr(), s.tunnelTCPConn.RemoteAddr())
	return nil
}

// initTunnelListener 初始化隧道监听器
func (s *Server) initTunnelListener() error {
	// 初始化隧道监听器
	tunnelListener, err := net.ListenTCP("tcp", s.tunnelAddr)
	if err != nil {
		return err
	}
	s.tunnelListener = tunnelListener

	return nil
}

// isLocalAddress 检查IP地址是否为本机接口地址之一
func (s *Server) isLocalAddress(ip net.IP) bool {
	// 处理未指定的IP地址
	if ip.IsUnspecified() || ip == nil {
		return true
	}

	// 添加例外，另有他用
	if ip.Equal(net.ParseIP("127.1.1.1")) {
		return false
	}

	// 获取本机所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		s.logger.Error("Get interfaces failed: %v", err)
		return false
	}

	// 遍历所有网络接口
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 遍历接口的所有地址
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.Equal(ip) {
					return true
				}
			case *net.IPAddr:
				if v.IP.Equal(ip) {
					return true
				}
			}
		}
	}
	return false
}
