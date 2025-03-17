package internal

import (
	"context"
	"crypto/tls"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yosebyte/x/log"
	"github.com/yosebyte/x/pool"
)

const (
	ReportSignal = "[NODEPASS]<REPORT>\n"
	LaunchSignal = "[NODEPASS]<LAUNCH>\n"
)

var (
	SemaphoreLimit   = getEnvAsInt("SEMAPHORE_LIMIT", 1024)
	SignalQueueLimit = getEnvAsInt("SIGNAL_QUEUE_LIMIT", 1024)
	SignalBuffer     = getEnvAsInt("SIGNAL_BUFFER", 1024)
	MinPoolCapacity  = getEnvAsInt("MIN_POOL_CAPACITY", 8)
	MaxPoolCapacity  = getEnvAsInt("MAX_POOL_CAPACITY", 1024)
	ReportInterval   = getEnvAsDuration("REPORT_INTERVAL", 5*time.Second)
	ServerCooldown   = getEnvAsDuration("SERVER_COOLDOWN", 5*time.Second)
	ClientCooldown   = getEnvAsDuration("CLIENT_COOLDOWN", 5*time.Second)
	ShutdownTimeout  = getEnvAsDuration("SHUTDOWN_TIMEOUT", 5*time.Second)
)

type Common struct {
	logger     *log.Logger
	tunnelAddr *net.TCPAddr
	remoteAddr *net.TCPAddr
	targetAddr *net.TCPAddr
	tunnelConn *tls.Conn
	targetConn net.Conn
	remoteConn net.Conn
	pool       *pool.Pool
	errChan    chan error
	ctx        context.Context
	cancel     context.CancelFunc
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

func getRandPort() int {
	return rand.Intn(7169) + 1024
}

func (c *Common) getAddress(parsedURL *url.URL) {
	if tunnelAddr, err := net.ResolveTCPAddr("tcp", parsedURL.Host); err == nil {
		c.tunnelAddr = tunnelAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
	c.remoteAddr = &net.TCPAddr{
		IP:   c.tunnelAddr.IP,
		Port: getRandPort(),
	}
	targetAddr := strings.TrimPrefix(parsedURL.Path, "/")
	if targetTCPAddr, err := net.ResolveTCPAddr("tcp", targetAddr); err == nil {
		c.targetAddr = targetTCPAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
}
