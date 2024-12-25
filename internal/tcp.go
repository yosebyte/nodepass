package internal

import (
	"crypto/tls"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/yosebyte/x/io"
	"github.com/yosebyte/x/log"
)

func ServeTCP(parsedURL *url.URL, targetTCPListen *net.TCPListener, serverListen net.Listener, clientConn net.Conn, mu *sync.Mutex, done <-chan struct{}) error {
	sem := make(chan struct{}, MaxSemaphoreLimit)
	for {
		select {
		case <-done:
			return nil
		default:
			targetConn, err := targetTCPListen.AcceptTCP()
			if err != nil {
				log.Error("Unable to accept connections form target address: %v %v", targetTCPListen.Addr(), err)
				time.Sleep(1 * time.Second)
				continue
			}
			log.Info("Target connection established from: %v", targetConn.RemoteAddr())
			defer func() {
				if targetConn != nil {
					targetConn.Close()
				}
			}()
			sem <- struct{}{}
			go func(targetConn *net.TCPConn) {
				defer func() { <-sem }()
				mu.Lock()
				_, err = clientConn.Write([]byte("[PASSPORT]<TCP>\n"))
				mu.Unlock()
				if err != nil {
					log.Error("Unable to send signal: %v", err)
					return
				}
				remoteConn, err := serverListen.Accept()
				if err != nil {
					log.Error("Unable to accept connections form link address: %v %v", serverListen.Addr(), err)
					return
				}
				log.Info("Remote connection established from: %v", remoteConn.RemoteAddr())
				defer func() {
					if remoteConn != nil {
						remoteConn.Close()
					}
				}()
				log.Info("Starting data exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
				if err := io.DataExchange(remoteConn, targetConn); err != nil {
					log.Info("Connection closed: %v", err)
				}
			}(targetConn)
		}
	}
}

func ClientTCP(serverAddr, targetTCPAddr *net.TCPAddr) error {
	remoteConn, err := tls.Dial("tcp", serverAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Error("Unable to dial server address: %v", serverAddr)
		return err
	}
	log.Info("Remote connection established to: %v", serverAddr)
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	targetConn, err := net.DialTCP("tcp", nil, targetTCPAddr)
	if err != nil {
		log.Error("Unable to dial target address: %v", targetTCPAddr)
		return err
	}
	log.Info("Target connection established to: %v", targetTCPAddr)
	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()
	log.Info("Starting data exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
	if err := io.DataExchange(remoteConn, targetConn); err != nil {
		log.Info("Connection closed: %v", err)
	}
	return nil
}
