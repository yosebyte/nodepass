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
			errChan:    make(chan error, 2),
			signalChan: make(chan string, semaphoreLimit),
		},
		tunnelName: parsedURL.Hostname(),
	}
	client.getAddress(parsedURL)
	return client
}

// Run 管理客户端生命周期
func (c *Client) Run() {
	c.logger.Info("Client started: %v/%v", c.tunnelAddr, c.targetTCPAddr)

	// 启动客户端服务并处理重启
	go func() {
		for {
			if err := c.start(); err != nil {
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

// start 启动客户端服务
func (c *Client) start() error {
	c.initContext()

	// 通过隧道地址判断是否单端转发或双端握手
	if c.isLocalAddress(c.tunnelTCPAddr.IP) {
		if err := c.initTunnelListener(); err != nil {
			return err
		}

		// 初始化连接池
		c.tunnelPool = pool.NewClientPool(
			minPoolCapacity,
			maxPoolCapacity,
			minPoolInterval,
			maxPoolInterval,
			reportInterval,
			c.tlsCode,
			true,
			c.tunnelName,
			func() (net.Conn, error) {
				return net.DialTCP("tcp", nil, c.targetTCPAddr)
			})

		go c.tunnelPool.ClientManager()

		return c.singleLoop()
	} else {
		if err := c.tunnelHandshake(); err != nil {
			return err
		}

		// 初始化连接池
		c.tunnelPool = pool.NewClientPool(
			minPoolCapacity,
			maxPoolCapacity,
			minPoolInterval,
			maxPoolInterval,
			reportInterval,
			c.tlsCode,
			false,
			c.tunnelName,
			func() (net.Conn, error) {
				return net.DialTCP("tcp", nil, c.tunnelTCPAddr)
			})

		go c.tunnelPool.ClientManager()

		switch c.dataFlow {
		case "-":
			go c.commonOnce()
			go c.commonQueue()
		case "+":
			// 初始化目标监听器
			if err := c.initTargetListener(); err != nil {
				return err
			}
			go c.commonLoop()
		}
		return c.healthCheck()
	}
}

// tunnelHandshake 与隧道服务端进行握手
func (c *Client) tunnelHandshake() error {
	// 建立隧道TCP连接
	tunnelTCPConn, err := net.DialTimeout("tcp", c.tunnelTCPAddr.String(), tcpDialTimeout)
	if err != nil {
		return err
	}
	c.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
	c.bufReader = bufio.NewReader(c.tunnelTCPConn)
	c.tunnelTCPConn.SetKeepAlive(true)
	c.tunnelTCPConn.SetKeepAlivePeriod(reportInterval)

	// 读取隧道URL
	rawTunnelURL, err := c.bufReader.ReadBytes('\n')
	if err != nil {
		return err
	}

	tunnelSignal := string(xor(bytes.TrimSuffix(rawTunnelURL, []byte{'\n'})))
	c.logger.Debug("Tunnel signal <- : %v <- %v", tunnelSignal, c.tunnelTCPConn.RemoteAddr())

	// 解析隧道URL
	tunnelURL, err := url.Parse(tunnelSignal)
	if err != nil {
		return err
	}
	c.dataFlow = tunnelURL.Host
	c.tlsCode = tunnelURL.Fragment

	// 反馈给服务端
	start := time.Now()
	if _, err := c.tunnelTCPConn.Write([]byte{'\n'}); err != nil {
		return err
	}

	// 等待服务端确认握手完成
	_, err = c.tunnelTCPConn.Read(make([]byte, 1))
	if err != nil {
		return err
	}
	c.logger.Event("Tunnel handshaked: %v <-> %v in %vms",
		c.tunnelTCPConn.LocalAddr(), c.tunnelTCPConn.RemoteAddr(), time.Since(start).Milliseconds())
	return nil
}
