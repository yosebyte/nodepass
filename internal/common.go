// 内部包，提供共享功能
package internal

import (
	"bufio"
	"context"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NodePassProject/conn"
	"github.com/NodePassProject/logs"
	"github.com/NodePassProject/pool"
)

// Common 包含所有模式共享的核心功能
type Common struct {
	mu             sync.Mutex         // 互斥锁
	tlsCode        string             // TLS模式代码
	dataFlow       string             // 数据流向
	logger         *logs.Logger       // 日志记录器
	tunnelAddr     *net.TCPAddr       // 隧道地址
	targetAddr     string             // 目标地址字符串
	targetTCPAddr  *net.TCPAddr       // 目标TCP地址
	targetUDPAddr  *net.UDPAddr       // 目标UDP地址
	targetListener *net.TCPListener   // 目标监听器
	tunnelTCPConn  *net.TCPConn       // 隧道TCP连接
	targetTCPConn  *net.TCPConn       // 目标TCP连接
	targetUDPConn  *net.UDPConn       // 目标UDP连接
	tunnelPool     *pool.Pool         // 隧道连接池
	semaphore      chan struct{}      // 信号量通道
	bufReader      *bufio.Reader      // 缓冲读取器
	signalChan     chan string        // 信号通道
	ctx            context.Context    // 上下文
	cancel         context.CancelFunc // 取消函数
}

// 配置变量，可通过环境变量调整
var (
	semaphoreLimit  = getEnvAsInt("NP_SEMAPHORE_LIMIT", 1024)                 // 信号量限制
	minPoolCapacity = getEnvAsInt("NP_MIN_POOL_CAPACITY", 16)                 // 最小池容量
	maxPoolCapacity = getEnvAsInt("NP_MAX_POOL_CAPACITY", 1024)               // 最大池容量
	udpDataBufSize  = getEnvAsInt("NP_UDP_DATA_BUF_SIZE", 8192)               // UDP数据缓冲区大小
	udpReadTimeout  = getEnvAsDuration("NP_UDP_READ_TIMEOUT", 5*time.Second)  // UDP读取超时
	udpDialTimeout  = getEnvAsDuration("NP_UDP_DIAL_TIMEOUT", 5*time.Second)  // UDP拨号超时
	tcpReadTimeout  = getEnvAsDuration("NP_TCP_READ_TIMEOUT", 5*time.Second)  // TCP读取超时
	tcpDialTimeout  = getEnvAsDuration("NP_TCP_DIAL_TIMEOUT", 5*time.Second)  // TCP拨号超时
	minPoolInterval = getEnvAsDuration("NP_MIN_POOL_INTERVAL", 1*time.Second) // 最小池间隔
	maxPoolInterval = getEnvAsDuration("NP_MAX_POOL_INTERVAL", 5*time.Second) // 最大池间隔
	reportInterval  = getEnvAsDuration("NP_REPORT_INTERVAL", 5*time.Second)   // 报告间隔
	serviceCooldown = getEnvAsDuration("NP_SERVICE_COOLDOWN", 5*time.Second)  // 服务冷却时间
	shutdownTimeout = getEnvAsDuration("NP_SHUTDOWN_TIMEOUT", 5*time.Second)  // 关闭超时
	ReloadInterval  = getEnvAsDuration("NP_RELOAD_INTERVAL", 1*time.Hour)     // 重载间隔
)

// getEnvAsInt 从环境变量获取整数值，如果不存在则使用默认值
func getEnvAsInt(name string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := strconv.Atoi(valueStr); err == nil && value >= 0 {
			return value
		}
	}
	return defaultValue
}

// getEnvAsDuration 从环境变量获取时间间隔，如果不存在则使用默认值
func getEnvAsDuration(name string, defaultValue time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := time.ParseDuration(valueStr); err == nil && value >= 0 {
			return value
		}
	}
	return defaultValue
}

// getAddress 解析和设置地址信息
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

// initContext 初始化上下文
func (c *Common) initContext() {
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
}

// initTargetListener 初始化目标监听器
func (c *Common) initTargetListener() error {
	// 初始化目标TCP监听器
	targetListener, err := net.ListenTCP("tcp", c.targetTCPAddr)
	if err != nil {
		return err
	}
	c.targetListener = targetListener

	// 初始化目标UDP监听器
	targetUDPConn, err := net.ListenUDP("udp", c.targetUDPAddr)
	if err != nil {
		return err
	}
	c.targetUDPConn = targetUDPConn

	return nil
}

