package websocket

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	ntls "github.com/yosebyte/nodepass/internal/tls"
	"github.com/yosebyte/x/log"
)

// Client 表示WebSocket客户端连接
type Client struct {
	logger     *log.Logger
	conn       *websocket.Conn
	remoteAddr string
	tlsConfig  *tls.Config
	mu         sync.Mutex
	closed     bool
}

// NewClient 创建一个新的WebSocket客户端
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

// Connect 连接到WebSocket服务器
func (c *Client) Connect() error {
	dialer := websocket.Dialer{
		TLSClientConfig: c.tlsConfig,
		HandshakeTimeout: 10 * time.Second,
	}
	
	// 确定协议
	protocol := "ws"
	if c.tlsConfig != nil {
		protocol = "wss"
	}
	
	// 建立WebSocket连接
	conn, _, err := dialer.Dial(protocol+"://"+c.remoteAddr, nil)
	if err != nil {
		return err
	}
	
	c.conn = conn
	c.logger.Debug("WebSocket connection established: %v", c.remoteAddr)
	return nil
}

// Read 从WebSocket连接中读取数据
func (c *Client) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return 0, net.ErrClosed
	}
	
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	
	n := copy(p, message)
	return n, nil
}

// Write 向WebSocket连接写入数据
func (c *Client) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return 0, net.ErrClosed
	}
	
	err := c.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	
	return len(p), nil
}

// Close 关闭WebSocket连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	
	c.closed = true
	if c.conn != nil {
		// 发送关闭消息
		err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			c.logger.Error("Error sending close message: %v", err)
		}
		
		return c.conn.Close()
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
	if c.conn != nil {
		err := c.conn.SetReadDeadline(t)
		if err != nil {
			return err
		}
		return c.conn.SetWriteDeadline(t)
	}
	return net.ErrClosed
}

// SetReadDeadline 设置读取超时
func (c *Client) SetReadDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetReadDeadline(t)
	}
	return net.ErrClosed
}

// SetWriteDeadline 设置写入超时
func (c *Client) SetWriteDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetWriteDeadline(t)
	}
	return net.ErrClosed
}
