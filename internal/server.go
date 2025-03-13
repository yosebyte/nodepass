package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/yosebyte/x/io"
	"github.com/yosebyte/x/log"
	"github.com/yosebyte/x/pool"
)

type Server struct {
	Common
	pool            *pool.Pool
	sharedMU        sync.Mutex
	tunnelListen    net.Listener
	remoteListen    net.Listener
	targetTCPListen *net.TCPListener
	tlsConfig       *tls.Config
	semaphore       chan struct{}
}

func NewServer(parsedURL *url.URL, tlsConfig *tls.Config, logger *log.Logger) *Server {
	common := &Common{
		logger:  logger,
		errChan: make(chan error, 1),
	}
	common.GetAddress(parsedURL, logger)
	return &Server{
		Common:    *common,
		tlsConfig: tlsConfig,
		semaphore: make(chan struct{}, SemaphoreLimit),
	}
}

func (s *Server) Start() error {
	if err := s.initListener(); err != nil {
		s.logger.Error("Initialize failed: %v", err)
	}
	if err := s.startTunnelConnection(); err != nil {
		s.logger.Error("Tunnel connection error: %v", err)
		return err
	}
	if err := s.startRemoteListener(); err != nil {
		s.logger.Error("Remote listener error: %v", err)
		return err
	}
	s.pool = pool.NewServerPool(MaxPoolCapacity, s.remoteListen)
	go s.pool.ServerManager()
	go s.serverLaunch()
	return <-s.errChan
}

func (s *Server) initListener() error {
	tunnelListen, err := tls.Listen("tcp", s.tunnelAddr.String(), s.tlsConfig)
	if err != nil {
		s.logger.Error("Listen failed: %v", err)
		return err
	}
	s.tunnelListen = tunnelListen
	targetTCPListen, err := net.ListenTCP("tcp", s.targetTCPAddr)
	if err != nil {
		s.logger.Error("Listen failed: %v", err)
		return err
	}
	s.targetTCPListen = targetTCPListen
	targetUDPConn, err := net.ListenUDP("udp", s.targetUDPAddr)
	if err != nil {
		s.logger.Error("Listen failed: %v", err)
		return err
	}
	s.targetUDPConn = targetUDPConn
	s.logger.Debug("Waiting for connection: %v", s.tunnelListen.Addr())
	return nil
}

func (s *Server) startTunnelConnection() error {
	tunnelConn, err := s.tunnelListen.Accept()
	if err != nil {
		s.logger.Error("Listen failed: %v", err)
		return err
	}
	s.tunnelConn = tunnelConn.(*tls.Conn)
	s.logger.Debug("Tunnel connection established from: %v", tunnelConn.RemoteAddr())
	return nil
}

func (s *Server) startRemoteListener() error {
	remoteListen, err := net.Listen("tcp", s.remoteAddr.String())
	if err != nil {
		s.logger.Error("Listen failed: %v", err)
		return err
	}
	s.remoteListen = remoteListen
	return nil
}

func (s *Server) serverLaunch() {
	go func() {
		s.errChan <- s.serverPingCheck()
	}()
	go func() {
		s.logger.Debug("Handling server TCP: %v", s.tunnelListen.Addr())
		s.handleServerTCP()
	}()
	go func() {
		s.logger.Debug("Handling server UDP: %v", s.tunnelListen.Addr())
		s.handleServerUDP()
	}()
}

