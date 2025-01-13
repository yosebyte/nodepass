package internal

import (
	"bufio"
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
	serverListen    net.Listener
	targetTCPListen *net.TCPListener
	targetUDPListen *net.UDPConn
	tlsConfig       *tls.Config
}

func NewServer(parsedURL *url.URL, tlsConfig *tls.Config, logger *log.Logger) *Server {
	common := &Common{
		logger: logger,
	}
	common.GetAddress(parsedURL, logger)
	return &Server{
		Common:    *common,
		tlsConfig: tlsConfig,
	}
}

func (s *Server) Start() error {
	serverListen, err := tls.Listen("tcp", s.serverAddr.String(), s.tlsConfig)
	if err != nil {
		s.logger.Error("Unable to listen server address: %v", s.serverAddr)
		return err
	}
	s.serverListen = serverListen
	defer func() {
		if s.serverListen != nil {
			s.serverListen.Close()
		}
	}()
	tunnleConn, err := serverListen.Accept()
	if err != nil {
		s.logger.Error("Unable to accept connections form server address: %v", s.serverAddr)
		return err
	}
	s.logger.Debug("Tunnel connection established from: %v", tunnleConn.RemoteAddr().String())
	s.tunnleConn = tunnleConn
	defer func() {
		if s.tunnleConn != nil {
			s.tunnleConn.Close()
		}
	}()
	targetTCPListen, err := net.ListenTCP("tcp", s.targetTCPAddr)
	if err != nil {
		s.logger.Error("Unable to listen target TCP address: [%v]", s.targetTCPAddr)
		return err
	}
	s.targetTCPListen = targetTCPListen
	defer func() {
		if s.targetTCPListen != nil {
			s.targetTCPListen.Close()
		}
	}()
	targetUDPListen, err := net.ListenUDP("udp", s.targetUDPAddr)
	if err != nil {
		s.logger.Error("Unable to listen target UDP address: [%v]", s.targetUDPAddr)
		return err
	}
	s.targetUDPListen = targetUDPListen
	defer func() {
		if s.targetUDPListen != nil {
			s.targetUDPListen.Close()
		}
	}()
	errChan := make(chan error, 1)
	go s.serverLaunch(errChan)
	return <-errChan
}

func (s *Server) Stop() {
	if s.serverListen != nil {
		s.serverListen.Close()
	}
	if s.targetTCPListen != nil {
		s.targetTCPListen.Close()
	}
	if s.targetUDPListen != nil {
		s.targetUDPListen.Close()
	}
	if s.tunnleConn != nil {
		s.tunnleConn.Close()
	}
	if s.targetTCPConn != nil {
		s.targetTCPConn.Close()
	}
	if s.targetUDPConn != nil {
		s.targetUDPConn.Close()
	}
	if s.remoteTCPConn != nil {
		s.remoteTCPConn.Close()
	}
	if s.remoteUDPConn != nil {
		s.remoteUDPConn.Close()
	}
}

func (s *Server) serverLaunch(errChan chan error) {
	go func() {
		errChan <- s.serverPing()
	}()
	go func() {
		errChan <- s.handleServerTCP()
	}()
	go func() {
		errChan <- s.handleServerUDP()
	}()
}

func (s *Server) serverPing() error {
	reader := bufio.NewReader(s.tunnleConn)
	for {
		time.Sleep(MaxReportInterval)
		s.sharedMU.Lock()
		_, err := s.tunnleConn.Write([]byte("[PING]\n"))
		s.sharedMU.Unlock()
		if err != nil {
			s.logger.Error("Tunnel connection health check failed")
			s.Stop()
			return err
		}
		s.tunnleConn.SetReadDeadline(time.Now().Add(MaxReportTimeout))
		line, err := reader.ReadString('\n')
		if err != nil || line != "[PONG]\n" {
			s.logger.Error("Tunnel connection health check failed")
			s.Stop()
			return err
		}
	}
}

func (s *Server) handleServerTCP() error {
	sem := make(chan struct{}, MaxSemaphoreLimit)
	for {
		targetConn, err := s.targetTCPListen.AcceptTCP()
		if err != nil {
			s.logger.Error("Unable to accept connections form target address: %v %v", s.targetTCPListen.Addr(), err)
			time.Sleep(1 * time.Second)
			continue
		}
		s.targetTCPConn = targetConn
		s.logger.Debug("Target connection established from: %v", targetConn.RemoteAddr())
		sem <- struct{}{}
		go func(targetConn *net.TCPConn) {
			defer func() { <-sem }()
			s.sharedMU.Lock()
			_, err = s.tunnleConn.Write([]byte("[NODEPASS]<TCP>\n"))
			s.sharedMU.Unlock()
			if err != nil {
				s.logger.Error("Unable to send signal: %v", err)
				return
			}
			remoteConn, err := s.serverListen.Accept()
			if err != nil {
				s.logger.Error("Unable to accept connections form link address: %v %v", s.serverListen.Addr(), err)
				return
			}
			s.remoteTCPConn = remoteConn
			s.logger.Debug("Remote connection established from: %v", remoteConn.RemoteAddr())
			defer func() {
				if remoteConn != nil {
					remoteConn.Close()
				}
			}()
			s.logger.Debug("Starting data exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
			if err := io.DataExchange(remoteConn, targetConn); err != nil {
				s.logger.Debug("Connection closed: %v", err)
			}
		}(targetConn)
	}
}

func (s *Server) handleServerUDP() error {
	sem := make(chan struct{}, MaxSemaphoreLimit)
	for {
		buffer := make([]byte, MaxUDPDataBuffer)
		n, clientAddr, err := s.targetUDPListen.ReadFromUDP(buffer)
		if err != nil {
			s.logger.Error("Unable to read from client address: %v %v", clientAddr, err)
			time.Sleep(1 * time.Second)
			continue
		}
		s.sharedMU.Lock()
		_, err = s.tunnleConn.Write([]byte("[NODEPASS]<UDP>\n"))
		s.sharedMU.Unlock()
		if err != nil {
			s.logger.Error("Unable to send signal: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		remoteConn, err := s.serverListen.Accept()
		if err != nil {
			s.logger.Error("Unable to accept connections from server address: %v %v", s.serverListen.Addr(), err)
			time.Sleep(1 * time.Second)
			continue
		}
		s.remoteUDPConn = remoteConn
		s.logger.Debug("Remote connection established from: %v", remoteConn.RemoteAddr())
		sem <- struct{}{}
		go func(buffer []byte, n int, remoteConn net.Conn, clientAddr *net.UDPAddr) {
			defer func() { <-sem }()
			s.logger.Debug("Starting data transfer: %v <-> %v", clientAddr, s.targetUDPListen.LocalAddr())
			_, err = remoteConn.Write(buffer[:n])
			if err != nil {
				s.logger.Error("Unable to write to server address: %v %v", s.serverListen.Addr(), err)
				return
			}
			n, err = remoteConn.Read(buffer)
			if err != nil {
				s.logger.Error("Unable to read from server address: %v %v", s.serverListen.Addr(), err)
				return
			}
			_, err = s.targetUDPListen.WriteToUDP(buffer[:n], clientAddr)
			if err != nil {
				s.logger.Error("Unable to write to client address: %v %v", clientAddr, err)
				return
			}
			s.logger.Debug("Transfer completed successfully")
		}(buffer, n, remoteConn, clientAddr)
	}
}
