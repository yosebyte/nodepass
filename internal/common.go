// 内部包，提供共享功能
package internal

import (
	"bufio"
	"bytes"
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
	mu               sync.Mutex         // 互斥锁
	tlsCode          string             // TLS模式代码
	dataFlow         string             // 数据流向
	logger           *logs.Logger       // 日志记录器
	tunnelAddr       string             // 隧道地址字符串
	tunnelTCPAddr    *net.TCPAddr       // 隧道TCP地址
	tunnelUDPAddr    *net.UDPAddr       // 隧道UDP地址
	targetAddr       string             // 目标地址字符串
	targetTCPAddr    *net.TCPAddr       // 目标TCP地址
	targetUDPAddr    *net.UDPAddr       // 目标UDP地址
	targetListener   *net.TCPListener   // 目标监听器
	tunnelListener   net.Listener       // 隧道监听器
	tunnelTCPConn    *net.TCPConn       // 隧道TCP连接
	tunnelUDPConn    *net.UDPConn       // 隧道UDP连接
	targetTCPConn    *net.TCPConn       // 目标TCP连接
	targetUDPConn    *net.UDPConn       // 目标UDP连接
	targetUDPSession sync.Map           // 目标UDP会话
	tunnelPool       *pool.Pool         // 隧道连接池
	semaphore        chan struct{}      // 信号量通道
	bufReader        *bufio.Reader      // 缓冲读取器
	signalChan       chan string        // 信号通道
	errChan          chan error         // 错误通道
	ctx              context.Context    // 上下文
	cancel           context.CancelFunc // 取消函数
}

// 配置变量，可通过环境变量调整
var (
	semaphoreLimit  = getEnvAsInt("NP_SEMAPHORE_LIMIT", 1024)                 // 信号量限制
	minPoolCapacity = getEnvAsInt("NP_MIN_POOL_CAPACITY", 16)                 // 最小池容量
	maxPoolCapacity = getEnvAsInt("NP_MAX_POOL_CAPACITY", 1024)               // 最大池容量
	udpDataBufSize  = getEnvAsInt("NP_UDP_DATA_BUF_SIZE", 8192)               // UDP数据缓冲区大小
	udpReadTimeout  = getEnvAsDuration("NP_UDP_READ_TIMEOUT", 15*time.Second) // UDP读取超时
	udpDialTimeout  = getEnvAsDuration("NP_UDP_DIAL_TIMEOUT", 15*time.Second) // UDP拨号超时
	tcpReadTimeout  = getEnvAsDuration("NP_TCP_READ_TIMEOUT", 15*time.Second) // TCP读取超时
	tcpDialTimeout  = getEnvAsDuration("NP_TCP_DIAL_TIMEOUT", 15*time.Second) // TCP拨号超时
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

// xor 对数据进行异或处理
func xor(data []byte) []byte {
	for i := range data {
		data[i] ^= byte(128)
	}
	return data
}

// getAddress 解析和设置地址信息
func (c *Common) getAddress(parsedURL *url.URL) {
	// 解析隧道地址
	c.tunnelAddr = parsedURL.Host

	// 解析隧道TCP地址
	if tunnelTCPAddr, err := net.ResolveTCPAddr("tcp", c.tunnelAddr); err == nil {
		c.tunnelTCPAddr = tunnelTCPAddr
	} else {
		c.logger.Error("Resolve failed: %v", err)
	}

	// 解析隧道UDP地址
	if tunnelUDPAddr, err := net.ResolveUDPAddr("udp", c.tunnelAddr); err == nil {
		c.tunnelUDPAddr = tunnelUDPAddr
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
		if targetListener != nil {
			targetListener.Close()
		}
		return err
	}
	c.targetListener = targetListener

	// 初始化目标UDP监听器
	targetUDPConn, err := net.ListenUDP("udp", c.targetUDPAddr)
	if err != nil {
		if targetUDPConn != nil {
			targetUDPConn.Close()
		}
		return err
	}
	c.targetUDPConn = targetUDPConn

	return nil
}

// initTunnelListener 初始化隧道监听器
func (c *Common) initTunnelListener() error {
	// 初始化隧道TCP监听器
	tunnelListener, err := net.ListenTCP("tcp", c.tunnelTCPAddr)
	if err != nil {
		if tunnelListener != nil {
			tunnelListener.Close()
		}
		return err
	}
	c.tunnelListener = tunnelListener

	// 初始化隧道UDP监听器
	tunnelUDPConn, err := net.ListenUDP("udp", c.tunnelUDPAddr)
	if err != nil {
		if tunnelUDPConn != nil {
			tunnelUDPConn.Close()
		}
		return err
	}
	c.tunnelUDPConn = tunnelUDPConn

	return nil
}

// isLocalAddress 检查IP地址是否为本机接口地址之一
func (c *Common) isLocalAddress(ip net.IP) bool {
	// 处理未指定的IP地址
	if ip.IsUnspecified() || ip == nil {
		return true
	}

	// 获取本机所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		c.logger.Error("Get interfaces failed: %v", err)
		return false
	}

	// 遍历所有网络接口
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 遍历接口的所有地址
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.Equal(ip) {
					return true
				}
			case *net.IPAddr:
				if v.IP.Equal(ip) {
					return true
				}
			}
		}
	}
	return false
}

