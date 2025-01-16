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
	MaxUDPDataBuffer  = 8192
	MaxCooldownDelay  = 5 * time.Second
	MaxUDPDataTimeout = 10 * time.Second
	MaxReportInterval = 15 * time.Second
)

type Common struct {
	logger        *log.Logger
	tunnelAddr    *net.TCPAddr
	targetTCPAddr *net.TCPAddr
	targetUDPAddr *net.UDPAddr
	tunnelConn    net.Conn
	targetTCPConn net.Conn
	targetUDPConn net.Conn
	remoteTCPConn net.Conn
	remoteUDPConn net.Conn
	errChan       chan error
}

func (c *Common) GetAddress(parsedURL *url.URL, logger *log.Logger) {
	if tunnelAddr, err := net.ResolveTCPAddr("tcp", parsedURL.Host); err == nil {
		c.tunnelAddr = tunnelAddr
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
