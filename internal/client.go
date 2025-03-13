package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"time"

	"github.com/yosebyte/x/io"
	"github.com/yosebyte/x/log"
	"github.com/yosebyte/x/pool"
)

type Client struct {
	Common
	pool       *pool.Pool
	signalChan chan string
}

func NewClient(parsedURL *url.URL, logger *log.Logger) *Client {
	common := &Common{
		logger:  logger,
		errChan: make(chan error, 1),
	}
	common.GetAddress(parsedURL, logger)
	return &Client{
		Common:     *common,
		signalChan: make(chan string, SignalQueueLimit),
	}
}

func (c *Client) Start() error {
	if err := c.startTunnelConnection(); err != nil {
		c.logger.Error("Tunnel connection error: %v", err)
		return err
	}
	c.pool = pool.NewClientPool(MinPoolCapacity, MaxPoolCapacity, func() (net.Conn, error) {
		return net.Dial("tcp", c.remoteAddr.String())
	})
	go c.pool.ClientManager()
	go c.clientLaunch()
	return <-c.errChan
}

func (c *Client) startTunnelConnection() error {
	tunnelConn, err := tls.Dial("tcp", c.tunnelAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return err
	}
	c.tunnelConn = tunnelConn
	c.logger.Debug("Tunnel connection established to: %v", c.tunnelConn.RemoteAddr())
	return nil
}

func (c *Client) clientLaunch() {
	go func() {
		buffer := make([]byte, SignalBuffer)
		for {
			n, err := c.tunnelConn.Read(buffer)
			if err != nil {
				c.logger.Error("Read failed: %v", err)
				c.errChan <- err
				return
			}
			signal := string(buffer[:n])
			select {
			case c.signalChan <- signal:
			default:
				c.logger.Debug("Max signal queue limit reached: %v", SignalQueueLimit)
			}
		}
	}()
	for signal := range c.signalChan {
		switch signal {
		case CheckSignalPING:
		case LaunchSignalTCP:
			go func() {
				c.logger.Debug("TCP signal received: %v", c.tunnelConn.RemoteAddr())
				c.handleClientTCP()
			}()
		case LaunchSignalUDP:
			go func() {
				c.logger.Debug("UDP signal received: %v", c.tunnelConn.RemoteAddr())
				c.handleClientUDP()
			}()
		}
	}
}

func (c *Client) Stop() {
	if c.targetTCPConn != nil {
		c.targetTCPConn.Close()
		c.logger.Debug("Target TCP connection closed: %v", c.targetTCPConn.LocalAddr())
	}
	if c.targetUDPConn != nil {
		c.targetUDPConn.Close()
		c.logger.Debug("Target UDP connection closed: %v", c.targetUDPConn.LocalAddr())
	}
	if c.tunnelConn != nil {
		c.tunnelConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelConn.LocalAddr())
	}
	if c.pool != nil {
		c.pool.Close()
		c.logger.Debug("Remote connection pool closed")
	}
}

func (c *Client) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Stop()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (c *Client) handleClientTCP() {
	id, remoteConn := c.pool.Get()
	if id == "" {
		c.logger.Error("Get failed: %v", remoteConn)
		return
	}
	c.logger.Debug("Remote connection ID: %v <- active %v / %v", id, c.pool.Active(), c.pool.Capacity())
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	c.remoteTCPConn = remoteConn
	c.logger.Debug("Remote connection: %v --- %v", c.remoteTCPConn.LocalAddr(), c.remoteTCPConn.RemoteAddr())
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
	c.logger.Debug("Target connection: %v --- %v", c.targetTCPConn.LocalAddr(), c.targetTCPConn.RemoteAddr())
	c.logger.Debug("Starting exchange: %v <-> %v", c.remoteTCPConn.LocalAddr(), c.targetTCPConn.LocalAddr())
	_, _, err = io.DataExchange(c.remoteTCPConn, c.targetTCPConn)
	c.logger.Debug("Target connection: %v -/- %v", c.targetTCPConn.LocalAddr(), c.targetTCPConn.RemoteAddr())
	c.logger.Debug("Remote connection: %v -/- %v", c.remoteTCPConn.LocalAddr(), c.remoteTCPConn.RemoteAddr())
	c.logger.Debug("Exchange complete: %v", err)
}

func (c *Client) handleClientUDP() {
	remoteConn, err := net.Dial("tcp", c.remoteAddr.String())
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	c.remoteUDPConn = remoteConn
	c.logger.Debug("Remote connection: %v --- %v", c.remoteUDPConn.LocalAddr(), c.remoteUDPConn.RemoteAddr())
	buffer := make([]byte, UDPDataBuffer)
	n, err := c.remoteUDPConn.Read(buffer)
	if err != nil {
		c.logger.Error("Read failed: %v", err)
		return
	}
	targetConn, err := net.DialUDP("udp", nil, c.targetUDPAddr)
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}
	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()
	c.targetUDPConn = targetConn
	c.logger.Debug("Target connection: %v --- %v", c.targetUDPConn.LocalAddr(), c.targetUDPConn.RemoteAddr())
	err = c.targetUDPConn.SetDeadline(time.Now().Add(UDPDataTimeout))
	if err != nil {
		c.logger.Error("Set deadline failed: %v", err)
		return
	}
	c.logger.Debug("Starting transfer: %v <-> %v", c.remoteUDPConn.LocalAddr(), c.targetUDPConn.LocalAddr())
	_, err = c.targetUDPConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Write failed: %v", err)
		return
	}
	n, _, err = c.targetUDPConn.ReadFromUDP(buffer)
	if err != nil {
		c.logger.Error("Read failed: %v", err)
		return
	}
	_, err = c.remoteUDPConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Write failed: %v", err)
		return
	}
	c.logger.Debug("Target connection: %v -/- %v", c.targetUDPConn.LocalAddr(), c.targetUDPConn.RemoteAddr())
	c.logger.Debug("Remote connection: %v -/- %v", c.remoteUDPConn.LocalAddr(), c.remoteUDPConn.RemoteAddr())
	c.logger.Debug("Transfer complete: %v -/- %v", c.remoteUDPConn.LocalAddr(), c.targetUDPConn.LocalAddr())
}
