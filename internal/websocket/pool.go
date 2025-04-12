package websocket

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yosebyte/x/log"
)

// Pool 表示WebSocket连接池
type Pool struct {
	logger       *log.Logger
	connections  map[string]*Connection
	mutex        sync.RWMutex
	capacity     int
	tlsConfig    *tls.Config
	serverAddr   string
	isServerPool bool
}

// NewClientPool 创建一个新的WebSocket客户端连接池
func NewClientPool(minCapacity, maxCapacity int, serverAddr string, tlsConfig *tls.Config, logger *log.Logger) *Pool {
	pool := &Pool{
		logger:       logger,
		connections:  make(map[string]*Connection),
		capacity:     maxCapacity,
		tlsConfig:    tlsConfig,
		serverAddr:   serverAddr,
		isServerPool: false,
	}

	// 预先创建最小容量的连接
	for i := 0; i < minCapacity; i++ {
		pool.createConnection()
	}

	return pool
}

// NewServerPool 创建一个新的WebSocket服务器连接池
func NewServerPool(maxCapacity int, server *Server, logger *log.Logger) *Pool {
	return &Pool{
		logger:       logger,
		connections:  make(map[string]*Connection),
		capacity:     maxCapacity,
		isServerPool: true,
	}
}

// createConnection 创建一个新的WebSocket客户端连接
func (p *Pool) createConnection() string {
	client := NewClient(p.serverAddr, p.tlsConfig, p.logger)
	err := client.Connect()
	if err != nil {
		p.logger.Error("Failed to create WebSocket connection: %v", err)
		return ""
	}

	id := client.RemoteAddr().String()
	connection := &Connection{
		conn:   client.conn,
		closed: false,
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.connections[id] = connection
	p.logger.Debug("WebSocket connection created: %v", id)
	return id
}

// ClientGet 从连接池获取一个客户端连接
func (p *Pool) ClientGet(id string) net.Conn {
	p.mutex.RLock()
	conn, exists := p.connections[id]
	p.mutex.RUnlock()

	if exists {
		p.mutex.Lock()
		delete(p.connections, id)
		p.mutex.Unlock()
		return conn
	}

	return nil
}

// ServerGet 从连接池获取一个服务器连接
func (p *Pool) ServerGet() (string, net.Conn) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 找到第一个可用连接
	for id, conn := range p.connections {
		delete(p.connections, id)
		return id, conn
	}

	return "", nil
}

// Put 将连接放回池中
func (p *Pool) Put(id string, conn *Connection) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.connections) < p.capacity {
		p.connections[id] = conn
	} else {
		conn.Close()
	}
}

// AddConnection 添加一个连接到池中
func (p *Pool) AddConnection(conn *websocket.Conn) {
	id := conn.RemoteAddr().String()
	connection := NewConnection(conn)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.connections) < p.capacity {
		p.connections[id] = connection
		p.logger.Debug("WebSocket connection added to pool: %v", id)
	} else {
		connection.Close()
		p.logger.Debug("WebSocket connection rejected (pool full): %v", id)
	}
}

// ClientManager 管理客户端连接池
func (p *Pool) ClientManager() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.mutex.RLock()
		currentSize := len(p.connections)
		p.mutex.RUnlock()

		// 如果连接数低于容量的一半，创建新连接
		if currentSize < p.capacity/2 {
			p.createConnection()
		}
	}
}

// ServerManager 管理服务器连接池
func (p *Pool) ServerManager(server *Server) {
	for {
		conn := server.AcceptConn()
		if conn == nil {
			// 服务器已关闭
			return
		}
		
		p.AddConnection(conn)
	}
}

// Active 返回活动连接数
func (p *Pool) Active() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.connections)
}

// Capacity 返回连接池容量
func (p *Pool) Capacity() int {
	return p.capacity
}

// Ready 检查连接池是否准备好
func (p *Pool) Ready() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.connections) > 0
}

// Flush 清空连接池
func (p *Pool) Flush() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for id, conn := range p.connections {
		conn.Close()
		delete(p.connections, id)
	}
}

// Close 关闭连接池
func (p *Pool) Close() {
	p.Flush()
}
