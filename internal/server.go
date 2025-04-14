// 内部包，实现服务器模式功能
package internal

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

// Server 实现服务器模式功能
type Server struct {
	Common                          // 继承通用功能
	serverMU       sync.Mutex       // 服务器互斥锁
	tunnelListener net.Listener     // 隧道监听器
	remoteListener net.Listener     // 远程连接监听器
	targetListener *net.TCPListener // 目标监听器
	tlsConfig      *tls.Config      // TLS配置
	semaphore      chan struct{}    // 信号量通道
}

// NewServer 创建新的服务器实例
func NewServer(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *log.Logger) *Server {
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
	s.logger.Info("Server started: %v(%v)/%v", s.tunnelAddr, s.remoteAddr, s.targetAddr)

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

	// 初始化远程连接池
	s.remotePool = conn.NewServerPool(maxPoolCapacity, s.tlsConfig, s.remoteListener)

	go s.remotePool.ServerManager()
	go s.serverLaunch()
	go s.statsReporter()

	return s.healthCheck()
}

// Stop 停止服务器
func (s *Server) Stop() {
	// 取消上下文
	if s.cancel != nil {
		s.cancel()
	}

	// 关闭远程连接池
	if s.remotePool != nil {
		active := s.remotePool.Active()
		s.remotePool.Close()
		s.logger.Debug("Remote connection closed: active %v", active)
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

	// 关闭远程监听器
	if s.remoteListener != nil {
		s.remoteListener.Close()
		s.logger.Debug("Remote listener closed: %v", s.remoteListener.Addr())
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
	tunnelListener, err := net.Listen("tcp", s.tunnelAddr.String())
	if err != nil {
		return err
	}
	s.tunnelListener = tunnelListener

	// 初始化远程监听器
	remoteListener, err := net.ListenTCP("tcp", s.remoteAddr)
	if err != nil {
		return err
	}
	s.remoteListener = remoteListener

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

	// 构建并发送隧道URL到客户端
	tunnelURL := &url.URL{
		Host:     strconv.Itoa(s.remoteAddr.Port),
		Fragment: s.tlsCode,
	}
	_, err = s.tunnelTCPConn.Write([]byte(tunnelURL.String() + "\n"))
	if err != nil {
		return err
	}

	s.logger.Debug("Tunnel signal -> : %v -> %v", tunnelURL.String(), s.tunnelTCPConn.RemoteAddr())
	s.logger.Debug("Tunnel connection: %v <-> %v", s.tunnelTCPConn.LocalAddr(), s.tunnelTCPConn.RemoteAddr())
	return nil
}

// 启动服务器处理循环
func (s *Server) serverLaunch() {
	for {
		// 等待连接池准备就绪
		if s.remotePool.Ready() {
			go s.serverTCPLoop()
			go s.serverUDPLoop()
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// 健康检查
func (s *Server) healthCheck() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			// 发送心跳包
			if !s.serverMU.TryLock() {
				continue
			}
			_, err := s.tunnelTCPConn.Write([]byte("\n"))
			s.serverMU.Unlock()
			if err != nil {
				return err
			}
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
				id, remoteConn := s.remotePool.ServerGet()
				if remoteConn == nil {
					s.logger.Error("Get failed: %v", id)
					return
				}

				s.logger.Debug("Remote connection: %v <- active %v", id, s.remotePool.Active())

				defer func() {
					if remoteConn != nil {
						remoteConn.Close()
					}
				}()

				s.logger.Debug("Remote connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

				// 构建并发送启动URL到客户端
				launchURL := &url.URL{
					Host:     id,
					Fragment: "1", // TCP模式
				}

				s.serverMU.Lock()
				_, err = s.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
				s.serverMU.Unlock()

				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}

				s.logger.Debug("TCP launch signal: %v -> %v", id, s.tunnelTCPConn.RemoteAddr())
				s.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())

				// 交换数据
				bytesReceived, bytesSent, err := conn.DataExchange(remoteConn, targetConn)
				s.AddTCPStats(uint64(bytesReceived), uint64(bytesSent))

				if err == io.EOF {
					s.logger.Debug("Exchange complete: %v bytes exchanged", bytesReceived+bytesSent)
				} else {
					s.logger.Error("Exchange complete: %v", err)
				}
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
			id, remoteConn := s.remotePool.ServerGet()
			if remoteConn == nil {
				continue
			}

			s.logger.Debug("Remote connection: %v <- active %v", id, s.remotePool.Active())

			defer func() {
				if remoteConn != nil {
					remoteConn.Close()
				}
			}()

			s.logger.Debug("Remote connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

			// 使用信号量限制并发数
			s.semaphore <- struct{}{}

			go func(buffer []byte, n int, clientAddr *net.UDPAddr, remoteConn net.Conn) {
				defer func() { <-s.semaphore }()

				// 构建并发送启动URL到客户端
				launchURL := &url.URL{
					Host:     id,
					Fragment: "2", // UDP模式
				}

				s.serverMU.Lock()
				_, err = s.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
				s.serverMU.Unlock()

				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}

				s.logger.Debug("UDP launch signal: %v -> %v", id, s.tunnelTCPConn.RemoteAddr())
				s.logger.Debug("Starting transfer: %v <-> %v", remoteConn.LocalAddr(), s.targetUDPConn.LocalAddr())

				// 发送数据到远程连接
				_, err = remoteConn.Write(buffer[:n])
				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}

				// 读取远程连接的响应
				n, err = remoteConn.Read(buffer)
				if err != nil {
					s.logger.Error("Read failed: %v", err)
					return
				}

				// 将响应发送回客户端
				_, err = s.targetUDPConn.WriteToUDP(buffer[:n], clientAddr)
				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}

				s.AddUDPSent(uint64(n))
				bytesReceived, bytesSent := s.GetUDPStats()
				s.logger.Debug("Transfer complete: %v bytes transferred", bytesReceived+bytesSent)
			}(buffer, n, clientAddr, remoteConn)
		}
	}
}
