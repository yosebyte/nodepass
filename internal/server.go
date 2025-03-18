package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/yosebyte/x/io"
	"github.com/yosebyte/x/log"
	"github.com/yosebyte/x/pool"
)

type Server struct {
	Common
	sharedMU     sync.Mutex
	tunnelListen net.Listener
	remoteListen net.Listener
	targetListen net.Listener
	tlsConfig    *tls.Config
	semaphore    chan struct{}
}

func NewServer(parsedURL *url.URL, tlsConfig *tls.Config, logger *log.Logger) *Server {
	common := &Common{
		logger: logger,
	}
	common.getAddress(parsedURL)
	return &Server{
		Common:    *common,
		tlsConfig: tlsConfig,
		semaphore: make(chan struct{}, SemaphoreLimit),
	}
}

func (s *Server) Start() error {
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
	s.remotePool = pool.NewServerPool(MaxPoolCapacity, s.remoteListen)
	go s.remotePool.ServerManager()
	go s.serverLoop()
	return s.healthCheck()
}

func (s *Server) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.targetListen != nil {
		s.targetListen.Close()
		s.logger.Debug("Target listener closed: %v", s.targetListen.Addr())
	}
	if s.tunnelListen != nil {
		s.tunnelListen.Close()
		s.logger.Debug("Tunnel listener closed: %v", s.tunnelListen.Addr())
	}
	if s.remoteListen != nil {
		s.remoteListen.Close()
		s.logger.Debug("Remote listener closed: %v", s.remoteListen.Addr())
	}
	if s.targetConn != nil {
		s.targetConn.Close()
		s.logger.Debug("Target connection closed: %v", s.targetConn.LocalAddr())
	}
	if s.tunnelConn != nil {
		s.tunnelConn.Close()
		s.logger.Debug("Tunnel connection closed: %v", s.tunnelConn.LocalAddr())
	}
	if s.remotePool != nil {
		s.remotePool.Close()
		s.logger.Debug("Remote connections closed")
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

func (s *Server) initListener() error {
	tunnelListen, err := tls.Listen("tcp", s.tunnelAddr.String(), s.tlsConfig)
	if err != nil {
		return err
	}
	s.tunnelListen = tunnelListen
	remoteListen, err := net.Listen("tcp", s.remoteAddr.String())
	if err != nil {
		return err
	}
	s.remoteListen = remoteListen
	targetListen, err := net.Listen("tcp", s.targetAddr.String())
	if err != nil {
		return err
	}
	s.targetListen = targetListen
	return nil
}

func (s *Server) getTunnelConnection() error {
	tunnelConn, err := s.tunnelListen.Accept()
	if err != nil {
		return err
	}
	s.tunnelConn = tunnelConn.(*tls.Conn)
	s.logger.Debug("Tunnel connection: %v <-> %v", s.tunnelConn.LocalAddr(), s.tunnelConn.RemoteAddr())
	remoteSignal := []byte(strconv.Itoa(s.remoteAddr.Port))
	s.sharedMU.Lock()
	_, err = s.tunnelConn.Write(remoteSignal)
	s.sharedMU.Unlock()
	if err != nil {
		return err
	}
	s.logger.Debug("Remote signal -> : %v", s.remoteAddr)
	return nil
}

func (s *Server) healthCheck() error {
	for {
		timer := time.NewTimer(ReportInterval)
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return s.ctx.Err()
		case <-timer.C:
			if !s.sharedMU.TryLock() {
				continue
			}
			_, err := s.tunnelConn.Write([]byte(ReportSignal))
			s.sharedMU.Unlock()
			if err != nil {
				return err
			}
		}
	}
}

func (s *Server) serverLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			targetConn, err := s.targetListen.Accept()
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
				s.sharedMU.Lock()
				_, err = s.tunnelConn.Write([]byte(LaunchSignal))
				s.sharedMU.Unlock()
				if err != nil {
					s.logger.Error("Write failed: %v", err)
					return
				}
				s.logger.Debug("Launch signal -> : %v", s.tunnelConn.RemoteAddr())
				id, remoteConn := s.remotePool.Get()
				if id == "" {
					s.logger.Error("Get failed: %v", remoteConn)
					return
				}
				s.logger.Debug("Remote connection: %v <- active %v", id, s.remotePool.Active())
				defer func() {
					if remoteConn != nil {
						remoteConn.Close()
					}
				}()
				s.remoteConn = remoteConn
				s.logger.Debug("Remote connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
				s.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
				_, _, err = io.DataExchange(remoteConn, targetConn)
				s.logger.Debug("Exchange complete: %v", err)
			}(targetConn)
		}
	}
}
