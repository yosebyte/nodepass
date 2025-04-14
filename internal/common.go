package internal

import (
	"context"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type Common struct {
	tlsCode          string
	logger           *log.Logger
	tunnelAddr       *net.TCPAddr
	remoteAddr       *net.TCPAddr
	targetAddr       string
	targetTCPAddr    *net.TCPAddr
	targetUDPAddr    *net.UDPAddr
	tunnelTCPConn    *net.TCPConn
	targetTCPConn    *net.TCPConn
	targetUDPConn    *net.UDPConn
	remotePool       *conn.Pool
	ctx              context.Context
	cancel           context.CancelFunc
	tcpBytesReceived uint64
	tcpBytesSent     uint64
	udpBytesReceived uint64
	udpBytesSent     uint64
}

var (
	semaphoreLimit  = getEnvAsInt("SEMAPHORE_LIMIT", 1024)
	minPoolCapacity = getEnvAsInt("MIN_POOL_CAPACITY", 16)
	maxPoolCapacity = getEnvAsInt("MAX_POOL_CAPACITY", 1024)
	udpDataBufSize  = getEnvAsInt("UDP_DATA_BUF_SIZE", 8192)
	udpReadTimeout  = getEnvAsDuration("UDP_READ_TIMEOUT", 5*time.Second)
	minPoolInterval = getEnvAsDuration("MIN_POOL_INTERVAL", 1*time.Second)
	maxPoolInterval = getEnvAsDuration("MAX_POOL_INTERVAL", 5*time.Second)
	reportInterval  = getEnvAsDuration("REPORT_INTERVAL", 5*time.Second)
	serviceCooldown = getEnvAsDuration("SERVICE_COOLDOWN", 5*time.Second)
	shutdownTimeout = getEnvAsDuration("SHUTDOWN_TIMEOUT", 5*time.Second)
	ReloadInterval  = getEnvAsDuration("RELOAD_INTERVAL", 1*time.Hour)
	dialTimeout     = getEnvAsDuration("DIAL_TIMEOUT", 10*time.Second)
)

func getEnvAsInt(name string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := strconv.Atoi(valueStr); err == nil && value >= 0 {
			return value
		}
	}
	return defaultValue
}

func getEnvAsDuration(name string, defaultValue time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := time.ParseDuration(valueStr); err == nil && value >= 0 {
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
	c.targetAddr = targetAddr
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

func (c *Common) initContext() {
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
}

func (c *Common) shutdown(ctx context.Context, stopFunc func()) error {
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

func (c *Common) AddTCPStats(received, sent uint64) {
	atomic.AddUint64(&c.tcpBytesReceived, received)
	atomic.AddUint64(&c.tcpBytesSent, sent)
}

func (c *Common) AddUDPReceived(bytes uint64) {
	atomic.AddUint64(&c.udpBytesReceived, bytes)
}

func (c *Common) AddUDPSent(bytes uint64) {
	atomic.AddUint64(&c.udpBytesSent, bytes)
}

func (c *Common) GetTCPStats() (uint64, uint64) {
	return atomic.LoadUint64(&c.tcpBytesReceived), atomic.LoadUint64(&c.tcpBytesSent)
}

func (c *Common) GetUDPStats() (uint64, uint64) {
	return atomic.LoadUint64(&c.udpBytesReceived), atomic.LoadUint64(&c.udpBytesSent)
}

func (c *Common) statsReporter() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			tcpReceived, tcpSent := c.GetTCPStats()
			udpReceived, udpSent := c.GetUDPStats()
			c.logger.Debug("Periodic reporter: TRAFFIC_STATS|TCP_RX=%d|TCP_TX=%d|UDP_RX=%d|UDP_TX=%d",
				tcpReceived, tcpSent, udpReceived, udpSent)
		}
		time.Sleep(reportInterval)
	}
}
