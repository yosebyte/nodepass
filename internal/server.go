package internal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/yosebyte/nodepass/internal/security"
	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

// Server 是服务端核心结构体
type Server struct {
	Common
	serverMU       sync.Mutex
	tunnelListener net.Listener
	remoteListener net.Listener
	targetListener *net.TCPListener
	tlsConfig      *tls.Config
	semaphore      chan struct{}
}

// NewServer 创建一个新的服务端实例
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

// Start 启动服务端
func (s *Server) Start() error {
	s.initContext()
	if err := s.initListener(); err != nil {
		return err
	}
	if err := s.tunnelHandshake(); err != nil {
		return err
	}
	s.remotePool = conn.NewServerPool(maxPoolCapacity, s.tlsConfig, s.remoteListener)
	go s.remotePool.ServerManager()
	go s.serverLaunch()
	go s.statsReporter()
	return s.healthCheck()
}

// Stop 停止服务端
func (s *Server) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.remotePool != nil {
		active := s.remotePool.Active()
		s.remotePool.Close()
		s.logger.Debug("Remote connection closed: active %v", active)
	}
	if s.targetUDPConn != nil {
		s.targetUDPConn.Close()
		s.logger.Debug("Target connection closed: %v", s.targetUDPConn.LocalAddr())
	}
	if s.targetTCPConn != nil {
		s.targetTCPConn.Close()
		s.logger.Debug("Target connection closed: %v", s.targetTCPConn.LocalAddr())
	}
	if s.tunnelTCPConn != nil {
		s.tunnelTCPConn.Close()
		s.logger.Debug("Tunnel connection closed: %v", s.tunnelTCPConn.LocalAddr())
	}
	if s.targetListener != nil {
		s.targetListener.Close()
		s.logger.Debug("Target listener closed: %v", s.targetListener.Addr())
	}
	if s.remoteListener != nil {
		s.remoteListener.Close()
		s.logger.Debug("Remote listener closed: %v", s.remoteListener.Addr())
	}
	if s.tunnelListener != nil {
		s.tunnelListener.Close()
		s.logger.Debug("Tunnel listener closed: %v", s.tunnelListener.Addr())
	}
}

// Shutdown 安全关闭服务端
func (s *Server) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx, s.Stop)
}

// initListener 初始化监听器
func (s *Server) initListener() error {
	tunnelListener, err := net.Listen("tcp", s.tunnelAddr.String())
	if err != nil {
		return err
	}
	s.tunnelListener = tunnelListener
	remoteListener, err := net.ListenTCP("tcp", s.remoteAddr)
	if err != nil {
		return err
	}
	s.remoteListener = remoteListener
	targetListener, err := net.ListenTCP("tcp", s.targetTCPAddr)
	if err != nil {
		return err
	}
	s.targetListener = targetListener
	targetUDPConn, err := net.ListenUDP("udp", s.targetUDPAddr)
	if err != nil {
		return err
	}
	s.targetUDPConn = targetUDPConn
	return nil
}

