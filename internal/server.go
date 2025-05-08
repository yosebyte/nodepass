// 内部包，实现服务器模式功能
package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/NodePassProject/conn"
	"github.com/NodePassProject/logs"
	"github.com/NodePassProject/pool"
)

// Server 实现服务器模式功能
type Server struct {
	Common                          // 继承通用功能
	mu             sync.Mutex       // 互斥锁
	tunnelListener net.Listener     // 隧道监听器
	targetListener *net.TCPListener // 目标监听器
	tlsConfig      *tls.Config      // TLS配置
	semaphore      chan struct{}    // 信号量通道
	clientIP       string           // 客户端IP
}

// NewServer 创建新的服务器实例
func NewServer(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *logs.Logger) *Server {
	server := &Server{
		Common: Common{
			tlsCode: tlsCode,
			logger:  logger,
		},
		tlsConfig: tlsConfig,
		semaphore: make(chan struct{}, semaphoreLimit),
	}
	server.getAddress(parsedURL)
	return server
}

// Manage 管理服务器生命周期
func (s *Server) Manage() {
	s.logger.Info("Server started: %v/%v", s.tunnelAddr, s.targetTCPAddr)

	// 启动服务器并处理重启
	go func() {
		for {
			if err := s.Start(); err != nil {
				s.logger.Error("Server error: %v", err)
				time.Sleep(serviceCooldown)
				s.Stop()
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
	if err := s.shutdown(shutdownCtx, s.Stop); err != nil {
		s.logger.Error("Server shutdown error: %v", err)
	} else {
		s.logger.Info("Server shutdown complete")
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.initContext()

	// 初始化监听器
	if err := s.initListener(); err != nil {
		return err
	}

	// 与客户端进行握手
	if err := s.tunnelHandshake(); err != nil {
		return err
	}

	// 初始化隧道连接池
	s.tunnelPool = pool.NewServerPool(s.clientIP, s.tlsConfig, s.tunnelListener)

	go s.tunnelPool.ServerManager()
	go s.serverLaunch()

	return s.healthCheck()
}

// Stop 停止服务器
func (s *Server) Stop() {
	// 取消上下文
	if s.cancel != nil {
		s.cancel()
	}

	// 关闭隧道连接池
	if s.tunnelPool != nil {
		active := s.tunnelPool.Active()
		s.tunnelPool.Close()
		s.logger.Debug("Tunnel connection closed: active %v", active)
	}

	// 关闭UDP连接
	if s.targetUDPConn != nil {
		s.targetUDPConn.Close()
		s.logger.Debug("Target connection closed: %v", s.targetUDPConn.LocalAddr())
	}

	// 关闭TCP连接
	if s.targetTCPConn != nil {
		s.targetTCPConn.Close()
		s.logger.Debug("Target connection closed: %v", s.targetTCPConn.LocalAddr())
	}

	// 关闭隧道连接
	if s.tunnelTCPConn != nil {
		s.tunnelTCPConn.Close()
		s.logger.Debug("Tunnel connection closed: %v", s.tunnelTCPConn.LocalAddr())
	}

	// 关闭目标监听器
	if s.targetListener != nil {
		s.targetListener.Close()
		s.logger.Debug("Target listener closed: %v", s.targetListener.Addr())
	}

	// 关闭隧道监听器
	if s.tunnelListener != nil {
		s.tunnelListener.Close()
		s.logger.Debug("Tunnel listener closed: %v", s.tunnelListener.Addr())
	}
}

// 初始化监听器
func (s *Server) initListener() error {
	// 初始化隧道监听器
	tunnelListener, err := net.ListenTCP("tcp", s.tunnelAddr)
	if err != nil {
		return err
	}
	s.tunnelListener = tunnelListener

	// 初始化目标TCP监听器
	targetListener, err := net.ListenTCP("tcp", s.targetTCPAddr)
	if err != nil {
		return err
	}
	s.targetListener = targetListener

	// 初始化目标UDP监听器
	targetUDPConn, err := net.ListenUDP("udp", s.targetUDPAddr)
	if err != nil {
		return err
	}
	s.targetUDPConn = targetUDPConn

	return nil
}

// 与客户端进行握手
func (s *Server) tunnelHandshake() error {
	// 接受隧道连接
	tunnelTCPConn, err := s.tunnelListener.Accept()
	if err != nil {
		return err
	}
	s.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)

	// 记录客户端IP
	s.clientIP = s.tunnelTCPConn.RemoteAddr().(*net.TCPAddr).IP.String()

	// 构建并发送隧道URL到客户端
	tunnelURL := &url.URL{
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

// 启动服务器处理循环
func (s *Server) serverLaunch() {
	for {
		// 等待连接池准备就绪
		if s.tunnelPool.Ready() {
			go s.serverTCPLoop()
			go s.serverUDPLoop()
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// 健康检查
func (s *Server) healthCheck() error {
	lastFlushed := time.Now()
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			// 发送心跳包
			if !s.mu.TryLock() {
				continue
			}
			// 定期刷新连接池
			if time.Since(lastFlushed) >= ReloadInterval {
				flushURL := &url.URL{
					Fragment: "0", // 刷新模式
				}

				_, err := s.tunnelTCPConn.Write([]byte(flushURL.String() + "\n"))
				if err != nil {
					s.mu.Unlock()
					return err
				}

				s.tunnelPool.Flush()
				lastFlushed = time.Now()
				time.Sleep(reportInterval) // 等待连接池刷新完成
				s.logger.Debug("Tunnel pool reset: %v active connections", s.tunnelPool.Active())
			} else {
				// 定期发送心跳包
				_, err := s.tunnelTCPConn.Write([]byte("\n"))
				if err != nil {
					s.mu.Unlock()
					return err
				}
			}
			s.mu.Unlock()
			time.Sleep(reportInterval)
		}
	}
}

// TCP请求处理循环
func (s *Server) serverTCPLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// 接受来自目标的TCP连接
			targetConn, err := s.targetListener.Accept()
			if err != nil {
				continue
			}

			defer func() {
				if targetConn != nil {
					targetConn.Close()
				}
			}()

			s.targetTCPConn = targetConn.(*net.TCPConn)
			s.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())

			// 使用信号量限制并发数
			s.semaphore <- struct{}{}

			go func(targetConn net.Conn) {
				defer func() { <-s.semaphore }()

				// 从连接池获取连接
				id, remoteConn := s.tunnelPool.ServerGet()
				if remoteConn == nil {
					s.logger.Error("Get failed: %v", id)
					return
				}

				s.logger.Debug("Tunnel connection: %v <- active %v", id, s.tunnelPool.Active())

				defer func() {
					if remoteConn != nil {
						remoteConn.Close()
					}
				}()

				s.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

				// 构建并发送启动URL到客户端
				launchURL := &url.URL{
					Host:     id,
					Fragment: "1", // TCP模式
				}

				s.mu.Lock()
				_, err = s.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
				s.mu.Unlock()

				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}

				s.logger.Debug("TCP launch signal: %v -> %v", id, s.tunnelTCPConn.RemoteAddr())
				s.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())

				// 交换数据
				bytesReceived, bytesSent, _ := conn.DataExchange(remoteConn, targetConn)
				s.AddTCPStats(uint64(bytesReceived), uint64(bytesSent))

				// 交换完成，广播统计信息
				s.logger.Debug("Exchange complete: TRAFFIC_STATS|TCP_RX=%v|TCP_TX=%v|UDP_RX=%v|UDP_TX=%v",
					s.tcpBytesReceived, s.tcpBytesSent, s.udpBytesReceived, s.udpBytesSent)
			}(targetConn)
		}
	}
}

