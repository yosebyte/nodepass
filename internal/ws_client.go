package internal

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	nws "github.com/yosebyte/nodepass/internal/websocket"
	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type WSClient struct {
	Common
	tunnelName    string
	bufReader     *bufio.Reader
	signalChan    chan string
	errorCount    int
	wsClient      *nws.Client
	wsPool        *nws.Pool
	supportsWS    bool
}

func NewWSClient(parsedURL *url.URL, tlsConfig *tls.Config, logger *log.Logger) *WSClient {
	common := &Common{
		logger: logger,
	}
	common.getAddress(parsedURL)
	return &WSClient{
		Common:     *common,
		tunnelName: parsedURL.Hostname(),
		signalChan: make(chan string, semaphoreLimit),
	}
}

func (c *WSClient) Start() error {
	c.initContext()
	if err := c.tunnelHandshake(); err != nil {
		return err
	}
	
	// 初始化标准TCP连接池
	c.remotePool = conn.NewClientPool(minPoolCapacity, maxPoolCapacity, c.tlsCode, c.tunnelName, func() (net.Conn, error) {
		return net.DialTCP("tcp", nil, c.remoteAddr)
	})
	
	// 初始化WebSocket连接池（如果服务器支持）
	if c.supportsWS {
		wsServerAddr := c.tunnelAddr.String()
		c.wsPool = nws.NewClientPool(minPoolCapacity, maxPoolCapacity, wsServerAddr, c.getTLSConfig(), c.logger)
	}
	
	go c.remotePool.ClientManager()
	if c.supportsWS {
		go c.wsPool.ClientManager()
	}
	go c.clientLaunch()
	return c.signalQueue()
}

func (c *WSClient) getTLSConfig() *tls.Config {
	// 根据tlsCode创建TLS配置
	if c.tlsCode == "0" {
		return nil
	}
	
	// 创建基本TLS配置
	config := &tls.Config{
		ServerName: c.tunnelName,
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}
	
	return config
}

func (c *WSClient) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.remotePool != nil {
		active := c.remotePool.Active()
		c.remotePool.Close()
		c.logger.Debug("Remote connection closed: active %v", active)
	}
	if c.wsPool != nil {
		active := c.wsPool.Active()
		c.wsPool.Close()
		c.logger.Debug("WebSocket connection closed: active %v", active)
	}
	if c.targetUDPConn != nil {
		c.targetUDPConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetUDPConn.LocalAddr())
	}
	if c.targetTCPConn != nil {
		c.targetTCPConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetTCPConn.LocalAddr())
	}
	if c.tunnelTCPConn != nil {
		c.tunnelTCPConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelTCPConn.LocalAddr())
	}
	for {
		select {
		case <-c.signalChan:
		default:
			return
		}
	}
}

func (c *WSClient) Shutdown(ctx context.Context) error {
	return c.shutdown(ctx, c.Stop)
}

func (c *WSClient) tunnelHandshake() error {
	tunnelTCPConn, err := net.DialTCP("tcp", nil, c.tunnelAddr)
	if err != nil {
		return err
	}
	c.tunnelTCPConn = tunnelTCPConn
	c.bufReader = bufio.NewReader(c.tunnelTCPConn)
	rawTunnelURL, err := c.bufReader.ReadBytes('\n')
	if err != nil {
		return err
	}
	tunnelSignal := strings.TrimSpace(string(rawTunnelURL))
	c.logger.Debug("Tunnel signal <- : %v <- %v", tunnelSignal, c.tunnelTCPConn.RemoteAddr())
	tunnelURL, err := url.Parse(tunnelSignal)
	if err != nil {
		return err
	}
	c.remoteAddr.Port, err = strconv.Atoi(tunnelURL.Host)
	if err != nil {
		return err
	}
	c.tlsCode = tunnelURL.Fragment
	
	// 检查是否支持WebSocket
	c.supportsWS = tunnelURL.Scheme == "ws" || tunnelURL.Scheme == "wss"
	if c.supportsWS {
		c.logger.Info("Server supports WebSocket protocol")
	}
	
	c.logger.Debug("Tunnel connection: %v <-> %v", c.tunnelTCPConn.LocalAddr(), c.tunnelTCPConn.RemoteAddr())
	return nil
}

func (c *WSClient) clientLaunch() {
	for {
		if !c.remotePool.Ready() || (c.supportsWS && !c.wsPool.Ready()) {
			time.Sleep(time.Millisecond)
			continue
		}
		select {
		case <-c.ctx.Done():
			return
		case signal := <-c.signalChan:
			signalURL, err := url.Parse(signal)
			if err != nil {
				c.logger.Error("Parse failed: %v", err)
				continue
			}
			switch signalURL.Fragment {
			case "1":
				go c.clientTCPOnce(signalURL.Host)
			case "2":
				go c.clientUDPOnce(signalURL.Host)
			case "4":
				// WebSocket连接
				if c.supportsWS {
					go c.clientWSOnce(signalURL.Host)
				} else {
					c.logger.Error("WebSocket not supported but received WebSocket signal")
				}
			default:
			}
		}
	}
}

func (c *WSClient) signalQueue() error {
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			rawSignal, err := c.bufReader.ReadBytes('\n')
			if err != nil {
				return err
			}
			signal := strings.TrimSpace(string(rawSignal))
			select {
			case c.signalChan <- signal:
			default:
				c.logger.Debug("Semaphore limit reached: %v", semaphoreLimit)
			}
		}
	}
}

func (c *WSClient) clientWSOnce(id string) {
	c.logger.Debug("WebSocket launch signal: %v <- %v", id, c.tunnelTCPConn.RemoteAddr())
	remoteConn := c.wsPool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get WebSocket connection failed: %v", id)
		c.errorCount++
		if c.errorCount > c.wsPool.Capacity()*1/3 {
			c.logger.Error("Too many WebSocket errors: %v", c.errorCount)
			c.wsPool.Flush()
			c.errorCount = 0
		}
		return
	}
	c.logger.Debug("WebSocket connection: %v <- active %v / %v", id, c.wsPool.Active(), c.wsPool.Capacity())
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	c.logger.Debug("WebSocket connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
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
	c.logger.Debug("Starting WebSocket exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
	bytesReceived, bytesSent, err := conn.DataExchange(remoteConn, targetConn)
	c.AddTCPStats(uint64(bytesReceived), uint64(bytesSent))
	if err == io.EOF {
		c.logger.Debug("WebSocket exchange complete: %v bytes exchanged", bytesReceived+bytesSent)
	} else {
		c.logger.Error("WebSocket exchange complete: %v", err)
	}
}
