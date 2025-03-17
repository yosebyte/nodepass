package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strconv"

	"github.com/yosebyte/x/io"
	"github.com/yosebyte/x/log"
	"github.com/yosebyte/x/pool"
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
	common.getAddress(parsedURL)
	return &Client{
		Common:     *common,
		signalChan: make(chan string, SignalQueueLimit),
	}
}

func (c *Client) Start() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	if err := c.startTunnelConnection(); err != nil {
		return err
	}
	c.pool = pool.NewClientPool(MinPoolCapacity, MaxPoolCapacity, func() (net.Conn, error) {
		return net.Dial("tcp", c.remoteAddr.String())
	})
	go c.pool.ClientManager()
	go c.signalQueue()
	go c.clientLaunch()
	return <-c.errChan
}

func (c *Client) startTunnelConnection() error {
	tunnelConn, err := tls.Dial("tcp", c.tunnelAddr.String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return err
	}
	c.tunnelConn = tunnelConn
	c.logger.Debug("Tunnel connection: %v <-> %v", c.tunnelConn.LocalAddr(), c.tunnelConn.RemoteAddr())
	buf := make([]byte, 4)
	if _, err := c.tunnelConn.Read(buf); err != nil {
		return err
	}
	c.remoteAddr.Port, err = strconv.Atoi(string(buf))
	if err != nil {
		return err
	}
	c.logger.Debug("Remote signal received: %v", c.remoteAddr)
	return nil
}

func (c *Client) signalQueue() {
	buffer := make([]byte, SignalBuffer)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
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
				c.logger.Debug("Signal queue limit reached: %v", SignalQueueLimit)
			}
		}
	}
}

func (c *Client) clientLaunch() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case signal := <-c.signalChan:
			switch signal {
			case ReportSignal:
			case LaunchSignal:
				go func() {
					c.logger.Debug("Launch signal received: %v", c.tunnelConn.RemoteAddr())
					c.handleClient()
				}()
			}
		}
	}
}

func (c *Client) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.targetConn != nil {
		c.targetConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetConn.LocalAddr())
	}
	if c.tunnelConn != nil {
		c.tunnelConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelConn.LocalAddr())
	}
	if c.pool != nil {
		c.pool.Close()
		c.logger.Debug("Remote connection pool closed")
	}
	for {
		select {
		case <-c.signalChan:
		default:
			return
		}
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

func (c *Client) handleClient() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
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
			c.remoteConn = remoteConn
			c.logger.Debug("Remote connection: %v <-> %v", c.remoteConn.LocalAddr(), c.remoteConn.RemoteAddr())
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
			c.logger.Debug("Target connection: %v <-> %v", c.targetConn.LocalAddr(), c.targetConn.RemoteAddr())
			c.logger.Debug("Starting exchange: %v <-> %v", c.remoteConn.LocalAddr(), c.targetConn.LocalAddr())
			_, _, err = io.DataExchange(c.remoteConn, c.targetConn)
			c.logger.Debug("Exchange complete: %v", err)
			return
		}
	}
}
