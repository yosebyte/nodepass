package test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/yosebyte/nodepass/internal/security"
	ntls "github.com/yosebyte/nodepass/internal/tls"
)

// TestCertificateFingerprint 测试证书指纹计算和验证功能
func TestCertificateFingerprint(t *testing.T) {
	// 生成自签名证书
	cert, key, err := generateSelfSignedCert("test.example.com")
	if err != nil {
		t.Fatalf("生成自签名证书失败: %v", err)
	}

	// 解析证书
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("解析证书失败: %v", err)
	}

	// 计算指纹
	fingerprint := ntls.CalculateCertificateFingerprint(x509Cert)
	if fingerprint == "" {
		t.Fatal("计算证书指纹失败")
	}

	// 添加到受信任列表
	ntls.AddPinnedCertificate(string(fingerprint), "测试证书")

	// 验证指纹
	err = ntls.VerifyCertificateFingerprint(x509Cert)
	if err != nil {
		t.Fatalf("验证证书指纹失败: %v", err)
	}

	// 验证未知指纹
	ntls.PinnedCertificates = make(map[ntls.CertificateFingerprint]string) // 清空受信任列表
	err = ntls.VerifyCertificateFingerprint(x509Cert)
	if err == nil {
		t.Fatal("验证未知证书指纹应该失败")
	}
}

// TestSecureHandshake 测试安全握手协议
func TestSecureHandshake(t *testing.T) {
	// 创建服务器和客户端
	serverDone := make(chan bool)
	clientDone := make(chan bool)
	errorChan := make(chan error, 2)

	// 启动服务器
	go func() {
		defer close(serverDone)
		
		// 创建监听器
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			errorChan <- fmt.Errorf("创建监听器失败: %v", err)
			return
		}
		defer listener.Close()
		
		// 通知客户端服务器地址
		clientAddr <- listener.Addr().String()
		
		// 接受连接
		conn, err := listener.Accept()
		if err != nil {
			errorChan <- fmt.Errorf("接受连接失败: %v", err)
			return
		}
		defer conn.Close()
		
		// 创建NonceManager
		nonceManager := security.NewNonceManager(30 * time.Minute)
		
		// 生成密钥
		secretKey, err := security.GenerateSecretKey()
		if err != nil {
			errorChan <- fmt.Errorf("生成密钥失败: %v", err)
			return
		}
		
		// 执行握手
		_, err = security.SecureHandshake(conn, true, nil, nonceManager, secretKey)
		if err != nil {
			errorChan <- fmt.Errorf("服务器握手失败: %v", err)
			return
		}
	}()

	// 启动客户端
	go func() {
		defer close(clientDone)
		
		// 获取服务器地址
		addr := <-clientAddr
		
		// 连接服务器
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			errorChan <- fmt.Errorf("连接服务器失败: %v", err)
			return
		}
		defer conn.Close()
		
		// 创建NonceManager
		nonceManager := security.NewNonceManager(30 * time.Minute)
		
		// 生成密钥
		secretKey, err := security.GenerateSecretKey()
		if err != nil {
			errorChan <- fmt.Errorf("生成密钥失败: %v", err)
			return
		}
		
		// 执行握手
		_, err = security.SecureHandshake(conn, false, nonceManager, secretKey)
		if err != nil {
			errorChan <- fmt.Errorf("客户端握手失败: %v", err)
			return
		}
	}()

	// 等待完成或超时
	select {
	case err := <-errorChan:
		t.Fatalf("握手测试失败: %v", err)
	case <-serverDone:
		t.Log("服务器完成握手")
	case <-clientDone:
		t.Log("客户端完成握手")
	case <-time.After(5 * time.Second):
		t.Fatal("握手测试超时")
	}
}

// TestAntiReplay 测试防重放攻击机制
func TestAntiReplay(t *testing.T) {
	// 创建NonceManager
	nonceManager := security.NewNonceManager(30 * time.Minute)
	
	// 生成密钥
	secretKey, err := security.GenerateSecretKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	
	// 创建安全消息
	message, err := security.CreateSecureMessage("test data", secretKey, nonceManager)
	if err != nil {
		t.Fatalf("创建安全消息失败: %v", err)
	}
	
	// 验证消息
	data, err := security.VerifySecureMessage(message.String(), secretKey, nonceManager, 30*time.Second)
	if err != nil {
		t.Fatalf("验证安全消息失败: %v", err)
	}
	if data != "test data" {
		t.Fatalf("消息数据不匹配: 期望 'test data', 实际 '%s'", data)
	}
	
	// 尝试重放攻击
	_, err = security.VerifySecureMessage(message.String(), secretKey, nonceManager, 30*time.Second)
	if err == nil {
		t.Fatal("重放攻击应该被检测到")
	}
	
	// 测试过期消息
	time.Sleep(2 * time.Second)
	expiredMessage, err := security.CreateSecureMessage("expired data", secretKey, nonceManager)
	if err != nil {
		t.Fatalf("创建过期消息失败: %v", err)
	}
	expiredMessage.Timestamp = time.Now().Unix() - 60 // 设置为1分钟前
	_, err = security.VerifySecureMessage(expiredMessage.String(), secretKey, nonceManager, 1*time.Second)
	if err == nil {
		t.Fatal("过期消息应该被拒绝")
	}
}

// TestConnectionVerifier 测试连接验证器
func TestConnectionVerifier(t *testing.T) {
	// 创建连接验证器
	verifier := security.NewConnectionVerifier(30 * time.Minute)
	
	// 创建模拟连接
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()
	
	// 生成连接令牌
	secretKey, _ := security.GenerateSecretKey()
	token := verifier.GenerateConnectionToken(conn1, secretKey)
	
	// 验证连接令牌
	err := verifier.VerifyConnectionToken(token, conn1)
	if err != nil {
		t.Fatalf("验证连接令牌失败: %v", err)
	}
	
	// 使用错误的连接验证令牌
	err = verifier.VerifyConnectionToken(token, conn2)
	if err == nil {
		t.Fatal("使用错误的连接验证令牌应该失败")
	}
	
	// 标记连接为已验证
	verifier.MarkConnectionVerified(conn1)
	
	// 检查连接是否已验证
	if !verifier.IsConnectionVerified(conn1) {
		t.Fatal("连接应该被标记为已验证")
	}
	
	// 检查未验证的连接
	if verifier.IsConnectionVerified(conn2) {
		t.Fatal("未验证的连接不应该被标记为已验证")
	}
}

// 辅助函数：生成自签名证书
func generateSelfSignedCert(commonName string) (tls.Certificate, []byte, error) {
	// 在实际实现中，这里应该生成自签名证书
	// 为简化测试，这里返回一个空证书
	return tls.Certificate{}, nil, nil
}

// 用于通信的通道
var clientAddr = make(chan string, 1)
