package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"time"

	"github.com/yosebyte/nodepass/internal/security"
	ntls "github.com/yosebyte/nodepass/internal/tls"
	"github.com/yosebyte/x/log"
)

// SecurityManager 管理所有安全相关功能
type SecurityManager struct {
	logger            *log.Logger
	nonceManager      *security.NonceManager
	connectionVerifier *security.ConnectionVerifier
	secretKey         string
	tlsConfig         *tls.Config
}

// NewSecurityManager 创建一个新的安全管理器
func NewSecurityManager(logger *log.Logger, tlsConfig *tls.Config) (*SecurityManager, error) {
	// 生成安全密钥
	secretKey, err := security.GenerateSecretKey()
	if err != nil {
		return nil, err
	}

	// 创建安全管理器
	return &SecurityManager{
		logger:            logger,
		nonceManager:      security.NewNonceManager(30 * time.Minute),
		connectionVerifier: security.NewConnectionVerifier(1 * time.Hour),
		secretKey:         secretKey,
		tlsConfig:         tlsConfig,
	}, nil
}

// SecureHandshake 执行安全握手
func (sm *SecurityManager) SecureHandshake(conn net.Conn, isServer bool) (*security.HandshakeData, error) {
	// 使用安全握手协议
	handshakeData, err := security.SecureHandshake(conn, isServer, sm.tlsConfig, sm.nonceManager, sm.secretKey)
	if err != nil {
		sm.logger.Error("安全握手失败: %v", err)
		return nil, err
	}

	// 标记连接为已验证
	sm.connectionVerifier.MarkConnectionVerified(conn)
	
	// 启动定期清理
	sm.connectionVerifier.StartCleanupRoutine(10 * time.Minute)
	
	return handshakeData, nil
}

// CreateSecureMessage 创建安全消息
func (sm *SecurityManager) CreateSecureMessage(data string) (string, error) {
	secureMsg, err := security.CreateSecureMessage(data, sm.secretKey, sm.nonceManager)
	if err != nil {
		return "", err
	}
	return secureMsg.String(), nil
}

// VerifySecureMessage 验证安全消息
func (sm *SecurityManager) VerifySecureMessage(messageStr string) (string, error) {
	return security.VerifySecureMessage(messageStr, sm.secretKey, sm.nonceManager, 30*time.Second)
}

// VerifyConnection 验证连接
func (sm *SecurityManager) VerifyConnection(conn net.Conn, token string) error {
	return sm.connectionVerifier.VerifyConnectionToken(token, conn)
}

// IsConnectionVerified 检查连接是否已验证
func (sm *SecurityManager) IsConnectionVerified(conn net.Conn) bool {
	return sm.connectionVerifier.IsConnectionVerified(conn)
}

// GenerateConnectionToken 为连接生成令牌
func (sm *SecurityManager) GenerateConnectionToken(conn net.Conn) string {
	return sm.connectionVerifier.GenerateConnectionToken(conn, sm.secretKey)
}

// GetSecureTLSConfig 获取安全的TLS配置
func (sm *SecurityManager) GetSecureTLSConfig() *tls.Config {
	if sm.tlsConfig == nil {
		return nil
	}
	return ntls.GetSecureTLS13Config(sm.tlsConfig)
}

// LoadTrustedCertificates 加载受信任的证书
func (sm *SecurityManager) LoadTrustedCertificates() error {
	// 这里应该从配置文件或其他来源加载受信任的证书指纹
	// 为了示例，我们添加一些假的证书指纹
	ntls.AddPinnedCertificate("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "示例证书1")
	ntls.AddPinnedCertificate("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "示例证书2")
	
	return nil
}
