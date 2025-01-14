package internal

import (
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/yosebyte/x/io"
	"github.com/yosebyte/x/log"
)

type Client struct {
	Common
}

func NewClient(parsedURL *url.URL, logger *log.Logger) *Client {
	common := &Common{
		logger: logger,
		done:   make(chan struct{}),
	}
	common.GetAddress(parsedURL, logger)
	return &Client{
		Common: *common,
	}
}

func (c *Client) Start() error {
	if err := c.initClient(); err != nil {
		return err
	}
	errChan := make(chan error, 1)
	go c.clientLaunch(errChan)
	return <-errChan
}

func (c *Client) initClient() error {
	tunnelConn, err := tls.Dial("tcp", c.serverAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		c.logger.Error("Unable to dial server address: %v", c.serverAddr)
		return err
	}
	c.tunnelConn = tunnelConn
	defer func() {
		if c.tunnelConn != nil {
			c.tunnelConn.Close()
		}
	}()
	c.logger.Debug("Tunnel connection established to: %v", c.serverAddr)
	return nil
}

func (c *Client) clientLaunch(errChan chan error) {
	buffer := make([]byte, MaxSignalBuffer)
	for {
		n, err := c.tunnelConn.Read(buffer)
		if err != nil {
			c.logger.Error("Unable to read launch signal: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		switch string(buffer[:n]) {
		case "[NODEPASS]<PING>\n":
			go func() {
				errChan <- c.clientPong()
			}()
		case "[NODEPASS]<TCP>\n":
			go func() {
				errChan <- c.handleClientTCP()
			}()
		case "[NODEPASS]<UDP>\n":
			go func() {
				errChan <- c.handleClientUDP()
			}()
		}
	}
}

func (c *Client) Stop() {
	if c.tunnelConn != nil {
		c.tunnelConn.Close()
	}
	if c.remoteTCPConn != nil {
		c.remoteTCPConn.Close()
	}
	if c.remoteUDPConn != nil {
		c.remoteUDPConn.Close()
	}
	if c.targetTCPConn != nil {
		c.targetTCPConn.Close()
	}
	if c.targetUDPConn != nil {
		c.targetUDPConn.Close()
	}
	c.done <- struct{}{}
}

func (c *Client) Shutdown() {
	c.Stop()
	close(c.done)
}

func (c *Client) clientPong() error {
	_, err := c.tunnelConn.Write([]byte("[NODEPASS]<PONG>\n"))
	if err != nil {
		c.logger.Error("Tunnel connection health check failed")
		c.Stop()
		return err
	}
	return nil
}

func (c *Client) handleClientTCP() error {
	go func() {
		remoteConn, err := tls.Dial("tcp", c.serverAddr.String(), &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			c.logger.Error("Unable to dial server address: %v", err)
			return
		}
		c.remoteTCPConn = remoteConn
		c.logger.Debug("Remote connection established to: %v", c.serverAddr)
		defer func() {
			if remoteConn != nil {
				remoteConn.Close()
			}
		}()
		targetConn, err := net.DialTCP("tcp", nil, c.targetTCPAddr)
		if err != nil {
			c.logger.Error("Unable to dial target address: %v", err)
			return
		}
		c.targetTCPConn = targetConn
		c.logger.Debug("Target connection established to: %v", c.targetTCPAddr)
		defer func() {
			if targetConn != nil {
				targetConn.Close()
			}
		}()
		c.logger.Debug("Starting data exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
		if err := io.DataExchange(remoteConn, targetConn); err != nil {
			c.logger.Debug("Connection closed: %v", err)
		}
	}()
	<-c.done
	return errors.New("EOF")
}

func (c *Client) handleClientUDP() error {
	go func() {
		remoteConn, err := tls.Dial("tcp", c.serverAddr.String(), &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			c.logger.Error("Unable to dial target address: %v", err)
			return
		}
		c.remoteUDPConn = remoteConn
		c.logger.Debug("Remote connection established to: %v", c.serverAddr)
		defer func() {
			if remoteConn != nil {
				remoteConn.Close()
			}
		}()
		buffer := make([]byte, MaxUDPDataBuffer)
		n, err := remoteConn.Read(buffer)
		if err != nil {
			c.logger.Error("Unable to read from remote address: %v", err)
			return
		}
		targetConn, err := net.DialUDP("udp", nil, c.targetUDPAddr)
		if err != nil {
			c.logger.Error("Unable to dial target address: %v", err)
			return
		}
		c.targetUDPConn = targetConn
		c.logger.Debug("Target connection established to: %v", c.targetUDPAddr)
		defer func() {
			if targetConn != nil {
				targetConn.Close()
			}
		}()
		err = targetConn.SetDeadline(time.Now().Add(MaxUDPDataTimeout))
		if err != nil {
			c.logger.Error("Unable to set deadline: %v", err)
			return
		}
		c.logger.Debug("Starting data transfer: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
		_, err = targetConn.Write(buffer[:n])
		if err != nil {
			c.logger.Error("Unable to write to target address: %v", err)
			return
		}
		n, _, err = targetConn.ReadFromUDP(buffer)
		if err != nil {
			c.logger.Error("Unable to read from target address: %v", err)
			return
		}
		_, err = remoteConn.Write(buffer[:n])
		if err != nil {
			c.logger.Error("Unable to write to remote address: %v", err)
			return
		}
		c.logger.Debug("Transfer completed successfully")
	}()
	<-c.done
	return errors.New("EOF")
}
