package internal

import (
	"crypto/tls"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/yosebyte/x/log"
)

func Server(parsedURL *url.URL, tlsConfig *tls.Config) error {
	serverAddr, err := net.ResolveTCPAddr("tcp", parsedURL.Host)
	if err != nil {
		log.Error("Unable to resolve link address: %v", parsedURL.Host)
		return err
	}
	targetAddr := strings.TrimPrefix(parsedURL.Path, "/")
	targetTCPAddr, err := net.ResolveTCPAddr("tcp", targetAddr)
	if err != nil {
		log.Error("Unable to resolve target TCP address: %v", targetAddr)
		return err
	}
	targetUDPAddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		log.Error("Unable to resolve target UDP address: %v", targetAddr)
		return err
	}
	serverListen, err := tls.Listen("tcp", serverAddr.String(), tlsConfig)
	if err != nil {
		log.Error("Unable to listen server address: %v", serverAddr)
		return err
	}
	defer func() {
		if serverListen != nil {
			serverListen.Close()
		}
	}()
	clientConn, err := serverListen.Accept()
	if err != nil {
		log.Error("Unable to accept connections form server address: %v", serverAddr)
		return err
	}
	log.Info("Tunnel connection established from: %v", clientConn.RemoteAddr().String())
	defer func() {
		if clientConn != nil {
			clientConn.Close()
		}
	}()
	targetTCPListen, err := net.ListenTCP("tcp", targetTCPAddr)
	if err != nil {
		log.Error("Unable to listen target TCP address: [%v]", targetTCPAddr)
		return err
	}
	defer func() {
		if targetTCPListen != nil {
			targetTCPListen.Close()
		}
	}()
	targetUDPConn, err := net.ListenUDP("udp", targetUDPAddr)
	if err != nil {
		log.Error("Unable to listen target UDP address: [%v]", targetUDPAddr)
		return err
	}
	defer func() {
		if targetUDPConn != nil {
			targetUDPConn.Close()
		}
	}()
	var sharedMU sync.Mutex
	errChan := make(chan error, 2)
	done := make(chan struct{})
	go func() {
		errChan <- healthCheck(serverListen, targetTCPListen, targetUDPConn, clientConn, &sharedMU, done)
	}()
	go func() {
		errChan <- ServeTCP(parsedURL, targetTCPListen, serverListen, clientConn, &sharedMU, done)
	}()
	go func() {
		errChan <- ServeUDP(parsedURL, targetUDPConn, serverListen, clientConn, &sharedMU, done)
	}()
	return <-errChan
}

func healthCheck(serverListen net.Listener, targetTCPListen *net.TCPListener, targetUDPConn *net.UDPConn, clientConn net.Conn, sharedMU *sync.Mutex, done chan struct{}) error {
	for {
		time.Sleep(MaxReportInterval * time.Second)
		sharedMU.Lock()
		_, err := clientConn.Write([]byte("[]\n"))
		sharedMU.Unlock()
		if err != nil {
			log.Error("Tunnel connection health check failed")
			if serverListen != nil {
				serverListen.Close()
			}
			if targetTCPListen != nil {
				targetTCPListen.Close()
			}
			if targetUDPConn != nil {
				targetUDPConn.Close()
			}
			if clientConn != nil {
				clientConn.Close()
			}
			close(done)
			return err
		}
	}
}
