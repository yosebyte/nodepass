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
	remoteListener net.Listener
	targetListener net.Listener
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
	if s.cancel != nil {
		s.cancel()
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	if err := s.initListener(); err != nil {
		return err
	}
	if err := s.getTunnelConnection(); err != nil {
		return err
	}
	s.remotePool = conn.NewServerPool(maxPoolCapacity, s.remoteListener)
	go s.remotePool.ServerManager()
	go s.serverLoop()
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
	if s.targetConn != nil {
		s.targetConn.Close()
		s.logger.Debug("Target connection closed: %v", s.targetConn.LocalAddr())
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

func (s *server) initListener() error {
	tunnelListener, err := tls.Listen("tcp", s.tunnelAddr.String(), s.tlsConfig)
	if err != nil {
		return err
	}
	s.tunnelListener = tunnelListener
	remoteListener, err := net.Listen("tcp", s.remoteAddr.String())
	if err != nil {
		return err
	}
	s.remoteListener = remoteListener
	targetListener, err := net.Listen("tcp", s.targetAddr.String())
	if err != nil {
		return err
	}
	s.targetListener = targetListener
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

func (s *server) serverLoop() {
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
			s.targetConn = targetConn
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
					Scheme: "launch",
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
