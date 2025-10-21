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
			cleanURL: &url.URL{Scheme: "np", Fragment: "c"},
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
		c.logger.Info("%v: client://%v@%v/%v?min=%v&mode=%v&read=%v&rate=%v&slot=%v&proxy=%v",
			prefix, c.tunnelKey, c.tunnelTCPAddr, c.getTargetAddrsString(),
			c.minPoolCapacity, c.runMode, c.readTimeout, c.rateLimit/125000, c.slotLimit, c.proxyProtocol)
	}
	logInfo("Client started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// 启动客户端服务并处理重启
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
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
				logInfo("Client restarting")
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
		if err := c.initTunnelListener(); err != nil {
			return fmt.Errorf("start: initTunnelListener failed: %w", err)
		}
		return c.singleStart()
	case "2": // 双端模式
		return c.commonStart()
	case "3": // NAT穿透模式
		return c.natStart()
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

	if c.dataFlow == "+" {
		// 初始化目标监听器
		if err := c.initTargetListener(); err != nil {
			return fmt.Errorf("commonStart: initTargetListener failed: %w", err)
		}
		go c.commonLoop()
	}
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

// natStart NAT穿透模式：STUN发现外部地址，本地监听并转发连接
func (c *Client) natStart() error {
	udpConn, err := net.DialTimeout("udp", c.tunnelTCPAddr.String(), udpDialTimeout)
	if err != nil {
		return fmt.Errorf("natStart: STUN dial failed: %w", err)
	}
	defer udpConn.Close()

	// 构造STUN请求
	req := make([]byte, 20)
	req[0], req[1] = 0x00, 0x01
	req[4], req[5], req[6], req[7] = 0x21, 0x12, 0xA4, 0x42
	for i := 8; i < 20; i++ {
		req[i] = byte(time.Now().UnixNano() % 256)
	}

	// 发送STUN请求
	if _, err := udpConn.Write(req); err != nil {
		return fmt.Errorf("natStart: STUN write failed: %w", err)
	}

	// 解析STUN响应
	resp := make([]byte, 1500)
	udpConn.SetReadDeadline(time.Now().Add(udpReadTimeout))
	n, err := udpConn.Read(resp)
	if err != nil {
		return fmt.Errorf("natStart: STUN read failed: %w", err)
	}
	if n < 20 || resp[0] != 0x01 || resp[1] != 0x01 {
		return fmt.Errorf("natStart: invalid STUN response")
	}

	// 查找映射地址
	magic := []byte{0x21, 0x12, 0xA4, 0x42}
	var extAddr string
	var localPort int
	for pos := 20; pos+4 <= n; {
		typ := uint16(resp[pos])<<8 | uint16(resp[pos+1])
		sz := uint16(resp[pos+2])<<8 | uint16(resp[pos+3])
		if typ == 0x0020 && pos+4+int(sz) <= n && sz >= 8 && resp[pos+5] == 0x01 {
			port := (uint16(resp[pos+6])<<8 | uint16(resp[pos+7])) ^ 0x2112
			a := make([]byte, 4)
			for i := 0; i < 4; i++ {
				a[i] = resp[pos+8+i] ^ magic[i]
			}
			extAddr = fmt.Sprintf("%d.%d.%d.%d:%d", a[0], a[1], a[2], a[3], port)
			if udpAddr, ok := udpConn.LocalAddr().(*net.UDPAddr); ok {
				localPort = udpAddr.Port
			}
			break
		}
		pos += 4 + int(sz)
		if sz%4 != 0 {
			pos += 4 - int(sz%4)
		}
	}
	if extAddr == "" {
		return fmt.Errorf("natStart: address not found in STUN response")
	}

	// 本地监听TCP连接
	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return fmt.Errorf("natStart: listen TCP failed: %w", err)
	}
	defer tcpListener.Close()

	c.logger.Info("NAT: external endpoint %v -> %v", extAddr, c.getTargetAddrsString())

	for c.ctx.Err() == nil {
		incomingConn, err := tcpListener.Accept()
		if err != nil {
			if c.ctx.Err() != nil || err == net.ErrClosed {
				return nil
			}
			c.logger.Error("NAT accept failed: %v", err)
			select {
			case <-c.ctx.Done():
				return nil
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		go func(incomingConn net.Conn) {
			defer incomingConn.Close()

			if !c.tryAcquireSlot(false) {
				return
			}
			defer c.releaseSlot(false)

			outgoingConn, err := c.dialWithRotation("tcp", tcpDialTimeout)
			if err != nil {
				c.logger.Error("NAT dial failed: %v", err)
				return
			}
			defer outgoingConn.Close()

			buf1, buf2 := c.getTCPBuffer(), c.getTCPBuffer()
			defer c.putTCPBuffer(buf1)
			defer c.putTCPBuffer(buf2)

			conn.DataExchange(incomingConn, outgoingConn, c.readTimeout, buf1, buf2)
		}(incomingConn)
	}
	return nil
}
