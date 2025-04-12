package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	nquic "github.com/yosebyte/nodepass/internal/quic"
	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type quicServer struct {
	common
	serverMU       sync.Mutex
	tunnelListener net.Listener
	quicListener   quic.Listener
	remoteListener net.Listener
	targetListener *net.TCPListener
	tlsConfig      *tls.Config
	semaphore      chan struct{}
	quicServer     *nquic.Server
	quicPool       *nquic.Pool
}

func NewQuicServer(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *log.Logger) *quicServer {
	common := &common{
		tlsCode: tlsCode,
		logger:  logger,
	}
	common.getAddress(parsedURL)
	return &quicServer{
		common:    *common,
		tlsConfig: tlsConfig,
		semaphore: make(chan struct{}, semaphoreLimit),
	}
}

func (s *quicServer) Start() error {
	s.initContext()
	if err := s.initListener(); err != nil {
		return err
	}
	if err := s.tunnelHandshake(); err != nil {
		return err
	}
	
	// 启动QUIC服务器
	quicAddr := &net.UDPAddr{
		IP:   s.remoteAddr.IP,
		Port: s.remoteAddr.Port,
	}
	s.quicServer = nquic.NewServer(quicAddr.String(), s.tlsConfig, s.logger)
	if err := s.quicServer.Start(); err != nil {
		return err
	}
	
	// 初始化QUIC连接池
	s.quicPool = nquic.NewServerPool(maxPoolCapacity, s.tlsConfig, s.quicServer.GetListener(), s.logger)
	
	// 启动标准TCP连接池
	s.remotePool = conn.NewServerPool(maxPoolCapacity, s.tlsConfig, s.remoteListener)
	
	go s.remotePool.ServerManager()
	go s.quicPool.ServerManager()
	go s.serverLaunch()
	return s.healthCheck()
}

func (s *quicServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.remotePool != nil {
		active := s.remotePool.Active()
		s.remotePool.Close()
		s.logger.Debug("Remote connection closed: active %v", active)
	}
	if s.quicPool != nil {
		active := s.quicPool.Active()
		s.quicPool.Close()
		s.logger.Debug("QUIC connection closed: active %v", active)
	}
	if s.quicServer != nil {
		s.quicServer.Stop()
		s.logger.Debug("QUIC server stopped")
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

func (s *quicServer) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx, s.Stop)
}

func (s *quicServer) initListener() error {
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

func (s *quicServer) tunnelHandshake() error {
	tunnelTCPConn, err := s.tunnelListener.Accept()
	if err != nil {
		return err
	}
	s.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
	tunnelURL := &url.URL{
		Host:     strconv.Itoa(s.remoteAddr.Port),
		Fragment: s.tlsCode,
		// 添加QUIC协议标识
		Scheme: "quic",
	}
	_, err = s.tunnelTCPConn.Write([]byte(tunnelURL.String() + "\n"))
	if err != nil {
		return err
	}
	s.logger.Debug("Tunnel signal -> : %v -> %v", tunnelURL.String(), s.tunnelTCPConn.RemoteAddr())
	s.logger.Debug("Tunnel connection: %v <-> %v", s.tunnelTCPConn.LocalAddr(), s.tunnelTCPConn.RemoteAddr())
	return nil
}

func (s *quicServer) serverLaunch() {
	for {
		if s.remotePool.Ready() && s.quicPool.Ready() {
			go s.serverTCPLoop()
			go s.serverUDPLoop()
			go s.serverQuicLoop()
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func (s *quicServer) serverQuicLoop() {
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
			s.logger.Debug("Target connection (QUIC): %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
			s.semaphore <- struct{}{}
			go func(targetConn net.Conn) {
				defer func() { <-s.semaphore }()
				id, remoteConn := s.quicPool.ServerGet()
				if remoteConn == nil {
					s.logger.Error("Get QUIC connection failed: %v", id)
					return
				}
				s.logger.Debug("QUIC connection: %v <- active %v", id, s.quicPool.Active())
				defer func() {
					if remoteConn != nil {
						remoteConn.Close()
					}
				}()
				s.logger.Debug("QUIC connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
				launchURL := &url.URL{
					Host:     id,
					Fragment: "3", // 使用3表示QUIC连接
				}
				s.serverMU.Lock()
				_, err = s.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
				s.serverMU.Unlock()
				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}
				s.logger.Debug("QUIC launch signal: %v -> %v", id, s.tunnelTCPConn.RemoteAddr())
				s.logger.Debug("Starting QUIC exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
				bytesReceived, bytesSent, err := conn.DataExchange(remoteConn, targetConn)
				s.AddTCPStats(uint64(bytesReceived), uint64(bytesSent))
				if err == io.EOF {
					s.logger.Debug("QUIC exchange complete: %v bytes exchanged", bytesReceived+bytesSent)
				} else {
					s.logger.Error("QUIC exchange complete: %v", err)
				}
			}(targetConn)
		}
	}
}