func (s *Server) Stop() {
	if s.targetTCPListen != nil {
		s.targetTCPListen.Close()
		s.logger.Debug("Target TCP listener closed: %v", s.targetTCPListen.Addr())
	}
	if s.tunnelListen != nil {
		s.tunnelListen.Close()
		s.logger.Debug("Tunnel listener closed: %v", s.tunnelListen.Addr())
	}
	if s.remoteListen != nil {
		s.remoteListen.Close()
		s.logger.Debug("Remote listener closed: %v", s.remoteListen.Addr())
	}
	if s.targetTCPConn != nil {
		s.targetTCPConn.Close()
		s.logger.Debug("Target TCP connection closed: %v", s.targetTCPConn.LocalAddr())
	}
	if s.targetUDPConn != nil {
		s.targetUDPConn.Close()
		s.logger.Debug("Target UDP connection closed: %v", s.targetUDPConn.LocalAddr())
	}
	if s.tunnelConn != nil {
		s.tunnelConn.Close()
		s.logger.Debug("Tunnel connection closed: %v", s.tunnelConn.LocalAddr())
	}
	if s.pool != nil {
		s.pool.Close()
		s.logger.Debug("Remote connection pool closed")
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Stop()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (s *Server) serverPingCheck() error {
	for {
		time.Sleep(ReportInterval)
		s.sharedMU.Lock()
		_, err := s.tunnelConn.Write([]byte(CheckSignalPING))
		s.sharedMU.Unlock()
		if err != nil {
			s.logger.Error("PING check failed: %v", err)
			return err
		}
	}
}

func (s *Server) handleServerTCP() {
	for {
		targetConn, err := s.targetTCPListen.AcceptTCP()
		if err != nil {
			s.logger.Error("Accept failed: %v", err)
			return
		}
		defer func() {
			if targetConn != nil {
				targetConn.Close()
			}
		}()
		s.targetTCPConn = targetConn
		s.logger.Debug("Target connection: %v --- %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
		s.semaphore <- struct{}{}
		go func(targetConn *net.TCPConn) {
			defer func() { <-s.semaphore }()
			s.sharedMU.Lock()
			_, err = s.tunnelConn.Write([]byte(LaunchSignalTCP))
			s.sharedMU.Unlock()
			if err != nil {
				s.logger.Error("Write failed: %v", err)
				return
			}
			s.logger.Debug("TCP launch signal sent: %v", s.tunnelConn.RemoteAddr())
			id, remoteConn := s.pool.Get()
			if id == "" {
				s.logger.Error("Get failed: %v", remoteConn)
				return
			}
			s.logger.Debug("Remote connection ID: %v <- active %v", id, s.pool.Active())
			defer func() {
				if remoteConn != nil {
					remoteConn.Close()
				}
			}()
			s.remoteTCPConn = remoteConn
			s.logger.Debug("Remote connection: %v --- %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
			s.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
			_, _, err = io.DataExchange(remoteConn, targetConn)
			s.logger.Debug("Remote connection: %v -/- %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
			s.logger.Debug("Target connection: %v -/- %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
			s.logger.Debug("Exchange complete: %v", err)
		}(targetConn)
	}
}

func (s *Server) handleServerUDP() {
	for {
		buffer := make([]byte, UDPDataBuffer)
		n, clientAddr, err := s.targetUDPConn.ReadFromUDP(buffer)
		if err != nil {
			s.logger.Error("Read failed: %v", err)
			return
		}
		s.logger.Debug("Target connection: %v --- %v", s.targetUDPConn.LocalAddr(), clientAddr)
		s.sharedMU.Lock()
		_, err = s.tunnelConn.Write([]byte(LaunchSignalUDP))
		s.sharedMU.Unlock()
		if err != nil {
			s.logger.Error("Write failed: %v", err)
			return
		}
		s.logger.Debug("UDP launch signal sent: %v", s.tunnelConn.RemoteAddr())
		id, remoteConn := s.pool.Get()
		if id == "" {
			s.logger.Error("Get failed: %v", remoteConn)
			return
		}
		s.logger.Debug("Remote connection ID: %v <- active %v", id, s.pool.Active())
		defer func() {
			if remoteConn != nil {
				remoteConn.Close()
			}
		}()
		s.remoteUDPConn = remoteConn
		s.logger.Debug("Remote connection: %v --- %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
		s.semaphore <- struct{}{}
		go func(buffer []byte, n int, remoteConn net.Conn, clientAddr *net.UDPAddr) {
			defer func() { <-s.semaphore }()
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
			remoteConn.Close()
			s.logger.Debug("Remote connection: %v -/- %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
			s.logger.Debug("Target connection: %v -/- %v", s.targetUDPConn.LocalAddr(), clientAddr)
			s.logger.Debug("Transfer complete: %v -/- %v", remoteConn.LocalAddr(), s.targetUDPConn.LocalAddr())
		}(buffer, n, remoteConn, clientAddr)
	}
}
