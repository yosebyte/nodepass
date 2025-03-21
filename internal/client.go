package internal

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/yosebyte/x/conn"
	"github.com/yosebyte/x/log"
)

type client struct {
	common
	signalChan chan string
}

func NewClient(parsedURL *url.URL, logger *log.Logger) *client {
	common := &common{
		logger: logger,
	}
	common.getAddress(parsedURL)
	return &client{
		common:     *common,
		signalChan: make(chan string, semaphoreLimit),
	}
}

func (c *client) Start() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	if err := c.startTunnelConnection(); err != nil {
		return err
	}
	c.remotePool = conn.NewClientPool(minPoolCapacity, maxPoolCapacity, func() (net.Conn, error) {
		return net.Dial("tcp", c.remoteAddr.String())
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
	if c.targetConn != nil {
		c.targetConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetConn.LocalAddr())
	}
	if c.tunnelConn != nil {
		c.tunnelConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelConn.LocalAddr())
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

func (c *client) startTunnelConnection() error {
	tunnelConn, err := tls.Dial("tcp", c.tunnelAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return err
	}
	c.tunnelConn = tunnelConn
	c.logger.Debug("Tunnel connection: %v <-> %v", c.tunnelConn.LocalAddr(), c.tunnelConn.RemoteAddr())
	return nil
}

func (c *client) clientLaunch() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case signal := <-c.signalChan:
			signalURL, err := url.Parse(signal)
			if err != nil {
				c.logger.Error("Parse failed: %v", err)
				return
			}
			switch signalURL.Scheme {
			case "remote":
				c.remoteAddr.Port, err = strconv.Atoi(signalURL.Host)
				if err != nil {
					c.logger.Error("Convert failed: %v", err)
					return
				}
			case "launch":
				go c.clientOnce(signalURL.Host)
			default:
			}
		}
	}
}

func (c *client) signalQueue() error {
	reader := bufio.NewReader(c.tunnelConn)
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			rawSignal, err := reader.ReadBytes('\n')
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

func (c *client) clientOnce(id string) {
	c.logger.Debug("Launch signal <- : %v <- %v", id, c.tunnelConn.RemoteAddr())
	remoteConn := c.remotePool.ClientGet(id)
	if remoteConn == nil {
		c.logger.Error("Get failed: %v", id)
		return
	}
	c.logger.Debug("Remote connection: %v <- active %v / %v", id, c.remotePool.Active(), c.remotePool.Capacity())
	defer func() {
		if remoteConn != nil {
			remoteConn.Close()
		}
	}()
	c.logger.Debug("Remote connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
	targetConn, err := net.Dial("tcp", c.targetAddr.String())
	if err != nil {
		c.logger.Error("Dial failed: %v", err)
		return
	}
	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()
	c.targetConn = targetConn
	c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
	c.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
	_, _, err = conn.DataExchange(remoteConn, targetConn)
	c.logger.Debug("Exchange complete: %v", err)
}
