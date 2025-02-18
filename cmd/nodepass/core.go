package main

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/yosebyte/nodepass/internal"
	"github.com/yosebyte/x/tls"
)

func coreDispatch(parsedURL *url.URL, signalChan chan os.Signal) {
	switch parsedURL.Scheme {
	case "server":
		runServer(parsedURL, signalChan)
	case "client":
		runClient(parsedURL, signalChan)
	default:
		logger.Fatal("Invalid scheme: %v", parsedURL.Scheme)
		getExitInfo()
	}
}

func runServer(parsedURL *url.URL, signalChan chan os.Signal) {
	logger.Info("Apply RAM cert: %v", version)
	tlsConfig, err := tls.GenerateTLSConfig("yosebyte/nodepass:" + version)
	if err != nil {
		logger.Fatal("Generate failed: %v", err)
		return
	}
	server := internal.NewServer(parsedURL, tlsConfig, logger)
	go func() {
		logger.Info("Server started: %v", parsedURL.String())
		for {
			if err := server.Start(); err != nil {
				logger.Error("Server error: %v", err)
				server.Stop()
				time.Sleep(internal.ServerCooldownDelay)
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
				client.Stop()
				time.Sleep(internal.ClientCooldownDelay)
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
