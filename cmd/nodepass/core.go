package main

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/NodePassProject/cert"
	"github.com/NodePassProject/logs"
	"github.com/yosebyte/nodepass/internal"
)

// start 启动核心逻辑
func start(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("start: empty URL command")
	}

	parsedURL, err := url.Parse(args[1])
	if err != nil {
		return fmt.Errorf("start: parse URL failed: %w", err)
	}

	logger := initLogger(parsedURL.Query().Get("log"))

	core, err := createCore(parsedURL, logger)
	if err != nil {
		return fmt.Errorf("start: create core failed: %w", err)
	}

	core.Run()
	return nil
}

// initLogger 初始化日志记录器
func initLogger(level string) *logs.Logger {
	logger := logs.NewLogger(logs.Info, true)
	switch level {
	case "none":
		logger.SetLogLevel(logs.None)
	case "debug":
		logger.SetLogLevel(logs.Debug)
		logger.Debug("Init log level: DEBUG")
	case "warn":
		logger.SetLogLevel(logs.Warn)
		logger.Warn("Init log level: WARN")
	case "error":
		logger.SetLogLevel(logs.Error)
		logger.Error("Init log level: ERROR")
	case "event":
		logger.SetLogLevel(logs.Event)
		logger.Event("Init log level: EVENT")
	default:
	}
	return logger
}

// createCore 创建核心
func createCore(parsedURL *url.URL, logger *logs.Logger) (interface{ Run() }, error) {
	switch parsedURL.Scheme {
	case "server":
		tlsCode, tlsConfig := getTLSProtocol(parsedURL, logger)
		return internal.NewServer(parsedURL, tlsCode, tlsConfig, logger)
	case "client":
		return internal.NewClient(parsedURL, logger)
	case "master":
		tlsCode, tlsConfig := getTLSProtocol(parsedURL, logger)
		return internal.NewMaster(parsedURL, tlsCode, tlsConfig, logger, version)
	default:
		return nil, fmt.Errorf("createCore: unknown core: %v", parsedURL)
	}
}

// getTLSProtocol 获取TLS配置
func getTLSProtocol(parsedURL *url.URL, logger *logs.Logger) (string, *tls.Config) {
	// 生成基本TLS配置
	tlsConfig, err := cert.NewTLSConfig(version)
	if err != nil {
		logger.Error("Generate TLS config failed: %v", err)
		logger.Warn("TLS code-0: nil cert")
		return "0", nil
	}

	tlsConfig.MinVersion = tls.VersionTLS13

	switch parsedURL.Query().Get("tls") {
	case "1":
		// 使用内存自签证书
		logger.Info("TLS code-1: RAM cert with TLS 1.3")
		return "1", tlsConfig
	case "2":
		// 使用自定义证书
		crtFile, keyFile := parsedURL.Query().Get("crt"), parsedURL.Query().Get("key")
		cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
		if err != nil {
			logger.Error("Certificate load failed: %v", err)
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
						logger.Error("Certificate reload failed: %v", err)
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
		return "2", tlsConfig
	default:
		if parsedURL.Query().Get("quic") == "1" {
			// QUIC模式下不支持明文传输
			logger.Info("TLS code-1: RAM cert with TLS 1.3 for QUIC")
			return "1", tlsConfig
		}
		// 不使用加密
		logger.Warn("TLS code-0: unencrypted")
		return "0", nil
	}
}

// exit 退出程序并显示帮助信息
func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v [%d]\n", err, os.Getpid())
	}
	fmt.Printf(`
╭─────────────────────────────────────╮
│ ░░█▀█░█▀█░░▀█░█▀▀░█▀█░█▀█░█▀▀░█▀▀░░ │
│ ░░█░█░█░█░█▀█░█▀▀░█▀▀░█▀█░▀▀█░▀▀█░░ │
│ ░░▀░▀░▀▀▀░▀▀▀░▀▀▀░▀░░░▀░▀░▀▀▀░▀▀▀░░ │
├─────────────────────────────────────┤
│%*s │
│%*s │
├─────────────────────────────────────┤
│ server://password@host/host?<query> │
│ client://password@host/host?<query> │
│ master://hostname:port/path?<query> │
╰─────────────────────────────────────╯

`, 36, version, 36, fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	os.Exit(1)
}
