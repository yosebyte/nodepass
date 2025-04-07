package main

import (
	"context"
	"crypto/tls"
	"net/url"
	"os"
	"time"

	"github.com/yosebyte/nodepass/internal"
	x "github.com/yosebyte/x/tls"
)

func coreDispatch(parsedURL *url.URL, signalChan chan os.Signal) {
	switch parsedURL.Scheme {
	case "server":
		runServer(parsedURL, signalChan)
	case "client":
		runClient(parsedURL, signalChan)
	default:
		logger.Fatal("Unknown core: %v", parsedURL.Scheme)
		getExitInfo()
	}
}

func runServer(parsedURL *url.URL, signalChan chan os.Signal) {
	tlsCode, tlsConfig := getTLSProtocol(parsedURL)
	server := internal.NewServer(parsedURL, tlsCode, tlsConfig, logger)
	go func() {
		logger.Info("Server started: %v", parsedURL.String())
		for {
			if err := server.Start(); err != nil {
				logger.Error("Server error: %v", err)
				time.Sleep(internal.ServiceCooldown)
				server.Stop()
				logger.Info("Server restarted")
			}
		}
	}()
	<-signalChan
	ctx, cancel := context.WithTimeout(context.Background(), internal.ShutdownTimeout)
	defer cancel()
	logger.Info("Server shutting down")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error: %v", err)
	}
	logger.Info("Server shutdown complete")
}

func runClient(parsedURL *url.URL, signalChan chan os.Signal) {
	client := internal.NewClient(parsedURL, logger)
	go func() {
		logger.Info("Client started: %v", parsedURL.String())
		for {
			if err := client.Start(); err != nil {
				logger.Error("Client error: %v", err)
				time.Sleep(internal.ServiceCooldown)
				client.Stop()
				logger.Info("Client restarted")
			}
		}
	}()
	<-signalChan
	ctx, cancel := context.WithTimeout(context.Background(), internal.ShutdownTimeout)
	defer cancel()
	logger.Info("Client shutting down")
	if err := client.Shutdown(ctx); err != nil {
		logger.Error("Client shutdown error: %v", err)
	}
	logger.Info("Client shutdown complete")
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
		logger.Info("TLS code-0: selected")
		return tlsCode, nil
	case "1":
		logger.Info("TLS code-1: RAM cert")
		return tlsCode, tlsConfig
	case "2":
		crtFile, keyFile := parsedURL.Query().Get("crt"), parsedURL.Query().Get("key")
		cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
		if err != nil {
			logger.Error("Load failed: %v", err)
			logger.Warn("TLS code-1: RAM cert")
			return "1", tlsConfig
		}
		cachedCert := cert
		lastReload := time.Now()
		tlsConfig = &tls.Config{
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				if time.Since(lastReload) >= internal.ReloadInterval {
					newCert, err := tls.LoadX509KeyPair(crtFile, keyFile)
					if err != nil {
						logger.Error("Reload failed: %v", err)
					} else {
						logger.Debug("Cert reloaded: %v", crtFile)
						cachedCert = newCert
					}
					lastReload = time.Now()
				}
				return &cachedCert, nil
			},
		}
		if cert.Leaf != nil {
			logger.Info("TLS code-2: %v", cert.Leaf.Subject.CommonName)
		} else {
			logger.Warn("TLS code-2: unknown")
		}
		return tlsCode, tlsConfig
	default:
		logger.Warn("TLS code-0: unencrypted")
		return "0", nil
	}
}
