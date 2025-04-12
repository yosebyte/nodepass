package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NonceManager 管理用于防止重放攻击的nonce
type NonceManager struct {
	usedNonces     map[string]time.Time
	mutex          sync.RWMutex
	expirationTime time.Duration
}

// NewNonceManager 创建一个新的NonceManager
func NewNonceManager(expirationTime time.Duration) *NonceManager {
	return &NonceManager{
		usedNonces:     make(map[string]time.Time),
		expirationTime: expirationTime,
	}
}

// GenerateNonce 生成一个新的随机nonce
func (nm *NonceManager) GenerateNonce() (string, error) {
	// 生成16字节的随机数
	nonceBytes := make([]byte, 16)
	_, err := rand.Read(nonceBytes)
	if err != nil {
		return "", err
	}
	
	// 转换为base64编码
	nonce := base64.StdEncoding.EncodeToString(nonceBytes)
	
	// 清理过期的nonce
	nm.cleanExpiredNonces()
	
	return nonce, nil
}

// VerifyNonce 验证nonce是否有效（未使用过）
func (nm *NonceManager) VerifyNonce(nonce string) error {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	
	// 检查nonce是否已使用
	if _, exists := nm.usedNonces[nonce]; exists {
		return errors.New("nonce已被使用，可能是重放攻击")
	}
	
	// 标记nonce为已使用
	nm.usedNonces[nonce] = time.Now()
	return nil
}

// cleanExpiredNonces 清理过期的nonce
func (nm *NonceManager) cleanExpiredNonces() {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	
	now := time.Now()
	for nonce, timestamp := range nm.usedNonces {
		if now.Sub(timestamp) > nm.expirationTime {
			delete(nm.usedNonces, nonce)
		}
	}
}

// SecureMessage 表示一个带有安全属性的消息
type SecureMessage struct {
	Timestamp int64  // 消息创建时间戳
	Nonce     string // 防重放攻击的nonce
	Data      string // 实际数据
	HMAC      string // 消息认证码
}

// CreateSecureMessage 创建一个新的安全消息
func CreateSecureMessage(data string, secretKey string, nonceManager *NonceManager) (*SecureMessage, error) {
	// 生成nonce
	nonce, err := nonceManager.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("生成nonce失败: %v", err)
	}
	
	// 创建消息
	message := &SecureMessage{
		Timestamp: time.Now().Unix(),
		Nonce:     nonce,
		Data:      data,
	}
	
	// 计算HMAC
	message.HMAC = calculateHMAC(message.getDataForHMAC(), secretKey)
	
	return message, nil
}

// VerifySecureMessage 验证安全消息的有效性
func VerifySecureMessage(messageStr string, secretKey string, nonceManager *NonceManager, maxAge time.Duration) (string, error) {
	// 解析消息
	parts := strings.Split(messageStr, "|")
	if len(parts) != 4 {
		return "", errors.New("消息格式无效")
	}
	
	// 解析时间戳
	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("解析时间戳失败: %v", err)
	}
	
	// 重建消息对象
	message := &SecureMessage{
		Timestamp: timestamp,
		Nonce:     parts[1],
		Data:      parts[2],
		HMAC:      parts[3],
	}
	
	// 验证时间戳（防止过期消息）
	messageTime := time.Unix(message.Timestamp, 0)
	if time.Since(messageTime) > maxAge {
		return "", errors.New("消息已过期")
	}
	
	// 验证nonce（防止重放攻击）
	if err := nonceManager.VerifyNonce(message.Nonce); err != nil {
		return "", err
	}
	
	// 验证HMAC（确保消息完整性和真实性）
	expectedHMAC := calculateHMAC(message.getDataForHMAC(), secretKey)
	if !hmac.Equal([]byte(message.HMAC), []byte(expectedHMAC)) {
		return "", errors.New("消息认证码无效，消息可能被篡改")
	}
	
	return message.Data, nil
}

// String 将安全消息转换为字符串
func (sm *SecureMessage) String() string {
	return fmt.Sprintf("%d|%s|%s|%s", sm.Timestamp, sm.Nonce, sm.Data, sm.HMAC)
}

// getDataForHMAC 获取用于计算HMAC的数据
func (sm *SecureMessage) getDataForHMAC() string {
	return fmt.Sprintf("%d|%s|%s", sm.Timestamp, sm.Nonce, sm.Data)
}

// calculateHMAC 计算HMAC
func calculateHMAC(data string, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateSecretKey 生成一个新的随机密钥
func GenerateSecretKey() (string, error) {
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(keyBytes), nil
}
