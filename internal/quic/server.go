package quic

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	ntls "github.com/yosebyte/nodepass/internal/tls"
	"github.com/yosebyte/x/log"
)

// Server 表示QUIC服务器
type Server struct {
	logger     *log.Logger
	listener   quic.Listener
	tlsConfig  *tls.Config
	listenAddr string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewServer 创建一个新的QUIC服务器
func NewServer(listenAddr string, tlsConfig *tls.Config, logger *log.Logger) *Server {
	// 确保使用TLS 1.3
	if tlsConfig != nil {
		tlsConfig = ntls.GetTLS13Config(tlsConfig)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Server{
		logger:     logger,
		listenAddr: listenAddr,
		tlsConfig:  tlsConfig,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动QUIC服务器
func (s *Server) Start() error {
	// 配置QUIC服务器
	quicConfig := &quic.Config{
		KeepAlivePeriod: 15 * time.Second,
		MaxIdleTimeout:  30 * time.Second,
	}

	// 创建QUIC监听器
	listener, err := quic.ListenAddr(s.listenAddr, s.tlsConfig, quicConfig)
	if err != nil {
		return err
	}
	s.listener = listener
	s.logger.Debug("QUIC server started on: %v", listener.Addr())

	// 启动接受连接的协程
	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// acceptLoop 接受新的QUIC连接
func (s *Server) acceptLoop() {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// 接受新连接
			conn, err := s.listener.Accept(s.ctx)
			if err != nil {
				if s.ctx.Err() != nil {
					// 服务器正在关闭
					return
				}
				s.logger.Error("Failed to accept QUIC connection: %v", err)
				continue
			}
			
			s.logger.Debug("QUIC connection accepted: %v <-> %v", conn.LocalAddr(), conn.RemoteAddr())
			
			// 为每个连接启动一个处理协程
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// handleConnection 处理QUIC连接
func (s *Server) handleConnection(conn quic.Connection) {
	defer s.wg.Done()
	defer conn.CloseWithError(0, "normal closure")
	
	// 接受流
	stream, err := conn.AcceptStream(s.ctx)
	if err != nil {
		s.logger.Error("Failed to accept QUIC stream: %v", err)
		return
	}
	defer stream.Close()
	
	s.logger.Debug("QUIC stream accepted: %v", stream.StreamID())
	
	// 这里可以处理流数据，例如转发到目标服务器
	// 在实际实现中，这里需要与nodepass的数据交换机制集成
}

// Stop 停止QUIC服务器
func (s *Server) Stop() error {
	s.cancel()
	if s.listener != nil {
		err := s.listener.Close()
		s.listener = nil
		s.wg.Wait()
		return err
	}
	return nil
}

// Addr 返回服务器监听地址
func (s *Server) Addr() net.Addr {
	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

// Connection 表示一个QUIC连接和流的组合，实现net.Conn接口
type Connection struct {
	conn   quic.Connection
	stream quic.Stream
}

// NewConnection 创建一个新的QUIC连接包装器
func NewConnection(conn quic.Connection, stream quic.Stream) *Connection {
	return &Connection{
		conn:   conn,
		stream: stream,
	}
}

// Read 从QUIC流中读取数据
func (c *Connection) Read(p []byte) (int, error) {
	if c.stream == nil {
		return 0, io.ErrClosedPipe
	}
	return c.stream.Read(p)
}

// Write 向QUIC流写入数据
func (c *Connection) Write(p []byte) (int, error) {
	if c.stream == nil {
		return 0, io.ErrClosedPipe
	}
	return c.stream.Write(p)
}

// Close 关闭QUIC连接
func (c *Connection) Close() error {
	var err error
	if c.stream != nil {
		err = c.stream.Close()
		c.stream = nil
	}
	if c.conn != nil {
		c.conn.CloseWithError(0, "normal closure")
		c.conn = nil
	}
	return err
}

// LocalAddr 返回本地地址
func (c *Connection) LocalAddr() net.Addr {
	if c.conn != nil {
		return c.conn.LocalAddr()
	}
	return nil
}

// RemoteAddr 返回远程地址
func (c *Connection) RemoteAddr() net.Addr {
	if c.conn != nil {
		return c.conn.RemoteAddr()
	}
	return nil
}

// SetDeadline 设置读写超时
func (c *Connection) SetDeadline(t time.Time) error {
	if c.stream == nil {
		return io.ErrClosedPipe
	}
	return c.stream.SetDeadline(t)
}

// SetReadDeadline 设置读取超时
func (c *Connection) SetReadDeadline(t time.Time) error {
	if c.stream == nil {
		return io.ErrClosedPipe
	}
	return c.stream.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (c *Connection) SetWriteDeadline(t time.Time) error {
	if c.stream == nil {
		return io.ErrClosedPipe
	}
	return c.stream.SetWriteDeadline(t)
}
