package internal

import (
	"context"
	"crypto/tls"
	"io"
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
	remoteListener net.Listener
	targetListener *net.TCPListener
	tlsConfig      *tls.Config
	semaphore      chan struct{}
}

func NewServer(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *log.Logger) *server {
	common := &common{
		tlsCode: tlsCode,
		logger:  logger,
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
	if err := s.tunnelHandshake(); err != nil {
		return err
	}
	s.remotePool = conn.NewServerPool(maxPoolCapacity, s.tlsConfig, s.remoteListener)
	go s.remotePool.ServerManager()
	go s.serverLaunch()
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

func (s *server) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx, s.Stop)
}

func (s *server) initListener() error {
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

func (s *server) tunnelHandshake() error {
	tunnelTCPConn, err := s.tunnelListener.Accept()
	if err != nil {
		return err
	}
	s.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
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

func (s *server) serverLaunch() {
	for {
		if s.remotePool.Ready() {
			go s.serverTCPLoop()
			go s.serverUDPLoop()
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func (s *server) healthCheck() error {
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
				bytesReceived, bytesSent := s.GetUDPStats()
				s.logger.Debug("Transfer complete: %v bytes transferred", bytesReceived+bytesSent)
			}(buffer, n, clientAddr, remoteConn)
		}
	}
}
