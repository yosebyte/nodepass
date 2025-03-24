package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type server struct {
	common
	serverMU       sync.Mutex
	tunnelListener net.Listener
	remoteListener *net.TCPListener
	targetListener *net.TCPListener
	tlsConfig      *tls.Config
	semaphore      chan struct{}
}

func NewServer(parsedURL *url.URL, tlsConfig *tls.Config, logger *log.Logger) *server {
	common := &common{
		logger: logger,
	}
	common.getAddress(parsedURL)
	return &server{
		common:    *common,
		tlsConfig: tlsConfig,
		semaphore: make(chan struct{}, semaphoreLimit),
	}
}

func (s *server) Start() error {
	s.initContext()
	if err := s.initListener(); err != nil {
		return err
	}
	if err := s.getTunnelConnection(); err != nil {
		return err
	}
	s.remotePool = conn.NewServerPool(maxPoolCapacity, s.remoteListener)
	go s.remotePool.ServerManager()
	go s.serverTCPLoop()
	go s.serverUDPLoop()
	return s.healthCheck()
}

func (s *server) Stop() {
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
	if s.tunnelConn != nil {
		s.tunnelConn.Close()
		s.logger.Debug("Tunnel connection closed: %v", s.tunnelConn.LocalAddr())
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

func (s *server) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx, s.Stop)
}

func (s *server) initListener() error {
	tunnelListener, err := tls.Listen("tcp", s.tunnelAddr.String(), s.tlsConfig)
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

func (s *server) getTunnelConnection() error {
	tunnelConn, err := s.tunnelListener.Accept()
	if err != nil {
		return err
	}
	s.tunnelConn = tunnelConn.(*tls.Conn)
	s.logger.Debug("Tunnel connection: %v <-> %v", s.tunnelConn.LocalAddr(), s.tunnelConn.RemoteAddr())
	return nil
}

func (s *server) healthCheck() error {
	remoteURL := &url.URL{
		Scheme: "remote",
		Host:   strconv.Itoa(s.remoteAddr.Port),
	}
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			if !s.serverMU.TryLock() {
				continue
			}
			_, err := s.tunnelConn.Write([]byte(remoteURL.String() + "\n"))
			s.serverMU.Unlock()
			if err != nil {
				return err
			}
			time.Sleep(reportInterval)
		}
	}
}

func (s *server) serverTCPLoop() {
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
				launchURL := &url.URL{
					Scheme: "tcp",
					Host:   id,
				}
				s.serverMU.Lock()
				_, err = s.tunnelConn.Write([]byte(launchURL.String() + "\n"))
				s.serverMU.Unlock()
				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}
				s.logger.Debug("Launch signal -> : %v -> %v", id, s.tunnelConn.RemoteAddr())
				s.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
				_, _, err = conn.DataExchange(remoteConn, targetConn)
				s.logger.Debug("Exchange complete: %v", err)
			}(targetConn)
		}
	}
}

func (s *server) serverUDPLoop() {
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
				launchURL := &url.URL{
					Scheme: "udp",
					Host:   id,
				}
				s.serverMU.Lock()
				_, err = s.tunnelConn.Write([]byte(launchURL.String() + "\n"))
				s.serverMU.Unlock()
				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}
				s.logger.Debug("Launch signal -> : %v -> %v", id, s.tunnelConn.RemoteAddr())
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
				s.logger.Debug("Transfer complete: %v", n)
			}(buffer, n, clientAddr, remoteConn)
		}
	}
}
