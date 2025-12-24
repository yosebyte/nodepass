// 内部包，实现客户端模式功能
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/NodePassProject/logs"
	"github.com/NodePassProject/nph2"
	"github.com/NodePassProject/npws"
	"github.com/NodePassProject/pool"
	"github.com/NodePassProject/quic"
)

// Client 实现客户端模式功能
type Client struct{ Common }

// NewClient 创建新的客户端实例
func NewClient(parsedURL *url.URL, logger *logs.Logger) (*Client, error) {
	client := &Client{
		Common: Common{
			parsedURL:  parsedURL,
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
	}
	if err := client.initConfig(); err != nil {
		return nil, fmt.Errorf("newClient: initConfig failed: %w", err)
	}
	client.initRateLimiter()
	return client, nil
}

// Run 管理客户端生命周期
func (c *Client) Run() {
	logInfo := func(prefix string) {
		c.logger.Info("%v: client://%v@%v/%v?dns=%v&sni=%v&min=%v&mode=%v&dial=%v&read=%v&rate=%v&slot=%v&proxy=%v&block=%v&notcp=%v&noudp=%v",
			prefix, c.tunnelKey, c.tunnelTCPAddr, c.getTargetAddrsString(), c.dnsCacheTTL, c.serverName, c.minPoolCapacity,
			c.runMode, c.dialerIP, c.readTimeout, c.rateLimit/125000, c.slotLimit,
			c.proxyProtocol, c.blockProtocol, c.disableTCP, c.disableUDP)
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
	// 发起隧道握手
	c.logger.Info("Pending tunnel handshake...")
	c.handshakeStart = time.Now()
	if err := c.tunnelHandshake(); err != nil {
		return fmt.Errorf("commonStart: tunnelHandshake failed: %w", err)
	}

	// 初始化连接池
	if err := c.initTunnelPool(); err != nil {
		return fmt.Errorf("commonStart: initTunnelPool failed: %w", err)
	}

	// 设置控制连接
	c.logger.Info("Getting tunnel pool ready...")
	if err := c.setControlConn(); err != nil {
		return fmt.Errorf("commonStart: setControlConn failed: %w", err)
	}

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

// initTunnelPool 初始化隧道连接池
func (c *Client) initTunnelPool() error {
	switch c.poolType {
	case "0":
		tcpPool := pool.NewClientPool(
			c.minPoolCapacity,
			c.maxPoolCapacity,
			minPoolInterval,
			maxPoolInterval,
			reportInterval,
			c.tlsCode,
			c.serverName,
			func() (net.Conn, error) {
				tcpAddr, err := c.getTunnelTCPAddr()
				if err != nil {
					return nil, err
				}
				return net.DialTimeout("tcp", tcpAddr.String(), tcpDialTimeout)
			})
		go tcpPool.ClientManager()
		c.tunnelPool = tcpPool
	case "1":
		quicPool := quic.NewClientPool(
			c.minPoolCapacity,
			c.maxPoolCapacity,
			minPoolInterval,
			maxPoolInterval,
			reportInterval,
			c.tlsCode,
			c.serverName,
			func() (string, error) {
				udpAddr, err := c.getTunnelUDPAddr()
				if err != nil {
					return "", err
				}
				return udpAddr.String(), nil
			})
		go quicPool.ClientManager()
		c.tunnelPool = quicPool
	case "2":
		websocketPool := npws.NewClientPool(
			c.minPoolCapacity,
			c.maxPoolCapacity,
			minPoolInterval,
			maxPoolInterval,
			reportInterval,
			c.tlsCode,
			c.tunnelAddr)
		go websocketPool.ClientManager()
		c.tunnelPool = websocketPool
	case "3":
		http2Pool := nph2.NewClientPool(
			c.minPoolCapacity,
			c.maxPoolCapacity,
			minPoolInterval,
			maxPoolInterval,
			reportInterval,
			c.tlsCode,
			c.serverName,
			func() (string, error) {
				tcpAddr, err := c.getTunnelTCPAddr()
				if err != nil {
					return "", err
				}
				return tcpAddr.String(), nil
			})
		go http2Pool.ClientManager()
		c.tunnelPool = http2Pool
	default:
		return fmt.Errorf("initTunnelPool: unknown pool type: %s", c.poolType)
	}
	return nil
}

// tunnelHandshake 与隧道服务端进行握手
func (c *Client) tunnelHandshake() error {
	scheme := "http"
	if c.serverPort == "443" {
		scheme = "https"
	}

	// 构建请求
	req, _ := http.NewRequest(http.MethodGet, scheme+"://"+c.tunnelAddr+"/", nil)
	req.Host = c.serverName
	req.Header.Set("Authorization", "Bearer "+c.generateAuthToken())

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("tunnelHandshake: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tunnelHandshake: status %d", resp.StatusCode)
	}

	// 解析配置
	var config struct {
		Flow string `json:"flow"`
		Max  int    `json:"max"`
		TLS  string `json:"tls"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return fmt.Errorf("tunnelHandshake: %w", err)
	}

	// 更新配置
	c.dataFlow = config.Flow
	c.maxPoolCapacity = config.Max
	c.tlsCode = config.TLS
	c.poolType = config.Type
	if c.tlsCode == "1" || c.tlsCode == "2" {
		c.certVerified = make(chan struct{})
	}

	c.logger.Info("Loading tunnel config: FLOW=%v|MAX=%v|TLS=%v|TYPE=%v",
		c.dataFlow, c.maxPoolCapacity, c.tlsCode, c.poolType)
	return nil
}
