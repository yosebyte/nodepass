package quic

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/yosebyte/x/log"
)

// Pool 表示QUIC连接池
type Pool struct {
	logger       *log.Logger
	connections  map[string]*Connection
	mutex        sync.RWMutex
	capacity     int
	tlsConfig    *tls.Config
	serverAddr   string
	dialFunc     func(context.Context) (quic.Connection, error)
	isServerPool bool
}

// NewClientPool 创建一个新的QUIC客户端连接池
func NewClientPool(minCapacity, maxCapacity int, tlsCode, serverAddr string, logger *log.Logger, tlsConfig *tls.Config) *Pool {
	pool := &Pool{
		logger:       logger,
		connections:  make(map[string]*Connection),
		capacity:     maxCapacity,
		tlsConfig:    tlsConfig,
		serverAddr:   serverAddr,
		isServerPool: false,
	}

	// 设置拨号函数
	pool.dialFunc = func(ctx context.Context) (quic.Connection, error) {
		quicConfig := &quic.Config{
			KeepAlivePeriod: 15 * time.Second,
			MaxIdleTimeout:  30 * time.Second,
		}
		return quic.DialAddr(ctx, serverAddr, tlsConfig, quicConfig)
	}

	// 预先创建最小容量的连接
	for i := 0; i < minCapacity; i++ {
		pool.createConnection()
	}

	return pool
}

// NewServerPool 创建一个新的QUIC服务器连接池
func NewServerPool(maxCapacity int, tlsConfig *tls.Config, listener quic.Listener, logger *log.Logger) *Pool {
	return &Pool{
		logger:       logger,
		connections:  make(map[string]*Connection),
		capacity:     maxCapacity,
		tlsConfig:    tlsConfig,
		isServerPool: true,
	}
}

// createConnection 创建一个新的QUIC连接
func (p *Pool) createConnection() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := p.dialFunc(ctx)
	if err != nil {
		p.logger.Error("Failed to create QUIC connection: %v", err)
		return ""
	}

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(1, "failed to open stream")
		p.logger.Error("Failed to open QUIC stream: %v", err)
		return ""
	}

	id := conn.RemoteAddr().String()
	connection := NewConnection(conn, stream)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.connections[id] = connection
	p.logger.Debug("QUIC connection created: %v", id)
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
func (p *Pool) AddConnection(conn quic.Connection, stream quic.Stream) {
	id := conn.RemoteAddr().String()
	connection := NewConnection(conn, stream)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.connections) < p.capacity {
		p.connections[id] = connection
		p.logger.Debug("QUIC connection added to pool: %v", id)
	} else {
		connection.Close()
		p.logger.Debug("QUIC connection rejected (pool full): %v", id)
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
func (p *Pool) ServerManager() {
	// 服务器连接池不需要主动创建连接
	// 它们是由客户端连接创建的
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
