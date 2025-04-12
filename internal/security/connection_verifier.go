package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// ConnectionVerifier 用于验证连接的合法性
type ConnectionVerifier struct {
	mutex            sync.RWMutex
	verifiedConns    map[string]time.Time
	connectionTokens map[string]string
	expirationTime   time.Duration
}

// NewConnectionVerifier 创建一个新的连接验证器
func NewConnectionVerifier(expirationTime time.Duration) *ConnectionVerifier {
	return &ConnectionVerifier{
		verifiedConns:    make(map[string]time.Time),
		connectionTokens: make(map[string]string),
		expirationTime:   expirationTime,
	}
}

// GenerateConnectionToken 为连接生成一个唯一的令牌
func (cv *ConnectionVerifier) GenerateConnectionToken(conn net.Conn, secretKey string) string {
	// 使用连接地址和时间戳创建唯一标识
	connID := fmt.Sprintf("%s-%s-%d", conn.LocalAddr().String(), conn.RemoteAddr().String(), time.Now().UnixNano())
	
	// 使用HMAC生成令牌
	h := sha256.New()
	io.WriteString(h, connID)
	io.WriteString(h, secretKey)
	token := hex.EncodeToString(h.Sum(nil))
	
	// 存储令牌
	cv.mutex.Lock()
	defer cv.mutex.Unlock()
	cv.connectionTokens[token] = conn.RemoteAddr().String()
	
	return token
}

// VerifyConnectionToken 验证连接令牌的有效性
func (cv *ConnectionVerifier) VerifyConnectionToken(token string, conn net.Conn) error {
	cv.mutex.RLock()
	defer cv.mutex.RUnlock()
	
	expectedAddr, exists := cv.connectionTokens[token]
	if !exists {
		return errors.New("无效的连接令牌")
	}
	
	// 验证连接地址
	if conn.RemoteAddr().String() != expectedAddr {
		return errors.New("连接地址不匹配，可能是会话劫持")
	}
	
	return nil
}

// MarkConnectionVerified 标记连接为已验证
func (cv *ConnectionVerifier) MarkConnectionVerified(conn net.Conn) {
	cv.mutex.Lock()
	defer cv.mutex.Unlock()
	
	cv.verifiedConns[conn.RemoteAddr().String()] = time.Now()
}

// IsConnectionVerified 检查连接是否已验证
func (cv *ConnectionVerifier) IsConnectionVerified(conn net.Conn) bool {
	cv.mutex.RLock()
	defer cv.mutex.RUnlock()
	
	timestamp, exists := cv.verifiedConns[conn.RemoteAddr().String()]
	if !exists {
		return false
	}
	
	// 检查验证是否过期
	if time.Since(timestamp) > cv.expirationTime {
		return false
	}
	
	return true
}

// RemoveConnection 从验证器中移除连接
func (cv *ConnectionVerifier) RemoveConnection(conn net.Conn) {
	cv.mutex.Lock()
	defer cv.mutex.Unlock()
	
	delete(cv.verifiedConns, conn.RemoteAddr().String())
	
	// 移除与此连接相关的所有令牌
	for token, addr := range cv.connectionTokens {
		if addr == conn.RemoteAddr().String() {
			delete(cv.connectionTokens, token)
		}
	}
}

// CleanExpiredConnections 清理过期的连接
func (cv *ConnectionVerifier) CleanExpiredConnections() {
	cv.mutex.Lock()
	defer cv.mutex.Unlock()
	
	now := time.Now()
	for addr, timestamp := range cv.verifiedConns {
		if now.Sub(timestamp) > cv.expirationTime {
			delete(cv.verifiedConns, addr)
			
			// 移除与此连接相关的所有令牌
			for token, connAddr := range cv.connectionTokens {
				if connAddr == addr {
					delete(cv.connectionTokens, token)
				}
			}
		}
	}
}

// StartCleanupRoutine 启动定期清理过期连接的例程
func (cv *ConnectionVerifier) StartCleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			cv.CleanExpiredConnections()
		}
	}()
}
