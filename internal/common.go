package internal

import (
	"crypto/tls"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yosebyte/x/log"
)

var (
	SemaphoreLimit   = getEnvAsInt("SEMAPHORE_LIMIT", 1024)
	SignalQueueLimit = getEnvAsInt("SIGNAL_QUEUE_LIMIT", 1024)
	SignalBuffer     = getEnvAsInt("SIGNAL_BUFFER", 1024)
	UDPDataBuffer    = getEnvAsInt("UDP_DATA_BUFFER", 8192)
	MinPoolCapacity  = getEnvAsInt("MIN_POOL_CAPACITY", 8)
	MaxPoolCapacity  = getEnvAsInt("MAX_POOL_CAPACITY", 1024)
	UDPDataTimeout   = getEnvAsDuration("UDP_DATA_TIMEOUT", 10*time.Second)
	ReportInterval   = getEnvAsDuration("REPORT_INTERVAL", 5*time.Second)
	ServerCooldown   = getEnvAsDuration("SERVER_COOLDOWN", 5*time.Second)
	ClientCooldown   = getEnvAsDuration("CLIENT_COOLDOWN", 5*time.Second)
	ShutdownTimeout  = getEnvAsDuration("SHUTDOWN_TIMEOUT", 5*time.Second)
	CheckSignalPING  = getEnv("CHECK_SIGNAL_PING", "[NODEPASS]<PING>\n")
	LaunchSignalTCP  = getEnv("LAUNCH_SIGNAL_TCP", "[NODEPASS]<TCP>\n")
	LaunchSignalUDP  = getEnv("LAUNCH_SIGNAL_UDP", "[NODEPASS]<UDP>\n")
)

type Common struct {
	logger        *log.Logger
	tunnelAddr    *net.TCPAddr
	remoteTCPAddr *net.TCPAddr
	remoteUDPAddr *net.TCPAddr
	targetTCPAddr *net.TCPAddr
	targetUDPAddr *net.UDPAddr
	tunnelConn    *tls.Conn
	targetTCPConn *net.TCPConn
	targetUDPConn *net.UDPConn
	remoteTCPConn net.Conn
	remoteUDPConn net.Conn
	errChan       chan error
}

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(name string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnvAsDuration(name string, defaultValue time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := time.ParseDuration(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func (c *Common) GetAddress(parsedURL *url.URL, logger *log.Logger) {
	if tunnelAddr, err := net.ResolveTCPAddr("tcp", parsedURL.Host); err == nil {
		c.tunnelAddr = tunnelAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
	c.remoteTCPAddr = &net.TCPAddr{
		IP:   c.tunnelAddr.IP,
		Port: c.tunnelAddr.Port + 1,
	}
	c.remoteUDPAddr = &net.TCPAddr{
		IP:   c.tunnelAddr.IP,
		Port: c.tunnelAddr.Port + 2,
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
