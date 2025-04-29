// 内部工具包，提供共享功能
package internal

import (
	"context"
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

// Common 包含所有模式共享的核心功能
type Common struct {
	tlsCode          string             // TLS模式代码
	logger           *log.Logger        // 日志记录器
	tunnelAddr       *net.TCPAddr       // 隧道地址
	targetAddr       string             // 目标地址字符串
	targetTCPAddr    *net.TCPAddr       // 目标TCP地址
	targetUDPAddr    *net.UDPAddr       // 目标UDP地址
	tunnelTCPConn    *net.TCPConn       // 隧道TCP连接
	targetTCPConn    *net.TCPConn       // 目标TCP连接
	targetUDPConn    *net.UDPConn       // 目标UDP连接
	tunnelPool       *conn.Pool         // 隧道连接池
	ctx              context.Context    // 上下文
	cancel           context.CancelFunc // 取消函数
	tcpBytesReceived uint64             // TCP接收字节数
	tcpBytesSent     uint64             // TCP发送字节数
	udpBytesReceived uint64             // UDP接收字节数
	udpBytesSent     uint64             // UDP发送字节数
}

// 配置变量，可通过环境变量调整
var (
	semaphoreLimit  = getEnvAsInt("NP_SEMAPHORE_LIMIT", 1024)                 // 信号量限制
	minPoolCapacity = getEnvAsInt("NP_MIN_POOL_CAPACITY", 16)                 // 最小池容量
	maxPoolCapacity = getEnvAsInt("NP_MAX_POOL_CAPACITY", 1024)               // 最大池容量
	udpDataBufSize  = getEnvAsInt("NP_UDP_DATA_BUF_SIZE", 8192)               // UDP数据缓冲区大小
	udpReadTimeout  = getEnvAsDuration("NP_UDP_READ_TIMEOUT", 5*time.Second)  // UDP读取超时
	udpDialTimeout  = getEnvAsDuration("NP_UDP_DIAL_TIMEOUT", 5*time.Second)  // UDP拨号超时
	tcpDialTimeout  = getEnvAsDuration("NP_TCP_DIAL_TIMEOUT", 5*time.Second)  // TCP拨号超时
	minPoolInterval = getEnvAsDuration("NP_MIN_POOL_INTERVAL", 1*time.Second) // 最小池间隔
	maxPoolInterval = getEnvAsDuration("NP_MAX_POOL_INTERVAL", 5*time.Second) // 最大池间隔
	reportInterval  = getEnvAsDuration("NP_REPORT_INTERVAL", 5*time.Second)   // 报告间隔
	serviceCooldown = getEnvAsDuration("NP_SERVICE_COOLDOWN", 5*time.Second)  // 服务冷却时间
	shutdownTimeout = getEnvAsDuration("NP_SHUTDOWN_TIMEOUT", 5*time.Second)  // 关闭超时
	ReloadInterval  = getEnvAsDuration("NP_RELOAD_INTERVAL", 1*time.Hour)     // 重载间隔
)

// 从环境变量获取整数值，如果不存在则使用默认值
func getEnvAsInt(name string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := strconv.Atoi(valueStr); err == nil && value >= 0 {
			return value
		}
	}
	return defaultValue
}

// 从环境变量获取时间间隔，如果不存在则使用默认值
func getEnvAsDuration(name string, defaultValue time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := time.ParseDuration(valueStr); err == nil && value >= 0 {
			return value
		}
	}
	return defaultValue
}

// 解析和设置地址信息
func (c *Common) getAddress(parsedURL *url.URL) {
	// 解析隧道地址
	if tunnelAddr, err := net.ResolveTCPAddr("tcp", parsedURL.Host); err == nil {
		c.tunnelAddr = tunnelAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}

	// 处理目标地址
	targetAddr := strings.TrimPrefix(parsedURL.Path, "/")
	c.targetAddr = targetAddr

	// 解析目标TCP地址
	if targetTCPAddr, err := net.ResolveTCPAddr("tcp", targetAddr); err == nil {
		c.targetTCPAddr = targetTCPAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}

	// 解析目标UDP地址
	if targetUDPAddr, err := net.ResolveUDPAddr("udp", targetAddr); err == nil {
		c.targetUDPAddr = targetUDPAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}
}

// 初始化上下文
func (c *Common) initContext() {
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
}

// 优雅关闭
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

// 添加TCP统计数据
func (c *Common) AddTCPStats(received, sent uint64) {
	atomic.AddUint64(&c.tcpBytesReceived, received)
	atomic.AddUint64(&c.tcpBytesSent, sent)
}

// 添加UDP接收统计
func (c *Common) AddUDPReceived(bytes uint64) {
	atomic.AddUint64(&c.udpBytesReceived, bytes)
}

// 添加UDP发送统计
func (c *Common) AddUDPSent(bytes uint64) {
	atomic.AddUint64(&c.udpBytesSent, bytes)
}

// 获取TCP统计数据
func (c *Common) GetTCPStats() (uint64, uint64) {
	return atomic.LoadUint64(&c.tcpBytesReceived), atomic.LoadUint64(&c.tcpBytesSent)
}

// 获取UDP统计数据
func (c *Common) GetUDPStats() (uint64, uint64) {
	return atomic.LoadUint64(&c.udpBytesReceived), atomic.LoadUint64(&c.udpBytesSent)
}
