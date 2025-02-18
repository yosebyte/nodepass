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
)

type Server struct {
	Common
	sharedMU        sync.Mutex
	tunnelListen    net.Listener
	remoteListen    net.Listener
	targetTCPListen *net.TCPListener
	targetUDPListen *net.UDPConn
	tlsConfig       *tls.Config
	semaphore       chan struct{}
}

func NewServer(parsedURL *url.URL, tlsConfig *tls.Config, logger *log.Logger) *Server {
	enableTLS := parsedURL.Query().Get("tls") != "false"
	common := &Common{
		logger:    logger,
		enableTLS: enableTLS,
		errChan:   make(chan error, 1),
	}
	common.GetAddress(parsedURL, logger)
	return &Server{
		Common:    *common,
		tlsConfig: tlsConfig,
		semaphore: make(chan struct{}, MaxSemaphoreLimit),
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
	targetUDPListen, err := net.ListenUDP("udp", s.targetUDPAddr)
	if err != nil {
		s.logger.Error("Listen failed: %v", err)
		return err
	}
	s.targetUDPListen = targetUDPListen
	s.logger.Debug("Waiting for connection: %v", s.tunnelListen.Addr())
	return nil
}

func (s *Server) startTunnelConnection() error {
	tunnelConn, err := s.tunnelListen.Accept()
	if err != nil {
		s.logger.Error("Listen failed: %v", err)
		return err
	}
	s.tunnelConn = tunnelConn
	s.logger.Debug("Tunnel connection established from: %v", tunnelConn.RemoteAddr())
	return nil
}

func (s *Server) startRemoteListener() error {
	if s.enableTLS {
		s.logger.Debug("Remote TLS enabled: %v", s.remoteAddr)
		remoteListen, err := tls.Listen("tcp", s.remoteAddr.String(), s.tlsConfig)
		if err != nil {
			s.logger.Error("Listen failed: %v", err)
			return err
		}
		s.remoteListen = remoteListen
	} else {
		remoteListen, err := net.Listen("tcp", s.remoteAddr.String())
		if err != nil {
			s.logger.Error("Listen failed: %v", err)
			return err
		}
		s.remoteListen = remoteListen
	}
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
	if s.targetUDPListen != nil {
		s.targetUDPListen.Close()
		s.logger.Debug("Target UDP listener closed: %v", s.targetUDPListen.LocalAddr())
	}
	if s.tunnelListen != nil {
		s.tunnelListen.Close()
		s.logger.Debug("Tunnel listener closed: %v", s.tunnelListen.Addr())
	}
	if s.remoteListen != nil {
		s.remoteListen.Close()
		s.logger.Debug("Remote listener closed: %v", s.remoteListen.Addr())
	}
	if s.remoteTCPConn != nil {
		s.remoteTCPConn.Close()
		s.logger.Debug("Remote TCP connection closed: %v", s.remoteTCPConn.LocalAddr())
	}
	if s.remoteUDPConn != nil {
		s.remoteUDPConn.Close()
		s.logger.Debug("Remote UDP connection closed: %v", s.remoteUDPConn.LocalAddr())
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
		time.Sleep(MaxReportInterval)
		s.sharedMU.Lock()
		_, err := s.tunnelConn.Write([]byte(CheckSignalPING))
		s.sharedMU.Unlock()
		if err != nil {
			s.logger.Error("PING check failed: %v", err)
			return err
		}
		s.logger.Debug("PING check passed: %v", s.tunnelConn.RemoteAddr())
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
		s.logger.Debug("Target connection established from: %v", targetConn.RemoteAddr())
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
			remoteConn, err := s.remoteListen.Accept()
			if err != nil {
				s.logger.Error("Accept failed: %v", err)
				return
			}
			defer func() {
				if remoteConn != nil {
					remoteConn.Close()
				}
			}()
			s.remoteTCPConn = remoteConn
			s.logger.Debug("Remote connection established from: %v", remoteConn.RemoteAddr())
			s.logger.Debug("Starting exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
			if err := io.DataExchange(remoteConn, targetConn); err != nil {
				s.logger.Debug("Exchange complete: %v", err)
			}
		}(targetConn)
	}
}

func (s *Server) handleServerUDP() {
	for {
		buffer := make([]byte, MaxUDPDataBuffer)
		n, clientAddr, err := s.targetUDPListen.ReadFromUDP(buffer)
		if err != nil {
			s.logger.Error("Read failed: %v", err)
			return
		}
		s.sharedMU.Lock()
		_, err = s.tunnelConn.Write([]byte(LaunchSignalUDP))
		s.sharedMU.Unlock()
		if err != nil {
			s.logger.Error("Write failed: %v", err)
			return
		}
		s.logger.Debug("UDP launch signal sent: %v", s.tunnelConn.RemoteAddr())
		remoteConn, err := s.remoteListen.Accept()
		if err != nil {
			s.logger.Error("Accept failed: %v", err)
			return
		}
		s.remoteUDPConn = remoteConn
		s.logger.Debug("Remote connection established from: %v", remoteConn.RemoteAddr())
		s.semaphore <- struct{}{}
		go func(buffer []byte, n int, remoteConn net.Conn, clientAddr *net.UDPAddr) {
			defer func() { <-s.semaphore }()
			s.logger.Debug("Starting transfer: %v <-> %v", clientAddr, s.targetUDPListen.LocalAddr())
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
			_, err = s.targetUDPListen.WriteToUDP(buffer[:n], clientAddr)
			if err != nil {
				s.logger.Error("Write failed: %v", err)
				return
			}
			remoteConn.Close()
			s.logger.Debug("Transfer complete: %v", clientAddr)
		}(buffer, n, remoteConn, clientAddr)
	}
}
