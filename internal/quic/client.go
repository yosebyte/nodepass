package quic

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	ntls "github.com/yosebyte/nodepass/internal/tls"
	"github.com/yosebyte/x/log"
)

// Client 表示QUIC客户端连接
type Client struct {
	logger     *log.Logger
	conn       quic.Connection
	stream     quic.Stream
	remoteAddr string
	tlsConfig  *tls.Config
}

// NewClient 创建一个新的QUIC客户端
func NewClient(remoteAddr string, tlsConfig *tls.Config, logger *log.Logger) *Client {
	// 确保使用TLS 1.3
	if tlsConfig != nil {
		tlsConfig = ntls.GetTLS13Config(tlsConfig)
	}
	
	return &Client{
		logger:     logger,
		remoteAddr: remoteAddr,
		tlsConfig:  tlsConfig,
	}
}

// Connect 连接到QUIC服务器
func (c *Client) Connect(ctx context.Context) error {
	// 配置QUIC连接
	quicConfig := &quic.Config{
		KeepAlivePeriod: 15 * time.Second,
		MaxIdleTimeout:  30 * time.Second,
	}

	// 建立QUIC连接
	conn, err := quic.DialAddr(ctx, c.remoteAddr, c.tlsConfig, quicConfig)
	if err != nil {
		return err
	}
	c.conn = conn
	c.logger.Debug("QUIC connection established: %v <-> %v", conn.LocalAddr(), conn.RemoteAddr())

	// 打开一个双向流
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(1, "failed to open stream")
		return err
	}
	c.stream = stream
	c.logger.Debug("QUIC stream opened: %v", stream.StreamID())

	return nil
}

// Read 从QUIC流中读取数据
func (c *Client) Read(p []byte) (int, error) {
	if c.stream == nil {
		return 0, io.ErrClosedPipe
	}
	return c.stream.Read(p)
}

// Write 向QUIC流写入数据
func (c *Client) Write(p []byte) (int, error) {
	if c.stream == nil {
		return 0, io.ErrClosedPipe
	}
	return c.stream.Write(p)
}

// Close 关闭QUIC连接
func (c *Client) Close() error {
	if c.stream != nil {
		c.stream.Close()
		c.stream = nil
	}
	if c.conn != nil {
		c.conn.CloseWithError(0, "normal closure")
		c.conn = nil
	}
	return nil
}

// LocalAddr 返回本地地址
func (c *Client) LocalAddr() net.Addr {
	if c.conn != nil {
		return c.conn.LocalAddr()
	}
	return nil
}

// RemoteAddr 返回远程地址
func (c *Client) RemoteAddr() net.Addr {
	if c.conn != nil {
		return c.conn.RemoteAddr()
	}
	return nil
}

// SetDeadline 设置读写超时
func (c *Client) SetDeadline(t time.Time) error {
	if c.stream == nil {
		return io.ErrClosedPipe
	}
	return c.stream.SetDeadline(t)
}

// SetReadDeadline 设置读取超时
func (c *Client) SetReadDeadline(t time.Time) error {
	if c.stream == nil {
		return io.ErrClosedPipe
	}
	return c.stream.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (c *Client) SetWriteDeadline(t time.Time) error {
	if c.stream == nil {
		return io.ErrClosedPipe
	}
	return c.stream.SetWriteDeadline(t)
}
