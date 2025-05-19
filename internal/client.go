// 内部包，实现客户端模式功能
package internal

import (
	"bufio"
	"context"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
				c.stop()
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
	if err := c.shutdown(shutdownCtx, c.stop); err != nil {
		c.logger.Error("Client shutdown error: %v", err)
	} else {
		c.logger.Info("Client shutdown complete")
	}
}

// Start 启动客户端服务
func (c *Client) Start() error {
	c.initContext()

	// 与隧道服务端进行握手
	if err := c.tunnelHandshake(); err != nil {
		return err
	}

	// 初始化隧道连接池
	c.tunnelPool = pool.NewClientPool(
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

	switch c.dataFlow {
	case "-":
		go c.commonOnce()
	case "+":
		// 初始化目标监听器
		if err := c.initTargetListener(); err != nil {
			return err
		}
		go c.commonLoop()
	}
	return c.commonQueue()
}

// tunnelHandshake 与隧道服务端进行握手
func (c *Client) tunnelHandshake() error {
	// 建立隧道TCP连接
	tunnelTCPConn, err := net.DialTimeout("tcp", c.tunnelAddr.String(), tcpDialTimeout)
	if err != nil {
		return err
	}
	c.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
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

	// 设置数据流向
	c.dataFlow = tunnelURL.Host

	// 设置TLS代码
	c.tlsCode = tunnelURL.Fragment

	c.logger.Debug("Tunnel handshaked: %v <-> %v", c.tunnelTCPConn.LocalAddr(), c.tunnelTCPConn.RemoteAddr())
	return nil
}
