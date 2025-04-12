package internal

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	nws "github.com/yosebyte/nodepass/internal/websocket"
	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type wsServer struct {
	common
	serverMU       sync.Mutex
	tunnelListener net.Listener
	wsServer       *nws.Server
	remoteListener net.Listener
	targetListener *net.TCPListener
	tlsConfig      *tls.Config
	semaphore      chan struct{}
	wsPool         *nws.Pool
}

func NewWSServer(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *log.Logger) *wsServer {
	common := &common{
		tlsCode: tlsCode,
		logger:  logger,
	}
	common.getAddress(parsedURL)
	return &wsServer{
		common:    *common,
		tlsConfig: tlsConfig,
		semaphore: make(chan struct{}, semaphoreLimit),
	}
}

func (s *wsServer) Start() error {
	s.initContext()
	if err := s.initListener(); err != nil {
		return err
	}
	if err := s.tunnelHandshake(); err != nil {
		return err
	}
	
	// 启动WebSocket服务器
	wsAddr := s.remoteAddr.String()
	s.wsServer = nws.NewServer(wsAddr, s.tlsConfig, s.logger)
	
	// 在单独的goroutine中启动WebSocket服务器
	go func() {
		if err := s.wsServer.Start(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("WebSocket server error: %v", err)
		}
	}()
	
	// 初始化WebSocket连接池
	s.wsPool = nws.NewServerPool(maxPoolCapacity, s.wsServer, s.logger)
	
	// 启动标准TCP连接池
	s.remotePool = conn.NewServerPool(maxPoolCapacity, s.tlsConfig, s.remoteListener)
	
	go s.remotePool.ServerManager()
	go s.wsPool.ServerManager(s.wsServer)
	go s.serverLaunch()
	return s.healthCheck()
}

func (s *wsServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.remotePool != nil {
		active := s.remotePool.Active()
		s.remotePool.Close()
		s.logger.Debug("Remote connection closed: active %v", active)
	}
	if s.wsPool != nil {
		active := s.wsPool.Active()
		s.wsPool.Close()
		s.logger.Debug("WebSocket connection closed: active %v", active)
	}
	if s.wsServer != nil {
		s.wsServer.Stop()
		s.logger.Debug("WebSocket server stopped")
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

func (s *wsServer) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx, s.Stop)
}

func (s *wsServer) initListener() error {
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

func (s *wsServer) tunnelHandshake() error {
	tunnelTCPConn, err := s.tunnelListener.Accept()
	if err != nil {
		return err
	}
	s.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
	
	// 确定WebSocket协议
	wsScheme := "ws"
	if s.tlsConfig != nil {
		wsScheme = "wss"
	}
	
	tunnelURL := &url.URL{
		Host:     strconv.Itoa(s.remoteAddr.Port),
		Fragment: s.tlsCode,
		// 添加WebSocket协议标识
		Scheme: wsScheme,
	}
	_, err = s.tunnelTCPConn.Write([]byte(tunnelURL.String() + "\n"))
	if err != nil {
		return err
	}
	s.logger.Debug("Tunnel signal -> : %v -> %v", tunnelURL.String(), s.tunnelTCPConn.RemoteAddr())
	s.logger.Debug("Tunnel connection: %v <-> %v", s.tunnelTCPConn.LocalAddr(), s.tunnelTCPConn.RemoteAddr())
	return nil
}

func (s *wsServer) serverLaunch() {
	for {
		if s.remotePool.Ready() && s.wsPool.Ready() {
			go s.serverTCPLoop()
			go s.serverUDPLoop()
			go s.serverWSLoop()
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func (s *wsServer) serverWSLoop() {
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
			s.logger.Debug("Target connection (WebSocket): %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
			s.semaphore <- struct{}{}
			go func(targetConn net.Conn) {
				defer func() { <-s.semaphore }()
				id, remoteConn := s.wsPool.ServerGet()
				if remoteConn == nil {
					s.logger.Error("Get WebSocket connection failed: %v", id)
					return
				}
				s.logger.Debug("WebSocket connection: %v <- active %v", id, s.wsPool.Active())
				defer func() {
					if remoteConn != nil {
						remoteConn.Close()
					}
				}()
				s.logger.Debug("WebSocket connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
				launchURL := &url.URL{
					Host:     id,
					Fragment: "4", // 使用4表示WebSocket连接
				}
				s.serverMU.Lock()
				_, err = s.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
				s.serverMU.Unlock()
				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}
				s.logger.Debug("WebSocket launch signal: %v -> %v", id, s.tunnelTCPConn.RemoteAddr())
				s.logger.Debug("Starting WebSocket exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
				bytesReceived, bytesSent, err := conn.DataExchange(remoteConn, targetConn)
				s.AddTCPStats(uint64(bytesReceived), uint64(bytesSent))
				if err == io.EOF {
					s.logger.Debug("WebSocket exchange complete: %v bytes exchanged", bytesReceived+bytesSent)
				} else {
					s.logger.Error("WebSocket exchange complete: %v", err)
				}
			}(targetConn)
		}
	}
}

func (s *wsServer) healthCheck() error {
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
