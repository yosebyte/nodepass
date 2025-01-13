package internal

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/yosebyte/x/log"
)

const (
	MaxSemaphoreLimit = 1024
	MaxSignalBuffer   = 1024
	MaxUDPDataBuffer  = 8192
	MaxUDPDataTimeout = 5 * time.Second
	MaxReportInterval = 5 * time.Second
	MaxReportTimeout  = 5 * time.Second
)

type Common struct {
	logger        *log.Logger
	serverAddr    *net.TCPAddr
	targetTCPAddr *net.TCPAddr
	targetUDPAddr *net.UDPAddr
	tunnleConn    net.Conn
	targetTCPConn net.Conn
	targetUDPConn net.Conn
	remoteTCPConn net.Conn
	remoteUDPConn net.Conn
	done          chan struct{}
}

func (c *Common) GetAddress(parsedURL *url.URL, logger *log.Logger) {
	if serverAddr, err := net.ResolveTCPAddr("tcp", parsedURL.Host); err == nil {
		c.serverAddr = serverAddr
	} else {
		c.logger.Error("Unable to resolve server address: %v", err)
	}
	targetAddr := strings.TrimPrefix(parsedURL.Path, "/")
	if targetTCPAddr, err := net.ResolveTCPAddr("tcp", targetAddr); err == nil {
		c.targetTCPAddr = targetTCPAddr
	} else {
		c.logger.Error("Unable to resolve target TCP address: %v", err)
	}
	if targetUDPAddr, err := net.ResolveUDPAddr("udp", targetAddr); err == nil {
		c.targetUDPAddr = targetUDPAddr
	} else {
		c.logger.Error("Unable to resolve target UDP address: %v", err)
	}
}
