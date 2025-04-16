// 内部包，实现客户端模式功能
package internal

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

// Client 实现客户端模式功能
type Client struct {
	Common                   // 继承通用功能
	tunnelName string        // 隧道名称
	bufReader  *bufio.Reader // 缓冲读取器
	signalChan chan string   // 信号通道
	errorCount int           // 错误计数
}

// NewClient 创建新的客户端实例
func NewClient(parsedURL *url.URL, logger *log.Logger) *Client {
	client := &Client{
		Common: Common{
			logger: logger,
		},
		tunnelName: parsedURL.Hostname(),
		signalChan: make(chan string, semaphoreLimit),
	}
	client.getAddress(parsedURL)
	return client
}

// Manage 管理客户端生命周期
func (c *Client) Manage() {
	c.logger.Info("Client started: %v/%v", c.tunnelAddr, c.targetTCPAddr)

	// 启动客户端服务并处理重启
	go func() {
		for {
			if err := c.Start(); err != nil {
				c.logger.Error("Client error: %v", err)
				time.Sleep(serviceCooldown)
				c.Stop()
				c.logger.Info("Client restarted")
			}
		}
	}()

	// 监听系统信号以优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	// 执行关闭过程
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := c.shutdown(shutdownCtx, c.Stop); err != nil {
		c.logger.Error("Client shutdown error: %v", err)
	} else {
		c.logger.Info("Client shutdown complete")
	}
}

// Start 启动客户端服务
func (c *Client) Start() error {
	c.initContext()

	// 与隧道服务器进行握手
	if err := c.tunnelHandshake(); err != nil {
		return err
	}

	// 初始化隧道连接池
	c.tunnelPool = conn.NewClientPool(
		minPoolCapacity,
		maxPoolCapacity,
		minPoolInterval,
		maxPoolInterval,
		c.tlsCode,
		c.tunnelName,
		func() (net.Conn, error) {
			return net.DialTCP("tcp", nil, c.tunnelAddr)
		})

	go c.tunnelPool.ClientManager()
	go c.clientLaunch()
	go c.statsReporter()

	return c.signalQueue()
}

// Stop 停止客户端服务
func (c *Client) Stop() {
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

	// 关闭UDP连接
	if c.targetUDPConn != nil {
		c.targetUDPConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetUDPConn.LocalAddr())
	}

	// 关闭TCP连接
	if c.targetTCPConn != nil {
		c.targetTCPConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetTCPConn.LocalAddr())
	}

	// 关闭隧道连接
	if c.tunnelTCPConn != nil {
		c.tunnelTCPConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelTCPConn.LocalAddr())
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

// tunnelHandshake 与隧道服务器进行握手
func (c *Client) tunnelHandshake() error {
	// 建立隧道TCP连接
	tunnelTCPConn, err := net.DialTCP("tcp", nil, c.tunnelAddr)
	if err != nil {
		return err
	}
	c.tunnelTCPConn = tunnelTCPConn
	c.bufReader = bufio.NewReader(c.tunnelTCPConn)

	// 读取隧道URL
	rawTunnelURL, err := c.bufReader.ReadBytes('\n')
	if err != nil {
		return err
	}
	tunnelSignal := strings.TrimSpace(string(rawTunnelURL))
	c.logger.Debug("Tunnel signal <- : %v <- %v", tunnelSignal, c.tunnelTCPConn.RemoteAddr())

	// 解析隧道URL
	tunnelURL, err := url.Parse(tunnelSignal)
	if err != nil {
		return err
	}

	// 设置TLS代码
	c.tlsCode = tunnelURL.Fragment

	c.logger.Debug("Tunnel handshaked: %v <-> %v", c.tunnelTCPConn.LocalAddr(), c.tunnelTCPConn.RemoteAddr())
	return nil
}

// clientLaunch 启动客户端请求处理
func (c *Client) clientLaunch() {
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

			// 根据信号类型处理TCP或UDP请求
			switch signalURL.Fragment {
			case "1": // TCP
				go c.clientTCPOnce(signalURL.Host)
			case "2": // UDP
				go c.clientUDPOnce(signalURL.Host)
			default:
			}
		}
	}
}

