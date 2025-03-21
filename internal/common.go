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

	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type common struct {
	logger     *log.Logger
	tunnelAddr *net.TCPAddr
	remoteAddr *net.TCPAddr
	targetAddr *net.TCPAddr
	tunnelConn *tls.Conn
	targetConn net.Conn
	remotePool *conn.Pool
	ctx        context.Context
	cancel     context.CancelFunc
}

var (
	semaphoreLimit  = getEnvAsInt("SEMAPHORE_LIMIT", 1024)
	minPoolCapacity = getEnvAsInt("MIN_POOL_CAPACITY", 16)
	maxPoolCapacity = getEnvAsInt("MAX_POOL_CAPACITY", 1024)
	reportInterval  = getEnvAsDuration("REPORT_INTERVAL", 5*time.Second)
	ServiceCooldown = getEnvAsDuration("SERVICE_COOLDOWN", 5*time.Second)
	ShutdownTimeout = getEnvAsDuration("SHUTDOWN_TIMEOUT", 5*time.Second)
)

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

func (c *common) getAddress(parsedURL *url.URL) {
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
	if targetAddr, err := net.ResolveTCPAddr("tcp", targetAddr); err == nil {
		c.targetAddr = targetAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
}
