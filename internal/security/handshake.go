package security

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	ntls "github.com/yosebyte/nodepass/internal/tls"
)

// HandshakeData 表示握手过程中交换的数据
type HandshakeData struct {
	ServerName      string   `json:"server_name"`      // 服务器名称
	Timestamp       int64    `json:"timestamp"`        // 时间戳
	Nonce           string   `json:"nonce"`            // 防重放攻击的nonce
	TLSMode         string   `json:"tls_mode"`         // TLS模式
	Port            int      `json:"port"`             // 端口
	SupportedProtos []string `json:"supported_protos"` // 支持的协议
	CertFingerprint string   `json:"cert_fingerprint"` // 证书指纹
	Signature       string   `json:"signature"`        // 数据签名
}

// SecureHandshake 执行安全握手
func SecureHandshake(conn net.Conn, isServer bool, tlsConfig *tls.Config, nonceManager *NonceManager, secretKey string) (*HandshakeData, error) {
	if isServer {
		return serverHandshake(conn, tlsConfig, nonceManager, secretKey)
	}
	return clientHandshake(conn, nonceManager, secretKey)
}

// serverHandshake 服务器端握手
func serverHandshake(conn net.Conn, tlsConfig *tls.Config, nonceManager *NonceManager, secretKey string) (*HandshakeData, error) {
	// 1. 接收客户端初始消息
	clientData := make([]byte, 1024)
	n, err := conn.Read(clientData)
	if err != nil {
		return nil, fmt.Errorf("读取客户端数据失败: %v", err)
	}

	// 2. 解析客户端消息
	clientMsg, err := parseSecureMessage(string(clientData[:n]), secretKey, nonceManager, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("解析客户端消息失败: %v", err)
	}

	var clientHandshake HandshakeData
	if err := json.Unmarshal([]byte(clientMsg), &clientHandshake); err != nil {
		return nil, fmt.Errorf("解析客户端握手数据失败: %v", err)
	}

	// 3. 准备服务器响应
	nonce, err := nonceManager.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("生成nonce失败: %v", err)
	}

	// 获取证书指纹
	var certFingerprint string
	if tlsConfig != nil && tlsConfig.Certificates != nil && len(tlsConfig.Certificates) > 0 {
		cert, err := x509.ParseCertificate(tlsConfig.Certificates[0].Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("解析证书失败: %v", err)
		}
		certFingerprint = string(ntls.CalculateCertificateFingerprint(cert))
	}

	// 创建服务器握手数据
	serverHandshake := HandshakeData{
		ServerName:      "nodepass-server",
		Timestamp:       time.Now().Unix(),
		Nonce:           nonce,
		TLSMode:         clientHandshake.TLSMode, // 使用客户端请求的TLS模式
		Port:            clientHandshake.Port,    // 使用客户端请求的端口
		SupportedProtos: []string{"tcp", "udp", "quic", "websocket"},
		CertFingerprint: certFingerprint,
	}

	// 4. 签名服务器数据
	serverHandshakeBytes, err := json.Marshal(serverHandshake)
	if err != nil {
		return nil, fmt.Errorf("序列化服务器握手数据失败: %v", err)
	}

	// 创建安全消息
	secureMsg, err := CreateSecureMessage(string(serverHandshakeBytes), secretKey, nonceManager)
	if err != nil {
		return nil, fmt.Errorf("创建安全消息失败: %v", err)
	}

	// 5. 发送服务器响应
	_, err = conn.Write([]byte(secureMsg.String() + "\n"))
	if err != nil {
		return nil, fmt.Errorf("发送服务器响应失败: %v", err)
	}

	return &serverHandshake, nil
}

// clientHandshake 客户端握手
func clientHandshake(conn net.Conn, nonceManager *NonceManager, secretKey string) (*HandshakeData, error) {
	// 1. 准备客户端请求
	nonce, err := nonceManager.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("生成nonce失败: %v", err)
	}

	// 创建客户端握手数据
	clientHandshake := HandshakeData{
		ServerName:      "",                                  // 将由服务器填充
		Timestamp:       time.Now().Unix(),
		Nonce:           nonce,
		TLSMode:         "1",                                 // 默认使用TLS模式1
		Port:            0,                                   // 将由服务器分配
		SupportedProtos: []string{"tcp", "udp", "websocket"}, // 客户端支持的协议
		CertFingerprint: "",                                  // 将由服务器填充
	}

	// 2. 序列化并签名客户端数据
	clientHandshakeBytes, err := json.Marshal(clientHandshake)
	if err != nil {
		return nil, fmt.Errorf("序列化客户端握手数据失败: %v", err)
	}

	// 创建安全消息
	secureMsg, err := CreateSecureMessage(string(clientHandshakeBytes), secretKey, nonceManager)
	if err != nil {
		return nil, fmt.Errorf("创建安全消息失败: %v", err)
	}

	// 3. 发送客户端请求
	_, err = conn.Write([]byte(secureMsg.String() + "\n"))
	if err != nil {
		return nil, fmt.Errorf("发送客户端请求失败: %v", err)
	}

	// 4. 接收服务器响应
	serverData := make([]byte, 1024)
	n, err := conn.Read(serverData)
	if err != nil {
		return nil, fmt.Errorf("读取服务器数据失败: %v", err)
	}

	// 5. 解析服务器响应
	serverMsg, err := parseSecureMessage(string(serverData[:n]), secretKey, nonceManager, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("解析服务器消息失败: %v", err)
	}

	var serverHandshake HandshakeData
	if err := json.Unmarshal([]byte(serverMsg), &serverHandshake); err != nil {
		return nil, fmt.Errorf("解析服务器握手数据失败: %v", err)
	}

	// 6. 验证服务器证书指纹（如果有）
	if serverHandshake.CertFingerprint != "" {
		// 在实际实现中，这里应该检查证书指纹是否在受信任列表中
		if _, ok := ntls.PinnedCertificates[ntls.CertificateFingerprint(serverHandshake.CertFingerprint)]; !ok {
			return nil, errors.New("服务器证书指纹不受信任")
		}
	}

	return &serverHandshake, nil
}

// parseSecureMessage 解析安全消息
func parseSecureMessage(messageStr string, secretKey string, nonceManager *NonceManager, maxAge time.Duration) (string, error) {
	// 去除可能的换行符
	messageStr = strings.TrimSpace(messageStr)
	return VerifySecureMessage(messageStr, secretKey, nonceManager, maxAge)
}

// VerifyHandshakeData 验证握手数据的有效性
func VerifyHandshakeData(data *HandshakeData, maxAge time.Duration) error {
	// 验证时间戳
	messageTime := time.Unix(data.Timestamp, 0)
	if time.Since(messageTime) > maxAge {
		return errors.New("握手数据已过期")
	}

	// 在实际实现中，这里应该有更多的验证逻辑
	return nil
}
