package main

import (
	"crypto/tls"
	"net/url"
	"time"

	"github.com/NodePassProject/cert"
	"github.com/yosebyte/nodepass/internal"
)

// coreDispatch 根据URL方案分派到不同的运行模式
func coreDispatch(parsedURL *url.URL) {
	switch parsedURL.Scheme {
	case "server":
		runServer(parsedURL)
	case "client":
		runClient(parsedURL)
	case "master":
		runMaster(parsedURL)
	case "worker":
		getExitInfo() // TODO
	default:
		logger.Fatal("Unknown core: %v", parsedURL.Scheme)
		getExitInfo()
	}
}

// runServer 运行服务端模式
func runServer(parsedURL *url.URL) {
	tlsCode, tlsConfig := getTLSProtocol(parsedURL)
	server := internal.NewServer(parsedURL, tlsCode, tlsConfig, logger)
	server.Manage()
}

// runClient 运行客户端模式
func runClient(parsedURL *url.URL) {
	client := internal.NewClient(parsedURL, logger)
	client.Manage()
}

// runMaster 运行主控模式
func runMaster(parsedURL *url.URL) {
	tlsCode, tlsConfig := getTLSProtocol(parsedURL)
	master := internal.NewMaster(parsedURL, tlsCode, tlsConfig, logger)
	master.Manage()
}

// getTLSProtocol 获取TLS配置
func getTLSProtocol(parsedURL *url.URL) (string, *tls.Config) {
	// 生成基本TLS配置
	tlsConfig, err := cert.NewTLSConfig("yosebyte/nodepass:" + version)
	if err != nil {
		logger.Error("Generate failed: %v", err)
		logger.Warn("TLS code-0: nil cert")
		return "0", nil
	}

	tlsConfig.MinVersion = tls.VersionTLS13
	tlsCode := parsedURL.Query().Get("tls")

	switch tlsCode {
	case "0":
		// 不使用加密
		logger.Info("TLS code-0: unencrypted")
		return tlsCode, nil

	case "1":
		// 使用内存中的证书
		logger.Info("TLS code-1: RAM cert with TLS 1.3")
		return tlsCode, tlsConfig

	case "2":
		// 使用自定义证书
		crtFile, keyFile := parsedURL.Query().Get("crt"), parsedURL.Query().Get("key")
		cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
		if err != nil {
			logger.Error("Cert load failed: %v", err)
			logger.Warn("TLS code-1: RAM cert with TLS 1.3")
			return "1", tlsConfig
		}

		// 缓存证书并设置自动重载
		cachedCert := cert
		lastReload := time.Now()
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS13,
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				// 定期重载证书
				if time.Since(lastReload) >= internal.ReloadInterval {
					newCert, err := tls.LoadX509KeyPair(crtFile, keyFile)
					if err != nil {
						logger.Error("Cert reload failed: %v", err)
					} else {
						logger.Debug("TLS cert reloaded: %v", crtFile)
						cachedCert = newCert
					}
					lastReload = time.Now()
				}
				return &cachedCert, nil
			},
		}

		if cert.Leaf != nil {
			logger.Info("TLS code-2: %v with TLS 1.3", cert.Leaf.Subject.CommonName)
		} else {
			logger.Warn("TLS code-2: unknown cert name with TLS 1.3")
		}
		return tlsCode, tlsConfig

	default:
		// 默认不使用加密
		logger.Warn("TLS code-0: unencrypted")
		return "0", nil
	}
}
