package internal

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/yosebyte/x/log"
)

const (
	MaxSemaphoreLimit   = 1024
	MaxSignalQueueLimit = 1024
	MaxSignalBuffer     = 1024
	MaxUDPDataBuffer    = 8192
	MaxUDPDataTimeout   = 10 * time.Second
	MaxReportInterval   = 15 * time.Second
	ServerCooldownDelay = 1 * time.Second
	ClientCooldownDelay = 5 * time.Second
	ShutdownTimeout     = 5 * time.Second
	CheckSignalPING     = "[NODEPASS]<PING>\n"
	LaunchSignalTCP     = "[NODEPASS]<TCP>\n"
	LaunchSignalUDP     = "[NODEPASS]<UDP>\n"
)

type Common struct {
	logger        *log.Logger
	tunnelAddr    *net.TCPAddr
	remoteAddr    *net.TCPAddr
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
		c.logger.Error("Resolve failed: %v", err)
	}
	tunnelHost, _, _ := net.SplitHostPort(parsedURL.Host)
	c.remoteAddr = &net.TCPAddr{
		IP:   net.ParseIP(tunnelHost),
		Port: getRemotePort(),
	}
	targetAddr := strings.TrimPrefix(parsedURL.Path, "/")
	if targetTCPAddr, err := net.ResolveTCPAddr("tcp", targetAddr); err == nil {
		c.targetTCPAddr = targetTCPAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
	if targetUDPAddr, err := net.ResolveUDPAddr("udp", targetAddr); err == nil {
		c.targetUDPAddr = targetUDPAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
}

func getRemotePort() int {
	timestamp := time.Now().Truncate(time.Minute).Unix()
	port := (timestamp % (7169)) + 1024
	return int(port)
}