// tunnelHandshake 执行隧道握手
func (s *Server) tunnelHandshake() error {
	tunnelTCPConn, err := s.tunnelListener.Accept()
	if err != nil {
		return err
	}
	s.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
	
	// 创建安全管理器
	securityManager, err := NewSecurityManager(s.logger, s.tlsConfig)
	if err == nil {
		// 加载受信任证书
		if err := securityManager.LoadTrustedCertificates(); err != nil {
			s.logger.Warn("加载受信任证书失败: %v", err)
		}
		
		// 执行安全握手
		handshakeData := &security.HandshakeData{
			ServerName:      "nodepass-server",
			Port:            s.remoteAddr.Port,
			TLSMode:         s.tlsCode,
			SupportedProtos: []string{"tcp", "udp", "quic", "websocket"},
		}
		
		// 执行安全握手
		_, err = securityManager.SecureHandshake(s.tunnelTCPConn, true)
		if err != nil {
			s.logger.Warn("安全握手失败，将使用标准握手: %v", err)
		} else {
			s.logger.Debug("安全握手完成: 端口=%d, TLS模式=%s", s.remoteAddr.Port, s.tlsCode)
			// 存储安全管理器以供后续使用
			s.securityManager = securityManager
			s.logger.Debug("Tunnel connection: %v <-> %v", s.tunnelTCPConn.LocalAddr(), s.tunnelTCPConn.RemoteAddr())
			return nil
		}
	}
	
	// 如果安全握手失败或不可用，使用标准握手
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

// serverLaunch 启动服务
func (s *Server) serverLaunch() {
	for {
		if s.remotePool.Ready() {
			go s.serverTCPLoop()
			go s.serverUDPLoop()
			
			// 如果支持QUIC和WebSocket，启动相应的服务
			if s.supportsQuic {
				go s.serverQUICLoop()
			}
			if s.supportsWS {
				go s.serverWSLoop()
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// healthCheck 健康检查
func (s *Server) healthCheck() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
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

// serverTCPLoop 处理TCP连接
func (s *Server) serverTCPLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
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
			s.semaphore <- struct{}{}
			go func(targetConn net.Conn) {
				defer func() { <-s.semaphore }()
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
				
				// 使用安全消息发送启动信号（如果安全管理器可用）
				if s.securityManager != nil {
					launchData := map[string]interface{}{
						"id": id,
						"type": "tcp",
						"timestamp": time.Now().Unix(),
					}
					launchDataJSON, err := json.Marshal(launchData)
					if err != nil {
						s.logger.Error("序列化启动数据失败: %v", err)
						return
					}
					
					secureMsg, err := s.securityManager.CreateSecureMessage(string(launchDataJSON))
					if err != nil {
						s.logger.Error("创建安全消息失败: %v", err)
						return
					}
					
					s.serverMU.Lock()
					_, err = s.tunnelTCPConn.Write([]byte(secureMsg + "\n"))
					s.serverMU.Unlock()
					if err != nil {
						s.logger.Error("Write failed: %v", err)
						return
					}
					
					// 验证连接
					if !s.securityManager.IsConnectionVerified(remoteConn) {
						s.logger.Warn("连接未验证，但将继续处理: %v", remoteConn.RemoteAddr())
					}
				} else {
					// 使用标准方式发送启动信号
					launchURL := &url.URL{
						Host:     id,
						Fragment: "1",
					}
					s.serverMU.Lock()
					_, err = s.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
					s.serverMU.Unlock()
					if err != nil {
						s.logger.Error("Write failed: %v", err)
						return
					}
				}
				
				s.logger.Debug("TCP launch signal: %v -> %v", id, s.tunnelTCPConn.RemoteAddr())
				s.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
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

// serverUDPLoop 处理UDP连接
func (s *Server) serverUDPLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			buffer := make([]byte, udpDataBufSize)
			n, clientAddr, err := s.targetUDPConn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}
			s.AddUDPReceived(uint64(n))
			s.logger.Debug("Target connection: %v <-> %v", s.targetUDPConn.LocalAddr(), clientAddr)
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
			s.semaphore <- struct{}{}
			go func(buffer []byte, n int, clientAddr *net.UDPAddr, remoteConn net.Conn) {
				defer func() { <-s.semaphore }()
				
				// 使用安全消息发送启动信号（如果安全管理器可用）
				if s.securityManager != nil {
					launchData := map[string]interface{}{
						"id": id,
						"type": "udp",
						"timestamp": time.Now().Unix(),
					}
					launchDataJSON, err := json.Marshal(launchData)
					if err != nil {
						s.logger.Error("序列化启动数据失败: %v", err)
						return
					}
					
					secureMsg, err := s.securityManager.CreateSecureMessage(string(launchDataJSON))
					if err != nil {
						s.logger.Error("创建安全消息失败: %v", err)
						return
					}
					
					s.serverMU.Lock()
					_, err = s.tunnelTCPConn.Write([]byte(secureMsg + "\n"))
					s.serverMU.Unlock()
					if err != nil {
						s.logger.Error("Write failed: %v", err)
						return
					}
					
					// 验证连接
					if !s.securityManager.IsConnectionVerified(remoteConn) {
						s.logger.Warn("连接未验证，但将继续处理: %v", remoteConn.RemoteAddr())
					}
					
					s.logger.Debug("UDP launch signal: %v -> %v", id, s.tunnelTCPConn.RemoteAddr())
					s.logger.Debug("Starting transfer: %v <-> %v", remoteConn.LocalAddr(), s.targetUDPConn.LocalAddr())
					
					// 创建安全消息包装UDP数据
					dataMsg, err := s.securityManager.CreateSecureMessage(string(buffer[:n]))
					if err != nil {
						s.logger.Error("创建安全消息失败: %v", err)
						return
					}
					
					// 发送安全消息
					_, err = remoteConn.Write([]byte(dataMsg))
					if err != nil {
						s.logger.Error("Write failed: %v", err)
						return
					}
					
					// 接收安全消息
					respBuffer := make([]byte, udpDataBufSize*2) // 增大缓冲区以容纳安全消息
					n, err = remoteConn.Read(respBuffer)
					if err != nil {
						s.logger.Error("Read failed: %v", err)
						return
					}
					
					// 验证并解析安全消息
					respData, err := s.securityManager.VerifySecureMessage(string(respBuffer[:n]))
					if err != nil {
						s.logger.Error("验证安全消息失败: %v", err)
						return
					}
					
					// 发送解析后的数据
					_, err = s.targetUDPConn.WriteToUDP([]byte(respData), clientAddr)
					if err != nil {
						s.logger.Error("Write failed: %v", err)
						return
					}
					s.AddUDPSent(uint64(len(respData)))
				} else {
					// 使用标准方式发送启动信号
					launchURL := &url.URL{
						Host:     id,
						Fragment: "2",
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
					_, err = remoteConn.Write(buffer[:n])
					if err != nil {
						s.logger.Error("Write failed: %v", err)
						return
					}
					n, err = remoteConn.Read(buffer)
					if err != nil {
						s.logger.Error("Read failed: %v", err)
						return
					}
					_, err = s.targetUDPConn.WriteToUDP(buffer[:n], clientAddr)
					if err != nil {
						s.logger.Error("Write failed: %v", err)
						return
					}
					s.AddUDPSent(uint64(n))
				}
				
				bytesReceived, bytesSent := s.GetUDPStats()
				s.logger.Debug("Transfer complete: %v bytes transferred", bytesReceived+bytesSent)
			}(buffer, n, clientAddr, remoteConn)
		}
	}
}

// serverQUICLoop 处理QUIC连接（占位符，实际实现在quic_server.go中）
func (s *Server) serverQUICLoop() {
	s.logger.Info("QUIC服务已启动")
	// 实际实现在quic_server.go中
}

// serverWSLoop 处理WebSocket连接（占位符，实际实现在ws_server.go中）
func (s *Server) serverWSLoop() {
	s.logger.Info("WebSocket服务已启动")
	// 实际实现在ws_server.go中
}