// signalQueue 处理信号队列
func (c *Client) signalQueue() error {
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
				c.logger.Debug("Semaphore limit reached: %v", semaphoreLimit)
			}
		}
	}
}

// clientTCPOnce 处理单个TCP请求
func (c *Client) clientTCPOnce(id string) {
	c.logger.Debug("TCP launch signal: %v <- %v", id, c.tunnelTCPConn.RemoteAddr())

	// 从连接池获取连接
	remoteConn := c.tunnelPool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get failed: %v", id)
		c.errorCount++
		// 错误过多时刷新连接池
		if c.errorCount > c.tunnelPool.Capacity()*1/3 {
			c.logger.Error("Too many errors: %v", c.errorCount)
			c.tunnelPool.Flush()
			c.errorCount = 0
		}
		return
	}

	c.logger.Debug("Tunnel connection: %v <- active %v / %v per %v", id, c.tunnelPool.Active(), c.tunnelPool.Capacity(), c.tunnelPool.Interval())

	// 确保连接关闭
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()

	c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

	// 连接到目标TCP地址
	targetConn, err := net.DialTCP("tcp", nil, c.targetTCPAddr)
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}

	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()

	c.targetTCPConn = targetConn
	c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
	c.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())

	// 交换数据
	bytesReceived, bytesSent, err := conn.DataExchange(remoteConn, targetConn)
	c.AddTCPStats(uint64(bytesReceived), uint64(bytesSent))

	if err == io.EOF {
		c.logger.Debug("Exchange complete: %v bytes exchanged", bytesReceived+bytesSent)
	} else {
		c.logger.Error("Exchange complete: %v", err)
	}
}

// clientUDPOnce 处理单个UDP请求
func (c *Client) clientUDPOnce(id string) {
	c.logger.Debug("UDP launch signal: %v <- %v", id, c.tunnelTCPConn.RemoteAddr())

	// 从连接池获取连接
	remoteConn := c.tunnelPool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get failed: %v", id)
		c.errorCount++
		// 错误过多时刷新连接池
		if c.errorCount > c.tunnelPool.Capacity()*1/3 {
			c.logger.Error("Too many errors: %v", c.errorCount)
			c.tunnelPool.Flush()
			c.errorCount = 0
		}
		return
	}

	c.logger.Debug("Tunnel connection: %v <- active %v / %v per %v", id, c.tunnelPool.Active(), c.tunnelPool.Capacity(), c.tunnelPool.Interval())

	// 确保连接关闭
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()

	c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

	// 读取来自远程连接的数据
	buffer := make([]byte, udpDataBufSize)
	n, err := remoteConn.Read(buffer)
	if err != nil {
		c.logger.Error("Read failed: %v", err)
		return
	}
	c.AddUDPReceived(uint64(n))

	// 连接到目标UDP地址
	targetUDPConn, err := net.DialUDP("udp", nil, c.targetUDPAddr)
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}

	defer func() {
		if targetUDPConn != nil {
			targetUDPConn.Close()
		}
	}()

	c.targetUDPConn = targetUDPConn
	c.logger.Debug("Target connection: %v <-> %v", targetUDPConn.LocalAddr(), targetUDPConn.RemoteAddr())

	// 发送数据到目标
	_, err = targetUDPConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Write failed: %v", err)
		return
	}

	// 设置读取超时并等待响应
	if err := targetUDPConn.SetReadDeadline(time.Now().Add(udpReadTimeout)); err != nil {
		c.logger.Error("Set deadline failed: %v", err)
		return
	}

	n, _, err = targetUDPConn.ReadFromUDP(buffer)
	if err != nil {
		c.logger.Error("Read failed: %v", err)
		return
	}

	// 响应写回远程连接
	_, err = remoteConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Write failed: %v", err)
		return
	}

	c.AddUDPSent(uint64(n))
	bytesReceived, bytesSent := c.GetUDPStats()
	c.logger.Debug("Transfer complete: %v bytes transferred", bytesReceived+bytesSent)
}
