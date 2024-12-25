package internal

import (
	"crypto/tls"
	"net"
	"net/url"
	"strings"

	"github.com/yosebyte/x/log"
)

func Client(parsedURL *url.URL) error {
	serverAddr, err := net.ResolveTCPAddr("tcp", parsedURL.Host)
	if err != nil {
		log.Error("Unable to resolve link address: %v", parsedURL.Host)
		return err
	}
	targetAddr := strings.TrimPrefix(parsedURL.Path, "/")
	targetTCPAddr, err := net.ResolveTCPAddr("tcp", targetAddr)
	if err != nil {
		log.Error("Unable to resolve target address: %v", targetAddr)
		return err
	}
	targetUDPAddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		log.Error("Unable to resolve target address: %v", targetAddr)
		return err
	}
	serverConn, err := tls.Dial("tcp", serverAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Error("Unable to dial server address: %v", serverAddr)
		return err
	}
	defer func() {
		if serverConn != nil {
			serverConn.Close()
		}
	}()
	log.Info("Tunnel connection established to: %v", serverAddr)
	errChan := make(chan error, 2)
	buffer := make([]byte, MaxSignalBuffer)
	for {
		n, err := serverConn.Read(buffer)
		if err != nil {
			log.Error("Unable to read form server address: %v %v", serverAddr, err)
			break
		}
		if string(buffer[:n]) == "[PASSPORT]<TCP>\n" {
			go func() {
				errChan <- ClientTCP(serverAddr, targetTCPAddr)
			}()
		}
		if string(buffer[:n]) == "[PASSPORT]<UDP>\n" {
			go func() {
				errChan <- ClientUDP(serverAddr, targetUDPAddr)
			}()
		}
	}
	return <-errChan
}
