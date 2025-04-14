package main

import (
	"crypto/tls"
	"net/url"
	"time"

	"github.com/yosebyte/nodepass/internal"
	x "github.com/yosebyte/x/tls"
)

func coreDispatch(parsedURL *url.URL) {
	switch parsedURL.Scheme {
	case "server":
		runServer(parsedURL)
	case "client":
		runClient(parsedURL)
	case "master":
		runMaster(parsedURL)
	default:
		logger.Fatal("Unknown core: %v", parsedURL.Scheme)
		getExitInfo()
	}
}

func runServer(parsedURL *url.URL) {
	tlsCode, tlsConfig := getTLSProtocol(parsedURL)
	server := internal.NewServer(parsedURL, tlsCode, tlsConfig, logger)
	server.Manage()
}

func runClient(parsedURL *url.URL) {
	client := internal.NewClient(parsedURL, logger)
	client.Manage()
}

func runMaster(parsedURL *url.URL) {
	tlsCode, tlsConfig := getTLSProtocol(parsedURL)
	master := internal.NewMaster(parsedURL, tlsCode, tlsConfig, logger)
	master.Manage()
}

func getTLSProtocol(parsedURL *url.URL) (string, *tls.Config) {
	tlsConfig, err := x.GenerateTLSConfig("yosebyte/nodepass:" + version)
	if err != nil {
		logger.Error("Generate failed: %v", err)
		logger.Warn("TLS code-0: nil cert")
		return "0", nil
	}
	
	tlsCode := parsedURL.Query().Get("tls")
	switch tlsCode {
	case "0":
		logger.Info("TLS code-0: unencrypted")
		return tlsCode, nil
	case "1":
		tlsConfig.MinVersion = tls.VersionTLS13
		logger.Info("TLS code-1: RAM cert with TLS 1.3")
		return tlsCode, tlsConfig
	case "2":
		crtFile, keyFile := parsedURL.Query().Get("crt"), parsedURL.Query().Get("key")
		cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
		if err != nil {
			logger.Error("Cert load failed: %v", err)
			tlsConfig.MinVersion = tls.VersionTLS13
			logger.Warn("TLS code-1: RAM cert with TLS 1.3")
			return "1", tlsConfig
		}
		cachedCert := cert
		lastReload := time.Now()
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS13,
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
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
			logger.Warn("TLS code-2: unknown with TLS 1.3")
		}
		return tlsCode, tlsConfig
	default:
		logger.Warn("TLS code-0: unencrypted")
		return "0", nil
	}
}
