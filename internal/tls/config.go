package tls

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// GetTLS13Config 返回一个强制使用TLS 1.3的配置
func GetTLS13Config(baseConfig *tls.Config) *tls.Config {
	if baseConfig == nil {
		baseConfig = &tls.Config{}
	}
	
	// 强制使用TLS 1.3
	baseConfig.MinVersion = tls.VersionTLS13
	baseConfig.MaxVersion = tls.VersionTLS13
	
	// 仅支持TLS 1.3的密码套件
	baseConfig.CipherSuites = []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	}
	
	return baseConfig
}

// CertificateFingerprint 表示证书的SHA-256指纹
type CertificateFingerprint string

// PinnedCertificates 存储受信任的证书指纹
var PinnedCertificates = make(map[CertificateFingerprint]string)

// AddPinnedCertificate 添加一个受信任的证书指纹
func AddPinnedCertificate(fingerprint string, description string) {
	PinnedCertificates[CertificateFingerprint(strings.ToLower(fingerprint))] = description
}

// CalculateCertificateFingerprint 计算证书的SHA-256指纹
func CalculateCertificateFingerprint(cert *x509.Certificate) CertificateFingerprint {
	if cert == nil {
		return ""
	}
	
	digest := sha256.Sum256(cert.Raw)
	return CertificateFingerprint(hex.EncodeToString(digest[:]))
}

// VerifyCertificateFingerprint 验证证书指纹是否在受信任列表中
func VerifyCertificateFingerprint(cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("证书为空")
	}
	
	fingerprint := CalculateCertificateFingerprint(cert)
	if _, ok := PinnedCertificates[fingerprint]; !ok {
		return fmt.Errorf("证书指纹不受信任: %s", fingerprint)
	}
	
	return nil
}

// GetSecureTLS13Config 返回一个带证书固定的TLS 1.3配置
func GetSecureTLS13Config(baseConfig *tls.Config) *tls.Config {
	config := GetTLS13Config(baseConfig)
	
	// 保存原始验证函数
	originalVerifyPeerCertificate := config.VerifyPeerCertificate
	
	// 添加证书固定验证
	config.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		// 首先执行原始验证（如果有）
		if originalVerifyPeerCertificate != nil {
			if err := originalVerifyPeerCertificate(rawCerts, verifiedChains); err != nil {
				return err
			}
		}
		
		// 如果没有验证链（可能是因为InsecureSkipVerify=true），则解析证书
		if len(verifiedChains) == 0 && len(rawCerts) > 0 {
			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("解析证书失败: %v", err)
			}
			
			// 验证证书指纹
			return VerifyCertificateFingerprint(cert)
		}
		
		// 验证所有验证链中的叶证书
		for _, chain := range verifiedChains {
			if len(chain) > 0 {
				// 验证叶证书的指纹
				if err := VerifyCertificateFingerprint(chain[0]); err != nil {
					return err
				}
			}
		}
		
		return nil
	}
	
	return config
}

// LoadPinnedCertificatesFromFile 从文件加载受信任的证书指纹
func LoadPinnedCertificatesFromFile(filename string) error {
	// 实际实现中，这里应该从文件读取证书指纹
	// 为简化示例，这里直接返回nil
	return nil
}

// SavePinnedCertificatesToFile 将受信任的证书指纹保存到文件
func SavePinnedCertificatesToFile(filename string) error {
	// 实际实现中，这里应该将证书指纹保存到文件
	// 为简化示例，这里直接返回nil
	return nil
}
