package websocket

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	ntls "github.com/yosebyte/nodepass/internal/tls"
	"github.com/yosebyte/x/log"
)

// Server 表示WebSocket服务器
type Server struct {
	logger     *log.Logger
	upgrader   websocket.Upgrader
	tlsConfig  *tls.Config
	listenAddr string
	httpServer *http.Server
	mu         sync.Mutex
	conns      map[*websocket.Conn]bool
	connChan   chan *websocket.Conn
}

// NewServer 创建一个新的WebSocket服务器
func NewServer(listenAddr string, tlsConfig *tls.Config, logger *log.Logger) *Server {
	// 确保使用TLS 1.3
	if tlsConfig != nil {
		tlsConfig = ntls.GetTLS13Config(tlsConfig)
	}
	
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			// 允许所有来源的WebSocket连接
			return true
		},
	}
	
	return &Server{
		logger:     logger,
		upgrader:   upgrader,
		tlsConfig:  tlsConfig,
		listenAddr: listenAddr,
		conns:      make(map[*websocket.Conn]bool),
		connChan:   make(chan *websocket.Conn, 100),
	}
}

// Start 启动WebSocket服务器
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebSocket)
	
	s.httpServer = &http.Server{
		Addr:      s.listenAddr,
		Handler:   mux,
		TLSConfig: s.tlsConfig,
	}
	
	// 根据是否有TLS配置决定启动方式
	var err error
	if s.tlsConfig != nil {
		s.logger.Info("Starting WebSocket server with TLS on %s", s.listenAddr)
		// 使用自签名证书
		err = s.httpServer.ListenAndServeTLS("", "")
	} else {
		s.logger.Info("Starting WebSocket server on %s", s.listenAddr)
		err = s.httpServer.ListenAndServe()
	}
	
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	
	return nil
}

// handleWebSocket 处理WebSocket连接请求
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection: %v", err)
		return
	}
	
	s.logger.Debug("WebSocket connection established: %v <-> %v", conn.LocalAddr(), conn.RemoteAddr())
	
	s.mu.Lock()
	s.conns[conn] = true
	s.mu.Unlock()
	
	// 将连接发送到通道，以便连接池使用
	select {
	case s.connChan <- conn:
		s.logger.Debug("WebSocket connection added to pool: %v", conn.RemoteAddr())
	default:
		s.logger.Debug("WebSocket connection channel full, closing connection: %v", conn.RemoteAddr())
		conn.Close()
	}
}

// AcceptConn 接受一个WebSocket连接
func (s *Server) AcceptConn() *websocket.Conn {
	return <-s.connChan
}

// Stop 停止WebSocket服务器
func (s *Server) Stop() error {
	if s.httpServer != nil {
		// 创建一个5秒超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// 关闭HTTP服务器
		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			s.logger.Error("Error shutting down WebSocket server: %v", err)
		}
		
		// 关闭所有WebSocket连接
		s.mu.Lock()
		for conn := range s.conns {
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
			delete(s.conns, conn)
		}
		s.mu.Unlock()
		
		close(s.connChan)
		s.logger.Info("WebSocket server stopped")
	}
	
	return nil
}

// Connection 表示一个WebSocket连接，实现net.Conn接口
type Connection struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
	readBuf []byte
}

// NewConnection 创建一个新的WebSocket连接包装器
func NewConnection(conn *websocket.Conn) *Connection {
	return &Connection{
		conn:   conn,
		closed: false,
	}
}

// Read 从WebSocket连接中读取数据
func (c *Connection) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return 0, net.ErrClosed
	}
	
	// 如果缓冲区中有数据，先从缓冲区读取
	if len(c.readBuf) > 0 {
		n := copy(p, c.readBuf)
		c.readBuf = c.readBuf[n:]
		return n, nil
	}
	
	// 否则从WebSocket读取新消息
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	
	// 复制数据到目标缓冲区
	n := copy(p, message)
	
	// 如果消息太大，存储剩余部分
	if n < len(message) {
		c.readBuf = message[n:]
	}
	
	return n, nil
}

// Write 向WebSocket连接写入数据
func (c *Connection) Write(p []byte) (int, error) {
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
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	
	c.closed = true
	if c.conn != nil {
		// 发送关闭消息
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return c.conn.Close()
	}
	
	return nil
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
func (c *Connection) SetReadDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetReadDeadline(t)
	}
	return net.ErrClosed
}

// SetWriteDeadline 设置写入超时
func (c *Connection) SetWriteDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetWriteDeadline(t)
	}
	return net.ErrClosed
}
