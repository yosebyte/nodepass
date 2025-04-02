package internal

import (
	"context"
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
	tlsCode       string
	logger        *log.Logger
	tunnelAddr    *net.TCPAddr
	remoteAddr    *net.TCPAddr
	targetTCPAddr *net.TCPAddr
	targetUDPAddr *net.UDPAddr
	tunnelTCPConn *net.TCPConn
	targetTCPConn *net.TCPConn
	targetUDPConn *net.UDPConn
	remotePool    *conn.Pool
	ctx           context.Context
	cancel        context.CancelFunc
}

var (
	semaphoreLimit  = getEnvAsInt("SEMAPHORE_LIMIT", 1024)
	minPoolCapacity = getEnvAsInt("MIN_POOL_CAPACITY", 16)
	maxPoolCapacity = getEnvAsInt("MAX_POOL_CAPACITY", 1024)
	udpDataBufSize  = getEnvAsInt("UDP_DATA_BUF_SIZE", 8192)
	udpReadTimeout  = getEnvAsDuration("UDP_READ_TIMEOUT", 5*time.Second)
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
	targetTCPAddr := strings.TrimPrefix(parsedURL.Path, "/")
	if targetTCPAddr, err := net.ResolveTCPAddr("tcp", targetTCPAddr); err == nil {
		c.targetTCPAddr = targetTCPAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
	if targetUDPAddr, err := net.ResolveUDPAddr("udp", targetTCPAddr); err == nil {
		c.targetUDPAddr = targetUDPAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
}

func (c *common) initContext() {
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
}

func (c *common) shutdown(ctx context.Context, stopFunc func()) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		stopFunc()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