// stop 共用停止服务
func (c *Common) stop() {
	// 取消上下文
	if c.cancel != nil {
		c.cancel()
	}

	// 关闭隧道连接池
	if c.tunnelPool != nil {
		active := c.tunnelPool.Active()
		c.tunnelPool.Close()
		c.logger.Debug("Tunnel connection closed: active %v", active)
	}

	// 清理目标UDP会话
	c.targetUDPSession.Range(func(key, value any) bool {
		if conn, ok := value.(*net.UDPConn); ok {
			conn.Close()
		}
		c.targetUDPSession.Delete(key)
		return true
	})

	// 关闭目标UDP连接
	if c.targetUDPConn != nil {
		c.targetUDPConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetUDPConn.LocalAddr())
	}

	// 关闭目标TCP连接
	if c.targetTCPConn != nil {
		c.targetTCPConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetTCPConn.LocalAddr())
	}

	// 关闭隧道UDP连接
	if c.tunnelUDPConn != nil {
		c.tunnelUDPConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelUDPConn.LocalAddr())
	}

	// 关闭隧道TCP连接
	if c.tunnelTCPConn != nil {
		c.tunnelTCPConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelTCPConn.LocalAddr())
	}

	// 关闭目标监听器
	if c.targetListener != nil {
		c.targetListener.Close()
		c.logger.Debug("Target listener closed: %v", c.targetListener.Addr())
	}

	// 关闭隧道监听器
	if c.tunnelListener != nil {
		c.tunnelListener.Close()
		c.logger.Debug("Tunnel listener closed: %v", c.tunnelListener.Addr())
	}

	// 清空信号通道
	for {
		select {
		case <-c.signalChan:
		default:
			return
		}
	}
}

// shutdown 共用优雅关闭
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
			signal := string(xor(bytes.TrimSuffix(rawSignal, []byte{'\n'})))

			// 将信号发送到通道
			select {
			case c.signalChan <- signal:
			default:
				c.logger.Debug("Queue limit reached: %v", semaphoreLimit)
			}
		}
	}
}

// healthCheck 共用健康度检查
func (c *Common) healthCheck() error {
	flushURL := &url.URL{Fragment: "0"} // 连接池刷新信号
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			if !c.mu.TryLock() {
				continue
			}

			// 连接池健康度检查
			if c.tunnelPool.ErrorCount() > c.tunnelPool.Active()/2 {
				// 发送刷新信号到对端
				_, err := c.tunnelTCPConn.Write(append(xor([]byte(flushURL.String())), '\n'))
				if err != nil {
					c.mu.Unlock()
					return err
				}
				c.tunnelPool.Flush()
				time.Sleep(reportInterval) // 等待连接池刷新完成
				c.logger.Debug("Tunnel pool reset: %v active connections", c.tunnelPool.Active())
			} else {
				// 发送普通心跳包
				_, err := c.tunnelTCPConn.Write([]byte("\n"))
				if err != nil {
					c.mu.Unlock()
					return err
				}
			}

			c.mu.Unlock()
			time.Sleep(reportInterval)
		}
	}
}