// shutdown 优雅关闭
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

// commonQueue 共用信号队列
func (c *Common) commonQueue() error {
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			// 读取原始信号
			rawSignal, err := c.bufReader.ReadBytes('\n')
			if err != nil {
				return err
			}
			signal := strings.TrimSpace(string(rawSignal))

			// 将信号发送到通道
			select {
			case c.signalChan <- signal:
			default:
				c.logger.Debug("Queue limit reached: %v", semaphoreLimit)
			}
		}
	}
}

// commonLoop 共用处理循环
func (c *Common) commonLoop() {
	for {
		// 等待连接池准备就绪
		if c.tunnelPool.Ready() {
			go c.commonTCPLoop()
			go c.commonUDPLoop()
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// commonTCPLoop 共用TCP请求处理循环
func (c *Common) commonTCPLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// 接受来自目标的TCP连接
			targetConn, err := c.targetListener.Accept()
			if err != nil {
				continue
			}

			defer func() {
				if targetConn != nil {
					targetConn.Close()
				}
			}()

			c.targetTCPConn = targetConn.(*net.TCPConn)
			c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())

			// 使用信号量限制并发数
			c.semaphore <- struct{}{}

			go func(targetConn net.Conn) {
				defer func() { <-c.semaphore }()

				// 从连接池获取连接
				id, remoteConn := c.tunnelPool.ServerGet()
				if remoteConn == nil {
					c.logger.Error("Get failed: %v", id)
					return
				}

				c.logger.Debug("Tunnel connection: %v <- active %v", id, c.tunnelPool.Active())

				defer func() {
					if remoteConn != nil {
						remoteConn.Close()
					}
				}()

				c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

				// 构建并发送启动URL到客户端
				launchURL := &url.URL{
					Host:     id,
					Fragment: "1", // TCP模式
				}

				c.mu.Lock()
				_, err = c.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
				c.mu.Unlock()

				if err != nil {
					c.logger.Error("Write failed: %v", err)
					return
				}

				c.logger.Debug("TCP launch signal: %v -> %v", id, c.tunnelTCPConn.RemoteAddr())
				c.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())

				// 交换数据
				bytesReceived, bytesSent, _ := conn.DataExchange(remoteConn, targetConn)

				// 交换完成，广播统计信息
				c.logger.Debug("Exchange complete: TRAFFIC_STATS|TCP_RX=%v|TCP_TX=%v|UDP_RX=0|UDP_TX=0", bytesReceived, bytesSent)
			}(targetConn)
		}
	}
}

// commonUDPLoop 共用UDP请求处理循环
func (c *Common) commonUDPLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// 读取来自目标的UDP数据
			buffer := make([]byte, udpDataBufSize)
			n, clientAddr, err := c.targetUDPConn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			c.logger.Debug("Target connection: %v <-> %v", c.targetUDPConn.LocalAddr(), clientAddr)

			// 从连接池获取连接
			id, remoteConn := c.tunnelPool.ServerGet()
			if remoteConn == nil {
				continue
			}

			c.logger.Debug("Tunnel connection: %v <- active %v", id, c.tunnelPool.Active())

			defer func() {
				if remoteConn != nil {
					remoteConn.Close()
				}
			}()

			c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

			// 使用信号量限制并发数
			c.semaphore <- struct{}{}

			go func(buffer []byte, n int, clientAddr *net.UDPAddr, remoteConn net.Conn) {
				defer func() { <-c.semaphore }()

				// 构建并发送启动URL到客户端
				launchURL := &url.URL{
					Host:     id,
					Fragment: "2", // UDP模式
				}

				c.mu.Lock()
				_, err = c.tunnelTCPConn.Write([]byte(launchURL.String() + "\n"))
				c.mu.Unlock()

				if err != nil {
					c.logger.Error("Write failed: %v", err)
					return
				}

				c.logger.Debug("UDP launch signal: %v -> %v", id, c.tunnelTCPConn.RemoteAddr())
				c.logger.Debug("Starting transfer: %v <-> %v", remoteConn.LocalAddr(), c.targetUDPConn.LocalAddr())

				// 处理UDP/TCP数据交换
				udpToTcp, tcpToUdp, err := conn.DataTransfer(
					c.targetUDPConn,
					remoteConn,
					clientAddr,
					buffer[:n],
					udpDataBufSize,
					tcpReadTimeout,
				)

				if err != nil {
					c.logger.Error("Transfer failed: %v", err)
					return
				}

				// 传输完成，广播统计信息
				c.logger.Debug("Transfer complete: TRAFFIC_STATS|TCP_RX=0|TCP_TX=0|UDP_RX=%v|UDP_TX=%v", udpToTcp, tcpToUdp)
			}(buffer, n, clientAddr, remoteConn)
		}
	}
}

