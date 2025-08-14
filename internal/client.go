// 内部包，实现客户端模式功能
package internal

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/NodePassProject/conn"
	"github.com/NodePassProject/logs"
	"github.com/NodePassProject/pool"
)

// Client 实现客户端模式功能
type Client struct {
	Common            // 继承共享功能
	tunnelName string // 隧道名称
}

// NewClient 创建新的客户端实例
func NewClient(parsedURL *url.URL, logger *logs.Logger) *Client {
	client := &Client{
		Common: Common{
			logger:     logger,
			semaphore:  make(chan struct{}, semaphoreLimit),
			signalChan: make(chan string, semaphoreLimit),
		},
		tunnelName: parsedURL.Hostname(),
	}
	client.initConfig(parsedURL)
	client.initRateLimiter()
	return client
}

// Run 管理客户端生命周期
func (c *Client) Run() {
	logInfo := func(prefix string) {
		c.logger.Info("%v: %v@%v/%v?min=%v&max=%v&mode=%v&read=%v&rate=%v",
			prefix, c.tunnelKey, c.tunnelAddr, c.targetTCPAddr,
			c.minPoolCapacity, c.maxPoolCapacity, c.runMode, c.readTimeout, c.rateLimit/125000)
	}
	logInfo("Client started")

	// 启动客户端服务并处理重启
	go func() {
		for {
			if err := c.start(); err != nil {
				c.logger.Error("Client error: %v", err)
				time.Sleep(serviceCooldown)
				c.stop()
				logInfo("Client restarted")
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
	if err := c.shutdown(shutdownCtx, c.stop); err != nil {
		c.logger.Error("Client shutdown error: %v", err)
	} else {
		c.logger.Info("Client shutdown complete")
	}
}

// start 启动客户端服务
func (c *Client) start() error {
	// 初始化上下文
	c.initContext()

	// 运行模式判断
	switch c.runMode {
	case "1": // 单端模式
		if err := c.initTunnelListener(); err != nil {
			return err
		}
		return c.singleStart()
	case "2": // 双端模式
		return c.commonStart()
	default: // 自动判断
		if err := c.initTunnelListener(); err == nil {
			return c.singleStart()
		} else {
			return c.commonStart()
		}
	}
}

// singleStart 启动单端转发模式
func (c *Client) singleStart() error {
	return c.singleControl()
}

// commonStart 启动双端握手模式
func (c *Client) commonStart() error {
	// 与隧道服务端进行握手
	if err := c.tunnelHandshake(); err != nil {
		return err
	}

	// 初始化连接池
	c.tunnelPool = pool.NewClientPool(
		c.minPoolCapacity,
		c.maxPoolCapacity,
		minPoolInterval,
		maxPoolInterval,
		reportInterval,
		c.tlsCode,
		c.tunnelName,
		func() (net.Conn, error) {
			return net.DialTimeout("tcp", c.tunnelTCPAddr.String(), tcpDialTimeout)
		})

	go c.tunnelPool.ClientManager()

	if c.dataFlow == "+" {
		// 初始化目标监听器
		if err := c.initTargetListener(); err != nil {
			return err
		}
		go c.commonLoop()
	}

	return c.commonControl()
}

// tunnelHandshake 与隧道服务端进行握手
func (c *Client) tunnelHandshake() error {
	// 建立隧道TCP连接
	tunnelTCPConn, err := net.DialTimeout("tcp", c.tunnelTCPAddr.String(), tcpDialTimeout)
	if err != nil {
		return err
	}

	c.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
	c.bufReader = bufio.NewReader(&conn.TimeoutReader{Conn: c.tunnelTCPConn, Timeout: c.readTimeout})
	c.tunnelTCPConn.SetKeepAlive(true)
	c.tunnelTCPConn.SetKeepAlivePeriod(reportInterval)

	// 发送隧道密钥
	_, err = c.tunnelTCPConn.Write(append(c.xor([]byte(c.tunnelKey)), '\n'))
	if err != nil {
		return err
	}

	// 读取隧道URL
	rawTunnelURL, err := c.bufReader.ReadBytes('\n')
	if err != nil {
		return err
	}

	tunnelSignal := string(c.xor(bytes.TrimSuffix(rawTunnelURL, []byte{'\n'})))

	// 解析隧道URL
	tunnelURL, err := url.Parse(tunnelSignal)
	if err != nil {
		return err
	}

	// 更新客户端配置
	if tunnelURL.Scheme != "" {
		c.dataFlow = tunnelURL.Scheme
	}
	if tunnelURL.Host != "" {
		if max, err := strconv.Atoi(tunnelURL.Host); err == nil {
			c.maxPoolCapacity = max
		}
	}
	if tunnelURL.Fragment != "" {
		c.tlsCode = tunnelURL.Fragment
	}

	c.logger.Info("Tunnel signal <- : %v <- %v", tunnelSignal, c.tunnelTCPConn.RemoteAddr())
	c.logger.Info("Tunnel handshaked: %v <-> %v", c.tunnelTCPConn.LocalAddr(), c.tunnelTCPConn.RemoteAddr())
	return nil
}