// commonLoop 共用处理循环
func (c *Common) commonLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// 等待连接池准备就绪
			if c.tunnelPool.Ready() {
				go c.commonTCPLoop()
				go c.commonUDPLoop()
				return
			}
			time.Sleep(time.Millisecond)
		}
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
					c.tunnelPool.AddError()
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
				_, err = c.tunnelTCPConn.Write(append(xor([]byte(launchURL.String())), '\n'))
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
				c.logger.Event("Exchange complete: TRAFFIC_STATS|TCP_RX=%v|TCP_TX=%v|UDP_RX=0|UDP_TX=0", bytesReceived, bytesSent)
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
				c.tunnelPool.AddError()
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
				_, err = c.tunnelTCPConn.Write(append(xor([]byte(launchURL.String())), '\n'))
				c.mu.Unlock()

				if err != nil {
					c.logger.Error("Write failed: %v", err)
					return
				}

				c.logger.Debug("UDP launch signal: %v -> %v", id, c.tunnelTCPConn.RemoteAddr())
				c.logger.Debug("Starting transfer: %v <-> %v", remoteConn.LocalAddr(), c.targetUDPConn.LocalAddr())

				// 处理UDP/TCP数据传输
				rx, tx, _ := conn.DataTransfer(
					c.targetUDPConn,
					remoteConn,
					clientAddr,
					buffer[:n],
					udpDataBufSize,
					tcpReadTimeout,
				)

				// 传输完成，广播统计信息
				c.logger.Event("Transfer complete: TRAFFIC_STATS|TCP_RX=0|TCP_TX=0|UDP_RX=%v|UDP_TX=%v", rx, tx)
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
				// 健康检查或无效信号
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
	c.logger.Event("Exchange complete: TRAFFIC_STATS|TCP_RX=%v|TCP_TX=%v|UDP_RX=0|UDP_TX=0", bytesReceived, bytesSent)
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
	targetConn, err := net.DialTimeout("udp", c.targetUDPAddr.String(), udpDialTimeout)
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}

	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()

	c.targetUDPConn = targetConn.(*net.UDPConn)
	c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
	c.logger.Debug("Starting transfer: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())

	// 处理UDP/TCP数据传输
	rx, tx, _ := conn.DataTransfer(
		c.targetUDPConn,
		remoteConn,
		nil,
		nil,
		udpDataBufSize,
		udpReadTimeout,
	)

	// 传输完成，广播统计信息
	c.logger.Event("Transfer complete: TRAFFIC_STATS|TCP_RX=0|TCP_TX=0|UDP_RX=%v|UDP_TX=%v", rx, tx)
}

// singleLoop 单端转发处理循环
func (c *Common) singleLoop() error {
	for {
		select {
		case <-c.ctx.Done():
			return context.Canceled
		default:
			go func() {
				c.errChan <- c.singleTCPLoop()
			}()
			go func() {
				c.errChan <- c.singleUDPLoop()
			}()
			return <-c.errChan
		}
	}
}

// singleTCPLoop 单端转发TCP处理循环
func (c *Common) singleTCPLoop() error {
	for {
		select {
		case <-c.ctx.Done():
			return context.Canceled
		default:
			// 接受来自隧道的TCP连接
			tunnelConn, err := c.tunnelListener.Accept()
			if err != nil {
				continue
			}

			defer func() {
				if tunnelConn != nil {
					tunnelConn.Close()
				}
			}()

			c.tunnelTCPConn = tunnelConn.(*net.TCPConn)
			c.logger.Debug("Tunnel connection: %v <-> %v", tunnelConn.LocalAddr(), tunnelConn.RemoteAddr())

			// 使用信号量限制并发数
			c.semaphore <- struct{}{}

			go func(tunnelConn net.Conn) {
				defer func() { <-c.semaphore }()

				// 从连接池中获取连接
				targetConn := c.tunnelPool.ClientGet("")
				if targetConn == nil {
					c.logger.Error("Get failed: no target connection available")
					return
				}

				c.logger.Debug("Target connection: active %v / %v per %v", c.tunnelPool.Active(), c.tunnelPool.Capacity(), c.tunnelPool.Interval())

				defer func() {
					if targetConn != nil {
						targetConn.Close()
					}
				}()

				c.targetTCPConn = targetConn.(*net.TCPConn)
				c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
				c.logger.Debug("Starting exchange: %v <-> %v", tunnelConn.LocalAddr(), targetConn.LocalAddr())

				// 交换数据
				bytesReceived, bytesSent, _ := conn.DataExchange(tunnelConn, targetConn)

				// 交换完成，广播统计信息
				c.logger.Event("Exchange complete: TRAFFIC_STATS|TCP_RX=%v|TCP_TX=%v|UDP_RX=0|UDP_TX=0", bytesReceived, bytesSent)
			}(tunnelConn)
		}
	}
}

