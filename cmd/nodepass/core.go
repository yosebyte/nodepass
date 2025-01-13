package main

import (
	"net/url"
	"os"
	"time"

	"github.com/yosebyte/nodepass/internal"
	"github.com/yosebyte/x/tls"
)

func coreDispatch(parsedURL *url.URL, stop chan os.Signal) {
	switch parsedURL.Scheme {
	case "server":
		runServer(parsedURL, stop)
	case "client":
		runClient(parsedURL, stop)
	default:
		logger.Fatal("Invalid scheme: %v", parsedURL.Scheme)
		getExitInfo()
	}
}

func runServer(parsedURL *url.URL, stop chan os.Signal) {
	tlsConfig, err := tls.NewTLSconfig("yosebyte/nodepass:" + version)
	if err != nil {
		logger.Error("Unable to generate TLS config: %v", err)
	}
	server := internal.NewServer(parsedURL, tlsConfig, logger)
	go func() {
		logger.Info("Server started: %v", parsedURL.String())
		for {
			if err := server.Start(); err != nil {
				logger.Error("Server error: %v", err)
			}
			time.Sleep(1 * time.Second)
			logger.Info("Server restarted")
		}
	}()
	<-stop
	logger.Info("Server stopping")
	server.Stop()
	logger.Info("Server stopped")
}

func runClient(parsedURL *url.URL, stop chan os.Signal) {
	client := internal.NewClient(parsedURL, logger)
	go func() {
		logger.Info("Client started: %v", parsedURL.String())
		for {
			if err := client.Start(); err != nil {
				logger.Error("Client error: %v", err)
			}
			time.Sleep(1 * time.Second)
			logger.Info("Client restarted")
		}
	}()
	<-stop
	logger.Info("Client stopping")
	client.Stop()
	logger.Info("Client stopped")
}
