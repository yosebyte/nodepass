package internal

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	nquic "github.com/yosebyte/nodepass/internal/quic"
	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type quicClient struct {
	common
	tunnelName string
	bufReader  *bufio.Reader
	signalChan chan string
	errorCount int
	quicClient *nquic.Client
	quicPool   *nquic.Pool
}

func NewQuicClient(parsedURL *url.URL, tlsConfig *tls.Config, logger *log.Logger) *quicClient {
	common := &common{
		logger: logger,
	}
	common.getAddress(parsedURL)
	return &quicClient{
		common:     *common,
		tunnelName: parsedURL.Hostname(),
		signalChan: make(chan string, semaphoreLimit),
	}
}

func (c *quicClient) Start() error {
	c.initContext()
	if err := c.tunnelHandshake(); err != nil {
		return err
	}
	
	// 初始化标准TCP连接池
	c.remotePool = conn.NewClientPool(minPoolCapacity, maxPoolCapacity, c.tlsCode, c.tunnelName, func() (net.Conn, error) {
		return net.DialTCP("tcp", nil, c.remoteAddr)
	})
	
	// 初始化QUIC连接池
	c.quicPool = nquic.NewClientPool(minPoolCapacity, maxPoolCapacity, c.tlsCode, c.remoteAddr.String(), c.logger, c.getTLSConfig())
	
	go c.remotePool.ClientManager()
	go c.quicPool.ClientManager()
	go c.clientLaunch()
	return c.signalQueue()
}

func (c *quicClient) getTLSConfig() *tls.Config {
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

func (c *quicClient) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.remotePool != nil {
		active := c.remotePool.Active()
		c.remotePool.Close()
		c.logger.Debug("Remote connection closed: active %v", active)
	}
	if c.quicPool != nil {
		active := c.quicPool.Active()
		c.quicPool.Close()
		c.logger.Debug("QUIC connection closed: active %v", active)
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

func (c *quicClient) Shutdown(ctx context.Context) error {
	return c.shutdown(ctx, c.Stop)
}

func (c *quicClient) tunnelHandshake() error {
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
	
	// 检查是否支持QUIC
	c.supportsQuic = tunnelURL.Scheme == "quic"
	if c.supportsQuic {
		c.logger.Info("Server supports QUIC protocol")
	}
	
	c.logger.Debug("Tunnel connection: %v <-> %v", c.tunnelTCPConn.LocalAddr(), c.tunnelTCPConn.RemoteAddr())
	return nil
}

func (c *quicClient) clientLaunch() {
	for {
		if !c.remotePool.Ready() || (c.supportsQuic && !c.quicPool.Ready()) {
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
			case "3":
				// QUIC连接
				if c.supportsQuic {
					go c.clientQuicOnce(signalURL.Host)
				} else {
					c.logger.Error("QUIC not supported but received QUIC signal")
				}
			default:
			}
		}
	}
}

func (c *quicClient) signalQueue() error {
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

func (c *quicClient) clientQuicOnce(id string) {
	c.logger.Debug("QUIC launch signal: %v <- %v", id, c.tunnelTCPConn.RemoteAddr())
	remoteConn := c.quicPool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get QUIC connection failed: %v", id)
		c.errorCount++
		if c.errorCount > c.quicPool.Capacity()*1/3 {
			c.logger.Error("Too many QUIC errors: %v", c.errorCount)
			c.quicPool.Flush()
			c.errorCount = 0
		}
		return
	}
	c.logger.Debug("QUIC connection: %v <- active %v / %v", id, c.quicPool.Active(), c.quicPool.Capacity())
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	c.logger.Debug("QUIC connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
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
	c.logger.Debug("Starting QUIC exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
	bytesReceived, bytesSent, err := conn.DataExchange(remoteConn, targetConn)
	c.AddTCPStats(uint64(bytesReceived), uint64(bytesSent))
	if err == io.EOF {
		c.logger.Debug("QUIC exchange complete: %v bytes exchanged", bytesReceived+bytesSent)
	} else {
		c.logger.Error("QUIC exchange complete: %v", err)
	}
}
