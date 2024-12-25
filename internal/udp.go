package internal

import (
	"crypto/tls"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/yosebyte/x/log"
)

func ServeUDP(parsedURL *url.URL, targetUDPConn *net.UDPConn, serverListen net.Listener, clientConn net.Conn, mu *sync.Mutex, done <-chan struct{}) error {
	sem := make(chan struct{}, MaxSemaphoreLimit)
	for {
		select {
		case <-done:
			return nil
		default:
			buffer := make([]byte, MaxDataBuffer)
			n, clientAddr, err := targetUDPConn.ReadFromUDP(buffer)
			if err != nil {
				log.Error("Unable to read from client address: %v %v", clientAddr, err)
				time.Sleep(1 * time.Second)
				continue
			}
			mu.Lock()
			_, err = clientConn.Write([]byte("[PASSPORT]<UDP>\n"))
			mu.Unlock()
			if err != nil {
				log.Error("Unable to send signal: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			remoteConn, err := serverListen.Accept()
			if err != nil {
				log.Error("Unable to accept connections from server address: %v %v", serverListen.Addr(), err)
				time.Sleep(1 * time.Second)
				continue
			}
			log.Info("Remote connection established from: %v", remoteConn.RemoteAddr())
			defer func() {
				if remoteConn != nil {
					remoteConn.Close()
				}
			}()
			sem <- struct{}{}
			go func(buffer []byte, n int, remoteConn net.Conn, clientAddr *net.UDPAddr) {
				defer func() { <-sem }()
				log.Info("Starting data transfer: %v <-> %v", clientAddr, targetUDPConn.LocalAddr())
				_, err = remoteConn.Write(buffer[:n])
				if err != nil {
					log.Error("Unable to write to link address: %v %v", serverListen.Addr(), err)
					return
				}
				n, err = remoteConn.Read(buffer)
				if err != nil {
					log.Error("Unable to read from link address: %v %v", serverListen.Addr(), err)
					return
				}
				_, err = targetUDPConn.WriteToUDP(buffer[:n], clientAddr)
				if err != nil {
					log.Error("Unable to write to client address: %v %v", clientAddr, err)
					return
				}
				log.Info("Transfer completed successfully")
			}(buffer, n, remoteConn, clientAddr)
		}
	}
}

func ClientUDP(serverAddr *net.TCPAddr, targetUDPAddr *net.UDPAddr) error {
	remoteConn, err := tls.Dial("tcp", serverAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Error("Unable to dial target address: %v", serverAddr)
		return err
	}
	log.Info("Remote connection established to: %v", serverAddr)
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	buffer := make([]byte, MaxDataBuffer)
	n, err := remoteConn.Read(buffer)
	if err != nil {
		log.Error("Unable to read from remote address: %v", remoteConn.RemoteAddr())
		return err
	}
	targetConn, err := net.DialUDP("udp", nil, targetUDPAddr)
	if err != nil {
		log.Error("Unable to dial target address: %v", targetUDPAddr)
		return err
	}
	log.Info("Target connection established to: %v", targetUDPAddr)
	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()
	err = targetConn.SetDeadline(time.Now().Add(MaxUDPTimeout * time.Second))
	if err != nil {
		log.Error("Unable to set deadline: %v", err)
		return err
	}
	log.Info("Starting data transfer: %v <-> %v", serverAddr, targetUDPAddr)
	_, err = targetConn.Write(buffer[:n])
	if err != nil {
		log.Error("Unable to write to target address: %v", targetUDPAddr)
		return err
	}
	n, _, err = targetConn.ReadFromUDP(buffer)
	if err != nil {
		log.Error("Unable to read from target address: %v", targetUDPAddr)
		return err
	}
	_, err = remoteConn.Write(buffer[:n])
	if err != nil {
		log.Error("Unable to write to remote address: %v", serverAddr)
		return err
	}
	log.Info("Transfer completed successfully")
	return nil
}
