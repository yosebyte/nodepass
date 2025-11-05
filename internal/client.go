// 内部包，实现客户端模式功能
package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
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
func NewClient(parsedURL *url.URL, logger *logs.Logger) (*Client, error) {
	client := &Client{
		Common: Common{
			logger:     logger,
			signalChan: make(chan string, semaphoreLimit),
			tcpBufferPool: &sync.Pool{
				New: func() any {
					buf := make([]byte, tcpDataBufSize)
					return &buf
				},
			},
			udpBufferPool: &sync.Pool{
				New: func() any {
					buf := make([]byte, udpDataBufSize)
					return &buf
				},
			},
			flushURL: &url.URL{Scheme: "np", Fragment: "f"},
			pingURL:  &url.URL{Scheme: "np", Fragment: "i"},
			pongURL:  &url.URL{Scheme: "np", Fragment: "o"},
		},
		tunnelName: parsedURL.Hostname(),
	}
	if err := client.initConfig(parsedURL); err != nil {
		return nil, fmt.Errorf("newClient: initConfig failed: %w", err)
	}
	client.initRateLimiter()
	return client, nil
}

// Run 管理客户端生命周期
func (c *Client) Run() {
	logInfo := func(prefix string) {
		c.logger.Info("%v: client://%v@%v/%v?min=%v&mode=%v&read=%v&rate=%v&slot=%v&proxy=%v&notcp=%v&noudp=%v",
			prefix, c.tunnelKey, c.tunnelTCPAddr, c.getTargetAddrsString(),
			c.minPoolCapacity, c.runMode, c.readTimeout, c.rateLimit/125000, c.slotLimit, c.proxyProtocol, c.disableTCP, c.disableUDP)
	}
	logInfo("Client started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// 启动客户端服务并处理重启
	go func() {
		for ctx.Err() == nil {
			// 启动客户端
			if err := c.start(); err != nil && err != io.EOF {
				c.logger.Error("Client error: %v", err)
				// 重启客户端
				c.stop()
				select {
				case <-ctx.Done():
					return
				case <-time.After(serviceCooldown):
				}
				logInfo("Client restart")
			}
		}
	}()

	// 监听系统信号以优雅关闭
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
		if err := c.initTunnelListener(); err == nil {
			return c.singleStart()
		} else {
			return fmt.Errorf("start: initTunnelListener failed: %w", err)
		}
	case "2": // 双端模式
		return c.commonStart()
	default: // 自动判断
		if err := c.initTunnelListener(); err == nil {
			c.runMode = "1"
			return c.singleStart()
		} else {
			c.runMode = "2"
			return c.commonStart()
		}
	}
}

// singleStart 启动单端转发模式
func (c *Client) singleStart() error {
	if err := c.singleControl(); err != nil {
		return fmt.Errorf("singleStart: singleControl failed: %w", err)
	}
	return nil
}

// commonStart 启动双端握手模式
func (c *Client) commonStart() error {
	// 与隧道服务端进行握手
	if err := c.tunnelHandshake(); err != nil {
		return fmt.Errorf("commonStart: tunnelHandshake failed: %w", err)
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

	// 判断数据流向
	if c.dataFlow == "+" {
		if err := c.initTargetListener(); err != nil {
			return fmt.Errorf("commonStart: initTargetListener failed: %w", err)
		}
		go c.commonLoop()
	}

	// 启动共用控制
	if err := c.commonControl(); err != nil {
		return fmt.Errorf("commonStart: commonControl failed: %w", err)
	}
	return nil
}

// tunnelHandshake 与隧道服务端进行握手
func (c *Client) tunnelHandshake() error {
	// 建立隧道TCP连接
	tunnelTCPConn, err := net.DialTimeout("tcp", c.tunnelTCPAddr.String(), tcpDialTimeout)
	if err != nil {
		return fmt.Errorf("tunnelHandshake: dialTimeout failed: %w", err)
	}

	c.tunnelTCPConn = tunnelTCPConn.(*net.TCPConn)
	c.bufReader = bufio.NewReader(&conn.TimeoutReader{Conn: c.tunnelTCPConn, Timeout: 3 * reportInterval})
	c.tunnelTCPConn.SetKeepAlive(true)
	c.tunnelTCPConn.SetKeepAlivePeriod(reportInterval)

	// 发送隧道密钥
	_, err = c.tunnelTCPConn.Write(c.encode([]byte(c.tunnelKey)))
	if err != nil {
		return fmt.Errorf("tunnelHandshake: write tunnel key failed: %w", err)
	}

	// 读取隧道URL
	rawTunnelURL, err := c.bufReader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("tunnelHandshake: readBytes failed: %w", err)
	}

	// 解码隧道URL
	tunnelURLData, err := c.decode(rawTunnelURL)
	if err != nil {
		return fmt.Errorf("tunnelHandshake: decode tunnel URL failed: %w", err)
	}

	// 解析隧道URL
	tunnelURL, err := url.Parse(string(tunnelURLData))
	if err != nil {
		return fmt.Errorf("tunnelHandshake: parse tunnel URL failed: %w", err)
	}

	// 更新客户端配置
	if tunnelURL.Host == "" || tunnelURL.Path == "" || tunnelURL.Fragment == "" {
		return net.UnknownNetworkError(tunnelURL.String())
	}
	if max, err := strconv.Atoi(tunnelURL.Host); err != nil {
		return fmt.Errorf("tunnelHandshake: parse max pool capacity failed: %w", err)
	} else {
		c.maxPoolCapacity = max
	}
	c.dataFlow = strings.TrimPrefix(tunnelURL.Path, "/")
	c.tlsCode = tunnelURL.Fragment

	c.logger.Info("Tunnel signal <- : %v <- %v", tunnelURL.String(), c.tunnelTCPConn.RemoteAddr())
	c.logger.Info("Tunnel handshaked: %v <-> %v", c.tunnelTCPConn.LocalAddr(), c.tunnelTCPConn.RemoteAddr())
	return nil
}