// UDP请求处理循环
func (s *Server) serverUDPLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// 读取来自目标的UDP数据
			buffer := make([]byte, udpDataBufSize)
			n, clientAddr, err := s.targetUDPConn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			s.AddUDPReceived(uint64(n))
			s.logger.Debug("Target connection: %v <-> %v", s.targetUDPConn.LocalAddr(), clientAddr)

			// 从连接池获取连接
			id, remoteConn := s.tunnelPool.ServerGet()
			if remoteConn == nil {
				continue
			}

			s.logger.Debug("Tunnel connection: %v <- active %v", id, s.tunnelPool.Active())

			defer func() {
				if remoteConn != nil {
					remoteConn.Close()
				}
			}()

			s.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

			// 使用信号量限制并发数
			s.semaphore <- struct{}{}

			go func(buffer []byte, n int, clientAddr *net.UDPAddr, remoteConn net.Conn) {
				defer func() { <-s.semaphore }()

				// 构建并发送启动URL到客户端
				launchURL := &url.URL{
					Host:     id,
					Fragment: "2", // UDP模式
				}

				s.mu.Lock()
				_, err = s.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
				s.mu.Unlock()

				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}

				s.logger.Debug("UDP launch signal: %v -> %v", id, s.tunnelTCPConn.RemoteAddr())
				s.logger.Debug("Starting transfer: %v <-> %v", remoteConn.LocalAddr(), s.targetUDPConn.LocalAddr())

				// 处理UDP/TCP数据交换
				udpToTcp, tcpToUdp, err := conn.DataTransfer(
					s.targetUDPConn,
					remoteConn,
					clientAddr, // 有UDP地址，自动确定为服务器模式
					buffer[:n],
					udpDataBufSize,
					tcpReadTimeout,
				)

				if err != nil {
					s.logger.Error("Transfer failed: %v", err)
					return
				}

				s.AddUDPReceived(udpToTcp)
				s.AddUDPSent(tcpToUdp)

				// 传输完成，广播统计信息
				s.logger.Debug("Transfer complete: TRAFFIC_STATS|TCP_RX=%v|TCP_TX=%v|UDP_RX=%v|UDP_TX=%v",
					s.tcpBytesReceived, s.tcpBytesSent, s.udpBytesReceived, s.udpBytesSent)
			}(buffer, n, clientAddr, remoteConn)
		}
	}
}
