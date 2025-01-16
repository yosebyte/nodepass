package internal

import (
	"bufio"
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
}

func NewClient(parsedURL *url.URL, logger *log.Logger) *Client {
	common := &Common{
		logger:  logger,
		errChan: make(chan error, 1),
	}
	common.GetAddress(parsedURL, logger)
	return &Client{
		Common: *common,
	}
}

func (c *Client) Start() error {
	tunnelConn, err := tls.Dial("tcp", c.tunnelAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		c.logger.Error("Unable to dial server address: %v", c.tunnelAddr)
		return err
	}
	c.tunnelConn = tunnelConn
	c.logger.Debug("Tunnel connection established to: %v", c.tunnelAddr)
	go c.clientLaunch()
	return <-c.errChan
}

func (c *Client) clientLaunch() {
	reader := bufio.NewReader(c.tunnelConn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			c.logger.Error("Unable to read launch signal: %v", err)
			c.errChan <- err
			return
		}
		switch strings.TrimSpace(line) {
		case "[NODEPASS]<PING>":
			c.logger.Debug("PING signal received: %v", c.tunnelConn.RemoteAddr())
		case "[NODEPASS]<TCP>":
			go func() {
				c.logger.Debug("TCP launch signal received: %v", c.tunnelConn.RemoteAddr())
				c.errChan <- c.handleClientTCP()
			}()
		case "[NODEPASS]<UDP>":
			go func() {
				c.logger.Debug("UDP launch signal received: %v", c.tunnelConn.RemoteAddr())
				c.errChan <- c.handleClientUDP()
			}()
		}
	}
}

func (c *Client) Stop() {
	if c.tunnelConn != nil {
		c.tunnelConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelConn.RemoteAddr())
	}
	if c.remoteTCPConn != nil {
		c.remoteTCPConn.Close()
		c.logger.Debug("Remote TCP connection closed: %v", c.remoteTCPConn.RemoteAddr())
	}
	if c.remoteUDPConn != nil {
		c.remoteUDPConn.Close()
		c.logger.Debug("Remote UDP connection closed: %v", c.remoteUDPConn.RemoteAddr())
	}
	if c.targetTCPConn != nil {
		c.targetTCPConn.Close()
		c.logger.Debug("Target TCP connection closed: %v", c.targetTCPConn.RemoteAddr())
	}
	if c.targetUDPConn != nil {
		c.targetUDPConn.Close()
		c.logger.Debug("Target UDP connection closed: %v", c.targetUDPConn.RemoteAddr())
	}
}

func (c *Client) Shutdown() {
	c.Stop()
}

func (c *Client) handleClientTCP() error {
	remoteConn, err := tls.Dial("tcp", c.tunnelAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		c.logger.Error("Unable to dial server address: %v", err)
		return err
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
		c.logger.Error("Unable to dial target address: %v", err)
		return err
	}
	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()
	c.targetTCPConn = targetConn
	c.logger.Debug("Target connection established to: %v", targetConn.RemoteAddr())
	c.logger.Debug("Starting data exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
	if err := io.DataExchange(remoteConn, targetConn); err != nil {
		c.logger.Debug("Connection closed: %v", err)
	}
	c.logger.Debug("Data exchange completed")
	return nil
}

func (c *Client) handleClientUDP() error {
	remoteConn, err := tls.Dial("tcp", c.tunnelAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		c.logger.Error("Unable to dial target address: %v", err)
		return err
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
		c.logger.Error("Unable to read from remote address: %v", err)
		return err
	}
	targetConn, err := net.DialUDP("udp", nil, c.targetUDPAddr)
	if err != nil {
		c.logger.Error("Unable to dial target address: %v", err)
		return err
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
		c.logger.Error("Unable to set deadline: %v", err)
		return err
	}
	c.logger.Debug("Starting data transfer: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
	_, err = targetConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Unable to write to target address: %v", err)
		return err
	}
	n, _, err = targetConn.ReadFromUDP(buffer)
	if err != nil {
		c.logger.Error("Unable to read from target address: %v", err)
		return err
	}
	_, err = remoteConn.Write(buffer[:n])
	if err != nil {
		c.logger.Error("Unable to write to remote address: %v", err)
		return err
	}
	c.logger.Debug("Data transfer completed")
	return nil
}
