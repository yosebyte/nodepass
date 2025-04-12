package internal

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type client struct {
	common
	tunnelName string
	bufReader  *bufio.Reader
	signalChan chan string
	errorCount int
}

func NewClient(parsedURL *url.URL, logger *log.Logger) *client {
	common := &common{
		logger: logger,
	}
	common.getAddress(parsedURL)
	return &client{
		common:     *common,
		tunnelName: parsedURL.Hostname(),
		signalChan: make(chan string, semaphoreLimit),
	}
}

func (c *client) Start() error {
	c.initContext()
	if err := c.tunnelHandshake(); err != nil {
		return err
	}
	c.remotePool = conn.NewClientPool(minPoolCapacity, maxPoolCapacity, c.tlsCode, c.tunnelName, func() (net.Conn, error) {
		return net.DialTCP("tcp", nil, c.remoteAddr)
	})
	go c.remotePool.ClientManager()
	go c.clientLaunch()
	return c.signalQueue()
}

func (c *client) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.remotePool != nil {
		active := c.remotePool.Active()
		c.remotePool.Close()
		c.logger.Debug("Remote connection closed: active %v", active)
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

func (c *client) Shutdown(ctx context.Context) error {
	return c.shutdown(ctx, c.Stop)
}

func (c *client) tunnelHandshake() error {
	// 建立TCP连接
	tunnelTCPConn, err := net.DialTCP("tcp", nil, c.tunnelAddr)
	if err != nil {
		return err
	}
	c.tunnelTCPConn = tunnelTCPConn
	c.bufReader = bufio.NewReader(c.tunnelTCPConn)
	
	// 创建安全管理器
	securityManager, err := NewSecurityManager(c.logger, nil)
	if err != nil {
		return fmt.Errorf("创建安全管理器失败: %v", err)
	}
	
	// 加载受信任证书
	if err := securityManager.LoadTrustedCertificates(); err != nil {
		c.logger.Warn("加载受信任证书失败: %v", err)
	}
	
	// 执行安全握手
	handshakeData, err := securityManager.SecureHandshake(c.tunnelTCPConn, false)
	if err != nil {
		return fmt.Errorf("安全握手失败: %v", err)
	}
	
	// 从握手数据中提取必要信息
	c.remoteAddr.Port = handshakeData.Port
	c.tlsCode = handshakeData.TLSMode
	
	// 检查服务器支持的协议
	for _, proto := range handshakeData.SupportedProtos {
		if proto == "quic" {
			c.supportsQuic = true
		}
		if proto == "websocket" {
			c.supportsWS = true
		}
	}
	
	c.logger.Debug("安全握手完成: 服务器=%s, TLS模式=%s, 端口=%d", 
		handshakeData.ServerName, c.tlsCode, c.remoteAddr.Port)
	c.logger.Debug("Tunnel connection: %v <-> %v", c.tunnelTCPConn.LocalAddr(), c.tunnelTCPConn.RemoteAddr())
	
	// 存储安全管理器以供后续使用
	c.securityManager = securityManager
	
	return nil
}

func (c *client) clientLaunch() {
	for {
		if !c.remotePool.Ready() {
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
			default:
			}
		}
	}
}

func (c *client) signalQueue() error {
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

func (c *client) clientTCPOnce(id string) {
	c.logger.Debug("TCP launch signal: %v <- %v", id, c.tunnelTCPConn.RemoteAddr())
	remoteConn := c.remotePool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get failed: %v", id)
		c.errorCount++
		if c.errorCount > c.remotePool.Capacity()*1/3 {
			c.logger.Error("Too many errors: %v", c.errorCount)
			c.remotePool.Flush()
			c.errorCount = 0
		}
		return
	}
	c.logger.Debug("Remote connection: %v <- active %v / %v", id, c.remotePool.Active(), c.remotePool.Capacity())
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	c.logger.Debug("Remote connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
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
	bytesReceived, bytesSent, err := conn.DataExchange(remoteConn, targetConn)
	c.AddTCPStats(uint64(bytesReceived), uint64(bytesSent))
	if err == io.EOF {
		c.logger.Debug("Exchange complete: %v bytes exchanged", bytesReceived+bytesSent)
	} else {
		c.logger.Error("Exchange complete: %v", err)
	}
}

func (c *client) clientUDPOnce(id string) {
	c.logger.Debug("UDP launch signal: %v <- %v", id, c.tunnelTCPConn.RemoteAddr())
	remoteConn := c.remotePool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get failed: %v", id)
		c.errorCount++
		if c.errorCount > c.remotePool.Capacity()*1/3 {
			c.logger.Error("Too many errors: %v", c.errorCount)
			c.remotePool.Flush()
			c.errorCount = 0
		}
		return
	}
	c.logger.Debug("Remote connection: %v <- active %v / %v", id, c.remotePool.Active(), c.remotePool.Capacity())
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	c.logger.Debug("Remote connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
	buffer := make([]byte, udpDataBufSize)
	n, err := remoteConn.Read(buffer)
	if err != nil {
		c.logger.Error("Read failed: %v", err)
		return
	}
	c.AddUDPReceived(uint64(n))
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
	_, err = targetUDPConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Write failed: %v", err)
		return
	}
	if err := targetUDPConn.SetReadDeadline(time.Now().Add(udpReadTimeout)); err != nil {
		c.logger.Error("Set deadline failed: %v", err)
		return
	}
	n, _, err = targetUDPConn.ReadFromUDP(buffer)
	if err != nil {
		c.logger.Error("Read failed: %v", err)
		return
	}
	_, err = remoteConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Write failed: %v", err)
		return
	}
	c.AddUDPSent(uint64(n))
	bytesReceived, bytesSent := c.GetUDPStats()
	c.logger.Debug("Transfer complete: %v bytes transferred", bytesReceived+bytesSent)
}
