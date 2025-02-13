package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/yosebyte/x/io"
	"github.com/yosebyte/x/log"
)

type Client struct {
	Common
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
		signalChan: make(chan string, MaxSignalQueueLimit),
	}
}

func (c *Client) Start() error {
	if err := c.startTunnelConnection(); err != nil {
		c.logger.Error("Tunnel connection error: %v", err)
		return err
	}
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
		buffer := make([]byte, MaxSignalBuffer)
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
				c.logger.Debug("Signal queued: %v", strings.TrimSpace(signal))
			default:
				c.logger.Debug("Max signal queue limit reached: %v", MaxSignalQueueLimit)
			}
		}
	}()
	for signal := range c.signalChan {
		switch signal {
		case CheckSignalPING:
			c.logger.Debug("PING signal received: %v", c.tunnelConn.RemoteAddr())
			c.errChan <- c.clientPingCheck()
		case LaunchSignalTCP:
			go func() {
				c.logger.Debug("TCP launch signal received: %v", c.tunnelConn.RemoteAddr())
				c.handleClientTCP()
			}()
		case LaunchSignalUDP:
			go func() {
				c.logger.Debug("UDP launch signal received: %v", c.tunnelConn.RemoteAddr())
				c.handleClientUDP()
			}()
		}
	}
}

func (c *Client) Stop() {
	if c.remoteTCPConn != nil {
		c.remoteTCPConn.Close()
		c.logger.Debug("Remote TCP connection closed: %v", c.remoteTCPConn.LocalAddr())
	}
	if c.remoteUDPConn != nil {
		c.remoteUDPConn.Close()
		c.logger.Debug("Remote UDP connection closed: %v", c.remoteUDPConn.LocalAddr())
	}
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
	c.remoteAddr.Port = getRemotePort()
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

func (c *Client) clientPingCheck() error {
	_, err := c.tunnelConn.Write([]byte(CheckSignalPING))
	return err
}

func (c *Client) handleClientTCP() {
	remoteConn, err := tls.Dial("tcp", c.remoteAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	c.remoteTCPConn = remoteConn
	c.logger.Debug("Remote connection established to: %v", remoteConn.RemoteAddr())
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
	c.logger.Debug("Target connection established to: %v", targetConn.RemoteAddr())
	c.logger.Debug("Starting exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
	if err := io.DataExchange(remoteConn, targetConn); err != nil {
		c.logger.Debug("Exchange complete: %v", err)
	}
}

func (c *Client) handleClientUDP() {
	remoteConn, err := tls.Dial("tcp", c.remoteAddr.String(), &tls.Config{InsecureSkipVerify: true})
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
	c.logger.Debug("Remote connection established to: %v", remoteConn.RemoteAddr())
	buffer := make([]byte, MaxUDPDataBuffer)
	n, err := remoteConn.Read(buffer)
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
	c.logger.Debug("Target connection established to: %v", targetConn.RemoteAddr())
	err = targetConn.SetDeadline(time.Now().Add(MaxUDPDataTimeout))
	if err != nil {
		c.logger.Error("Set deadline failed: %v", err)
		return
	}
	c.logger.Debug("Starting data transfer: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
	_, err = targetConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Write failed: %v", err)
		return
	}
	n, _, err = targetConn.ReadFromUDP(buffer)
	if err != nil {
		c.logger.Error("Read failed: %v", err)
		return
	}
	_, err = remoteConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Write failed: %v", err)
		return
	}
	c.logger.Debug("Transfer complete: %v", remoteConn.RemoteAddr())
}