// commonOnce 共用处理单个请求
func (c *Common) commonOnce() {
	for {
		// 等待连接池准备就绪
		if !c.tunnelPool.Ready() {
			time.Sleep(time.Millisecond)
			continue
		}

		select {
		case <-c.ctx.Done():
			return
		case signal := <-c.signalChan:
			// 解析信号URL
			signalURL, err := url.Parse(signal)
			if err != nil {
				c.logger.Error("Parse failed: %v", err)
				continue
			}

			// 处理信号
			switch signalURL.Fragment {
			case "0": // 连接池刷新
				go func() {
					c.tunnelPool.Flush()
					time.Sleep(reportInterval) // 等待连接池刷新完成
					c.logger.Debug("Tunnel pool reset: %v active connections", c.tunnelPool.Active())
				}()
			case "1": // TCP
				go c.commonTCPOnce(signalURL.Host)
			case "2": // UDP
				go c.commonUDPOnce(signalURL.Host)
			default:
			}
		}
	}
}

// commonTCPOnce 共用处理单个TCP请求
func (c *Common) commonTCPOnce(id string) {
	c.logger.Debug("TCP launch signal: %v <- %v", id, c.tunnelTCPConn.RemoteAddr())

	// 从连接池获取连接
	remoteConn := c.tunnelPool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get failed: %v", id)
		return
	}

	c.logger.Debug("Tunnel connection: %v <- active %v", id, c.tunnelPool.Active())

	// 确保连接关闭
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()

	c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

	// 连接到目标TCP地址
	targetConn, err := net.DialTimeout("tcp", c.targetTCPAddr.String(), tcpDialTimeout)
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}

	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()

	c.targetTCPConn = targetConn.(*net.TCPConn)
	c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
	c.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())

	// 交换数据
	bytesReceived, bytesSent, _ := conn.DataExchange(remoteConn, targetConn)

	// 交换完成，广播统计信息
	c.logger.Debug("Exchange complete: TRAFFIC_STATS|TCP_RX=%v|TCP_TX=%v|UDP_RX=0|UDP_TX=0", bytesReceived, bytesSent)
}

// commonUDPOnce 共用处理单个UDP请求
func (c *Common) commonUDPOnce(id string) {
	c.logger.Debug("UDP launch signal: %v <- %v", id, c.tunnelTCPConn.RemoteAddr())

	// 从连接池获取连接
	remoteConn := c.tunnelPool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get failed: %v", id)
		return
	}

	c.logger.Debug("Tunnel connection: %v <- active %v", id, c.tunnelPool.Active())

	// 确保连接关闭
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()

	c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

	// 连接到目标UDP地址
	targetUDPConn, err := net.DialTimeout("udp", c.targetUDPAddr.String(), udpDialTimeout)
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}

	defer func() {
		if targetUDPConn != nil {
			targetUDPConn.Close()
		}
	}()

	c.targetUDPConn = targetUDPConn.(*net.UDPConn)
	c.logger.Debug("Target connection: %v <-> %v", targetUDPConn.LocalAddr(), targetUDPConn.RemoteAddr())

	// 处理UDP/TCP数据交换
	udpToTcp, tcpToUdp, err := conn.DataTransfer(
		c.targetUDPConn,
		remoteConn,
		nil,
		nil,
		udpDataBufSize,
		udpReadTimeout,
	)

	if err != nil {
		c.logger.Error("Transfer failed: %v", err)
		return
	}

	// 交换完成，广播统计信息
	c.logger.Debug("Transfer complete: TRAFFIC_STATS|TCP_RX=0|TCP_TX=0|UDP_RX=%v|UDP_TX=%v", udpToTcp, tcpToUdp)
}