// singleUDPLoop 单端转发UDP处理循环
func (c *Common) singleUDPLoop() error {
	for {
		select {
		case <-c.ctx.Done():
			return context.Canceled
		default:
			buffer := make([]byte, udpDataBufSize)

			// 读取来自隧道的UDP数据
			rx, clientAddr, err := c.tunnelUDPConn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			c.logger.Debug("Tunnel connection: %v <-> %v", c.tunnelUDPConn.LocalAddr(), clientAddr)

			var targetConn *net.UDPConn
			sessionKey := clientAddr.String()

			// 获取或创建目标UDP会话
			if session, ok := c.targetUDPSession.Load(sessionKey); ok {
				// 复用现有会话
				targetConn = session.(*net.UDPConn)
				c.logger.Debug("Using UDP session: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
			} else {
				// 创建新的会话
				session, err := net.DialTimeout("udp", c.targetUDPAddr.String(), udpDialTimeout)
				if err != nil {
					c.logger.Error("Dial failed: %v", err)
					continue
				}
				c.targetUDPSession.Store(sessionKey, session)

				targetConn = session.(*net.UDPConn)
				c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())

				// 使用信号量限制并发数
				c.semaphore <- struct{}{}

				go func(targetConn *net.UDPConn, clientAddr *net.UDPAddr, sessionKey string) {
					defer func() { <-c.semaphore }()

					buffer := make([]byte, udpDataBufSize)

					for {
						select {
						case <-c.ctx.Done():
							return
						default:
							// 设置UDP读取超时
							if err := targetConn.SetReadDeadline(time.Now().Add(udpReadTimeout)); err != nil {
								c.logger.Error("SetReadDeadline failed: %v", err)
								c.targetUDPSession.Delete(sessionKey)
								targetConn.Close()
								return
							}

							// 从UDP读取响应
							n, err := targetConn.Read(buffer)
							if err != nil {
								// 检查是否为超时错误
								if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
									c.logger.Debug("UDP session abort: %v", err)
								} else {
									c.logger.Error("Read failed: %v", err)
								}
								c.targetUDPSession.Delete(sessionKey)
								targetConn.Close()
								return
							}

							// 将响应写回隧道UDP连接
							tx, err := c.tunnelUDPConn.WriteToUDP(buffer[:n], clientAddr)
							if err != nil {
								c.logger.Error("WriteToUDP failed: %v", err)
								c.targetUDPSession.Delete(sessionKey)
								targetConn.Close()
								return
							}
							// 传输完成，广播统计信息
							c.logger.Event("Transfer complete: TRAFFIC_STATS|TCP_RX=0|TCP_TX=0|UDP_RX=0|UDP_TX=%v", tx)
						}
					}
				}(targetConn, clientAddr, sessionKey)
			}

			// 将初始数据发送到目标UDP连接
			c.logger.Debug("Starting transfer: %v <-> %v", targetConn.LocalAddr(), c.tunnelUDPConn.LocalAddr())
			_, err = targetConn.Write(buffer[:rx])
			if err != nil {
				c.logger.Error("Write failed: %v", err)
				c.targetUDPSession.Delete(sessionKey)
				targetConn.Close()
				return err
			}

			// 传输完成，广播统计信息
			c.logger.Event("Transfer complete: TRAFFIC_STATS|TCP_RX=0|TCP_TX=0|UDP_RX=%v|UDP_TX=0", rx)
		}
	}
}
