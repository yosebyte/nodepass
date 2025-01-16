package internal

import (
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
	targetTCPListen *net.TCPListener
	targetUDPListen *net.UDPConn
	tlsConfig       *tls.Config
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
	}
}

func (s *Server) Init() error {
	tunnelListen, err := tls.Listen("tcp", s.tunnelAddr.String(), s.tlsConfig)
	if err != nil {
		s.logger.Error("Unable to listen server address: %v", s.tunnelAddr)
		return err
	}
	s.tunnelListen = tunnelListen
	targetTCPListen, err := net.ListenTCP("tcp", s.targetTCPAddr)
	if err != nil {
		s.logger.Error("Unable to listen target TCP address: [%v]", s.targetTCPAddr)
		return err
	}
	s.targetTCPListen = targetTCPListen
	targetUDPListen, err := net.ListenUDP("udp", s.targetUDPAddr)
	if err != nil {
		s.logger.Error("Unable to listen target UDP address: [%v]", s.targetUDPAddr)
		return err
	}
	s.targetUDPListen = targetUDPListen
	s.logger.Debug("Server initialized: %v", s.tunnelAddr)
	return nil
}

func (s *Server) Start() error {
	tunnelConn, err := s.tunnelListen.Accept()
	if err != nil {
		s.logger.Error("Unable to accept connections form server address: %v", s.tunnelAddr)
		return err
	}
	s.tunnelConn = tunnelConn
	s.logger.Debug("Tunnel connection established from: %v", tunnelConn.RemoteAddr())
	go s.serverLaunch()
	return <-s.errChan
}

func (s *Server) serverLaunch() {
	go func() {
		s.errChan <- s.serverPing()
	}()
	go func() {
		s.logger.Debug("Handling server TCP: %v", s.tunnelListen.Addr())
		s.errChan <- s.handleServerTCP()

	}()
	go func() {
		s.logger.Debug("Handling server UDP: %v", s.tunnelListen.Addr())
		s.errChan <- s.handleServerUDP()
	}()
}

func (s *Server) Stop() {
	if s.tunnelConn != nil {
		s.tunnelConn.Close()
		s.logger.Debug("Tunnel connection closed: %v", s.tunnelConn.RemoteAddr())
	}
	if s.targetTCPConn != nil {
		s.targetTCPConn.Close()
		s.logger.Debug("Target TCP connection closed: %v", s.targetTCPConn.RemoteAddr())
	}
	if s.targetUDPConn != nil {
		s.targetUDPConn.Close()
		s.logger.Debug("Target UDP connection closed: %v", s.targetUDPConn.RemoteAddr())
	}
	if s.remoteTCPConn != nil {
		s.remoteTCPConn.Close()
		s.logger.Debug("Remote TCP connection closed: %v", s.remoteTCPConn.RemoteAddr())
	}
	if s.remoteUDPConn != nil {
		s.remoteUDPConn.Close()
		s.logger.Debug("Remote UDP connection closed: %v", s.remoteUDPConn.RemoteAddr())
	}
}

func (s *Server) Shutdown() {
	s.Stop()
	if s.tunnelListen != nil {
		s.tunnelListen.Close()
		s.logger.Debug("Tunnel listener closed: %v", s.tunnelListen.Addr())
	}
	if s.targetTCPListen != nil {
		s.targetTCPListen.Close()
		s.logger.Debug("Target TCP listener closed: %v", s.targetTCPListen.Addr())
	}
	if s.targetUDPListen != nil {
		s.targetUDPListen.Close()
		s.logger.Debug("Target UDP listener closed: %v", s.targetUDPListen.LocalAddr())
	}
}

func (s *Server) serverPing() error {
	for {
		time.Sleep(MaxReportInterval)
		s.sharedMU.Lock()
		_, err := s.tunnelConn.Write([]byte("[NODEPASS]<PING>\n"))
		s.sharedMU.Unlock()
		if err != nil {
			s.logger.Error("Server PING failed: %v", s.tunnelConn.RemoteAddr())
			return err
		}
		s.logger.Debug("Server PING passed: %v", s.tunnelConn.RemoteAddr())
	}
}

func (s *Server) handleServerTCP() error {
	sem := make(chan struct{}, MaxSemaphoreLimit)
	for {
		targetConn, err := s.targetTCPListen.AcceptTCP()
		if err != nil {
			s.logger.Error("Unable to accept connections form target address: %v", err)
			return err
		}
		s.targetTCPConn = targetConn
		s.logger.Debug("Target connection established from: %v", targetConn.RemoteAddr())
		sem <- struct{}{}
		go func(targetConn *net.TCPConn) error {
			defer func() { <-sem }()
			s.sharedMU.Lock()
			_, err = s.tunnelConn.Write([]byte("[NODEPASS]<TCP>\n"))
			s.sharedMU.Unlock()
			if err != nil {
				s.logger.Error("Unable to send TCP launch signal: %v", err)
				return err
			}
			s.logger.Debug("TCP launch signal sent: %v", s.tunnelConn.RemoteAddr())
			remoteConn, err := s.tunnelListen.Accept()
			if err != nil {
				s.logger.Error("Unable to accept connections form tunnel address: %v %v", s.tunnelListen.Addr(), err)
				return err
			}
			defer func() {
				if remoteConn != nil {
					remoteConn.Close()
				}
			}()
			s.remoteTCPConn = remoteConn
			s.logger.Debug("Remote connection established from: %v", remoteConn.RemoteAddr())
			s.logger.Debug("Starting data exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
			if err := io.DataExchange(remoteConn, targetConn); err != nil {
				s.logger.Debug("Connection closed: %v", err)
			}
			return nil
		}(targetConn)
	}
}

func (s *Server) handleServerUDP() error {
	sem := make(chan struct{}, MaxSemaphoreLimit)
	for {
		buffer := make([]byte, MaxUDPDataBuffer)
		n, clientAddr, err := s.targetUDPListen.ReadFromUDP(buffer)
		if err != nil {
			s.logger.Error("Unable to read from client address: %v", err)
			return err
		}
		s.sharedMU.Lock()
		_, err = s.tunnelConn.Write([]byte("[NODEPASS]<UDP>\n"))
		s.sharedMU.Unlock()
		if err != nil {
			s.logger.Error("Unable to send UDP launch signal: %v", err)
			return err
		}
		s.logger.Debug("UDP launch signal sent: %v", s.tunnelConn.RemoteAddr())
		remoteConn, err := s.tunnelListen.Accept()
		if err != nil {
			s.logger.Error("Unable to accept connections from server address: %v", err)
			return err
		}
		s.remoteUDPConn = remoteConn
		s.logger.Debug("Remote connection established from: %v", remoteConn.RemoteAddr())
		sem <- struct{}{}
		go func(buffer []byte, n int, remoteConn net.Conn, clientAddr *net.UDPAddr) error {
			defer func() { <-sem }()
			s.logger.Debug("Starting data transfer: %v <-> %v", clientAddr, s.targetUDPListen.LocalAddr())
			_, err = remoteConn.Write(buffer[:n])
			if err != nil {
				s.logger.Error("Unable to write to server address: %v", err)
				return err
			}
			n, err = remoteConn.Read(buffer)
			if err != nil {
				s.logger.Error("Unable to read from server address: %v", err)
				return err
			}
			_, err = s.targetUDPListen.WriteToUDP(buffer[:n], clientAddr)
			if err != nil {
				s.logger.Error("Unable to write to client address: %v", err)
				return err
			}
			s.logger.Debug("Transfer completed successfully")
			return nil
		}(buffer, n, remoteConn, clientAddr)
	}
}
