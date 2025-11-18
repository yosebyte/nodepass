// 内部包，提供共享功能
package internal

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NodePassProject/conn"
	"github.com/NodePassProject/logs"
)

// Common 包含所有模式共享的核心功能
type Common struct {
	mu               sync.Mutex         // 互斥锁
	logger           *logs.Logger       // 日志记录器
	tlsCode          string             // TLS模式代码
	tlsConfig        *tls.Config        // TLS配置
	coreType         string             // 核心类型
	runMode          string             // 运行模式
	quicMode         string             // QUIC模式
	dataFlow         string             // 数据流向
	dialerIP         string             // 拨号本地IP
	dialerFallback   uint32             // 拨号回落标志
	tunnelKey        string             // 隧道密钥
	tunnelTCPAddr    *net.TCPAddr       // 隧道TCP地址
	tunnelUDPAddr    *net.UDPAddr       // 隧道UDP地址
	targetTCPAddrs   []*net.TCPAddr     // 目标TCP地址组
	targetUDPAddrs   []*net.UDPAddr     // 目标UDP地址组
	targetIdx        uint64             // 目标地址索引
	targetListener   *net.TCPListener   // 目标监听器
	tunnelListener   net.Listener       // 隧道监听器
	tunnelTCPConn    *net.TCPConn       // 隧道TCP连接
	tunnelUDPConn    *conn.StatConn     // 隧道UDP连接
	targetUDPConn    *conn.StatConn     // 目标UDP连接
	targetUDPSession sync.Map           // 目标UDP会话
	tunnelPool       TransportPool      // 隧道连接池
	minPoolCapacity  int                // 最小池容量
	maxPoolCapacity  int                // 最大池容量
	proxyProtocol    string             // 代理协议
	disableTCP       string             // 禁用TCP
	disableUDP       string             // 禁用UDP
	rateLimit        int                // 速率限制
	rateLimiter      *conn.RateLimiter  // 全局限速器
	readTimeout      time.Duration      // 读取超时
	bufReader        *bufio.Reader      // 缓冲读取器
	tcpBufferPool    *sync.Pool         // TCP缓冲区池
	udpBufferPool    *sync.Pool         // UDP缓冲区池
	signalChan       chan string        // 信号通道
	checkPoint       time.Time          // 检查点时间
	flushURL         *url.URL           // 重置信号
	pingURL          *url.URL           // PING信号
	pongURL          *url.URL           // PONG信号
	slotLimit        int32              // 槽位限制
	tcpSlot          int32              // TCP连接数
	udpSlot          int32              // UDP连接数
	tcpRX            uint64             // TCP接收字节数
	tcpTX            uint64             // TCP发送字节数
	udpRX            uint64             // UDP接收字节数
	udpTX            uint64             // UDP发送字节数
	ctx              context.Context    // 上下文
	cancel           context.CancelFunc // 取消函数
}

// TransportPool 统一连接池接口
type TransportPool interface {
	IncomingGet(timeout time.Duration) (string, net.Conn, error)
	OutgoingGet(id string, timeout time.Duration) (net.Conn, error)
	Flush()
	Close()
	Ready() bool
	Active() int
	Capacity() int
	Interval() time.Duration
	AddError()
	ErrorCount() int
	ResetError()
}

// 配置变量，可通过环境变量调整
var (
	semaphoreLimit   = getEnvAsInt("NP_SEMAPHORE_LIMIT", 65536)                       // 信号量限制
	tcpDataBufSize   = getEnvAsInt("NP_TCP_DATA_BUF_SIZE", 16384)                     // TCP缓冲区大小
	udpDataBufSize   = getEnvAsInt("NP_UDP_DATA_BUF_SIZE", 16384)                     // UDP缓冲区大小
	handshakeTimeout = getEnvAsDuration("NP_HANDSHAKE_TIMEOUT", 5*time.Second)        // 握手超时
	tcpDialTimeout   = getEnvAsDuration("NP_TCP_DIAL_TIMEOUT", 5*time.Second)         // TCP拨号超时
	udpDialTimeout   = getEnvAsDuration("NP_UDP_DIAL_TIMEOUT", 5*time.Second)         // UDP拨号超时
	udpReadTimeout   = getEnvAsDuration("NP_UDP_READ_TIMEOUT", 30*time.Second)        // UDP读取超时
	poolGetTimeout   = getEnvAsDuration("NP_POOL_GET_TIMEOUT", 5*time.Second)         // 池连接获取超时
	minPoolInterval  = getEnvAsDuration("NP_MIN_POOL_INTERVAL", 100*time.Millisecond) // 最小池间隔
	maxPoolInterval  = getEnvAsDuration("NP_MAX_POOL_INTERVAL", 1*time.Second)        // 最大池间隔
	reportInterval   = getEnvAsDuration("NP_REPORT_INTERVAL", 5*time.Second)          // 报告间隔
	serviceCooldown  = getEnvAsDuration("NP_SERVICE_COOLDOWN", 3*time.Second)         // 服务冷却时间
	shutdownTimeout  = getEnvAsDuration("NP_SHUTDOWN_TIMEOUT", 5*time.Second)         // 关闭超时
	ReloadInterval   = getEnvAsDuration("NP_RELOAD_INTERVAL", 1*time.Hour)            // 重载间隔
)

// 默认配置
const (
	defaultMinPool       = 64              // 默认最小池容量
	defaultMaxPool       = 1024            // 默认最大池容量
	defaultRunMode       = "0"             // 默认运行模式
	defaultQuicMode      = "0"             // 默认QUIC模式
	defaultDialerIP      = "auto"          // 默认拨号本地IP
	defaultReadTimeout   = 0 * time.Second // 默认读取超时
	defaultRateLimit     = 0               // 默认速率限制
	defaultSlotLimit     = 65536           // 默认槽位限制
	defaultProxyProtocol = "0"             // 默认代理协议
	defaultTCPStrategy   = "0"             // 默认TCP策略
	defaultUDPStrategy   = "0"             // 默认UDP策略
)

// getTCPBuffer 获取TCP缓冲区
func (c *Common) getTCPBuffer() []byte {
	buf := c.tcpBufferPool.Get().(*[]byte)
	return (*buf)[:tcpDataBufSize]
}

// putTCPBuffer 归还TCP缓冲区
func (c *Common) putTCPBuffer(buf []byte) {
	if buf != nil && cap(buf) >= tcpDataBufSize {
		c.tcpBufferPool.Put(&buf)
	}
}

// getUDPBuffer 获取UDP缓冲区
func (c *Common) getUDPBuffer() []byte {
	buf := c.udpBufferPool.Get().(*[]byte)
	return (*buf)[:udpDataBufSize]
}

// putUDPBuffer 归还UDP缓冲区
func (c *Common) putUDPBuffer(buf []byte) {
	if buf != nil && cap(buf) >= udpDataBufSize {
		c.udpBufferPool.Put(&buf)
	}
}

// tryAcquireSlot 尝试获取一个连接槽位
func (c *Common) tryAcquireSlot(isUDP bool) bool {
	if c.slotLimit == 0 {
		return true
	}

	currentTotal := atomic.LoadInt32(&c.tcpSlot) + atomic.LoadInt32(&c.udpSlot)
	if currentTotal >= c.slotLimit {
		return false
	}

	if isUDP {
		atomic.AddInt32(&c.udpSlot, 1)
	} else {
		atomic.AddInt32(&c.tcpSlot, 1)
	}
	return true
}

// releaseSlot 释放一个连接槽位
func (c *Common) releaseSlot(isUDP bool) {
	if c.slotLimit == 0 {
		return
	}

	if isUDP {
		if current := atomic.LoadInt32(&c.udpSlot); current > 0 {
			atomic.AddInt32(&c.udpSlot, -1)
		}
	} else {
		if current := atomic.LoadInt32(&c.tcpSlot); current > 0 {
			atomic.AddInt32(&c.tcpSlot, -1)
		}
	}
}

// getEnvAsInt 从环境变量获取整数值，如果不存在则使用默认值
func getEnvAsInt(name string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := strconv.Atoi(valueStr); err == nil && value >= 0 {
			return value
		}
	}
	return defaultValue
}

// getEnvAsDuration 从环境变量获取时间间隔，如果不存在则使用默认值
func getEnvAsDuration(name string, defaultValue time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(name); exists {
		if value, err := time.ParseDuration(valueStr); err == nil && value >= 0 {
			return value
		}
	}
	return defaultValue
}

// formatCertFingerprint 格式化证书指纹为标准格式
func (c *Common) formatCertFingerprint(certRaw []byte) string {
	hash := sha256.Sum256(certRaw)
	hashHex := hex.EncodeToString(hash[:])

	var formatted strings.Builder
	for i := 0; i < len(hashHex); i += 2 {
		if i > 0 {
			formatted.WriteByte(':')
		}
		formatted.WriteString(strings.ToUpper(hashHex[i : i+2]))
	}

	return "sha256:" + formatted.String()
}

// xor 对数据进行异或处理
func (c *Common) xor(data []byte) []byte {
	for i := range data {
		data[i] ^= c.tunnelKey[i%len(c.tunnelKey)]
	}
	return data
}

// encode base64编码数据
func (c *Common) encode(data []byte) []byte {
	return append([]byte(base64.StdEncoding.EncodeToString(c.xor(data))), '\n')
}

// decode base64解码数据
func (c *Common) decode(data []byte) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(bytes.TrimSuffix(data, []byte{'\n'})))
	if err != nil {
		return nil, fmt.Errorf("decode: base64 decode failed: %w", err)
	}
	return c.xor(decoded), nil
}

// getAddress 解析和设置地址信息
func (c *Common) getAddress(parsedURL *url.URL) error {
	// 解析隧道地址
	tunnelAddr := parsedURL.Host
	if tunnelAddr == "" {
		return fmt.Errorf("getAddress: no valid tunnel address found")
	}

	// 解析隧道TCP地址
	if tunnelTCPAddr, err := net.ResolveTCPAddr("tcp", tunnelAddr); err == nil {
		c.tunnelTCPAddr = tunnelTCPAddr
	} else {
		return fmt.Errorf("getAddress: resolveTCPAddr failed: %w", err)
	}

	// 解析隧道UDP地址
	if tunnelUDPAddr, err := net.ResolveUDPAddr("udp", tunnelAddr); err == nil {
		c.tunnelUDPAddr = tunnelUDPAddr
	} else {
		return fmt.Errorf("getAddress: resolveUDPAddr failed: %w", err)
	}

	// 处理目标地址组
	targetAddr := strings.TrimPrefix(parsedURL.Path, "/")
	if targetAddr == "" {
		return fmt.Errorf("getAddress: no valid target address found")
	}

	addrList := strings.Split(targetAddr, ",")
	tempTCPAddrs := make([]*net.TCPAddr, 0, len(addrList))
	tempUDPAddrs := make([]*net.UDPAddr, 0, len(addrList))

	for _, addr := range addrList {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}

		// 解析目标TCP地址
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return fmt.Errorf("getAddress: resolveTCPAddr failed for %s: %w", addr, err)
		}

		// 解析目标UDP地址
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return fmt.Errorf("getAddress: resolveUDPAddr failed for %s: %w", addr, err)
		}

		tempTCPAddrs = append(tempTCPAddrs, tcpAddr)
		tempUDPAddrs = append(tempUDPAddrs, udpAddr)
	}

	if len(tempTCPAddrs) == 0 || len(tempUDPAddrs) == 0 || len(tempTCPAddrs) != len(tempUDPAddrs) {
		return fmt.Errorf("getAddress: no valid target address found")
	}

	// 设置目标地址组
	c.targetTCPAddrs = tempTCPAddrs
	c.targetUDPAddrs = tempUDPAddrs
	c.targetIdx = 0

	// 无限循环检查
	tunnelPort := c.tunnelTCPAddr.Port
	for _, targetAddr := range c.targetTCPAddrs {
		if targetAddr.Port == tunnelPort && (targetAddr.IP.IsLoopback() || c.tunnelTCPAddr.IP.IsUnspecified()) {
			return fmt.Errorf("getAddress: tunnel port %d conflicts with target address %s", tunnelPort, targetAddr.String())
		}
	}

	return nil
}

// getCoreType 获取核心类型
func (c *Common) getCoreType(parsedURL *url.URL) {
	c.coreType = parsedURL.Scheme
}

// getTargetAddrsString 获取目标地址组的字符串表示
func (c *Common) getTargetAddrsString() string {
	addrs := make([]string, len(c.targetTCPAddrs))
	for i, addr := range c.targetTCPAddrs {
		addrs[i] = addr.String()
	}
	return strings.Join(addrs, ",")
}

// nextTargetIdx 获取下一个目标地址索引
func (c *Common) nextTargetIdx() int {
	if len(c.targetTCPAddrs) <= 1 {
		return 0
	}
	return int((atomic.AddUint64(&c.targetIdx, 1) - 1) % uint64(len(c.targetTCPAddrs)))
}

// dialWithRotation 轮询拨号到目标地址组
func (c *Common) dialWithRotation(network string, timeout time.Duration) (net.Conn, error) {
	var addrCount int
	var getAddr func(int) string

	if network == "tcp" {
		addrCount = len(c.targetTCPAddrs)
		getAddr = func(i int) string { return c.targetTCPAddrs[i].String() }
	} else {
		addrCount = len(c.targetUDPAddrs)
		getAddr = func(i int) string { return c.targetUDPAddrs[i].String() }
	}

	// 配置拨号器
	dialer := &net.Dialer{Timeout: timeout}
	if c.dialerIP != defaultDialerIP && atomic.LoadUint32(&c.dialerFallback) == 0 {
		if network == "tcp" {
			dialer.LocalAddr = &net.TCPAddr{IP: net.ParseIP(c.dialerIP)}
		} else {
			dialer.LocalAddr = &net.UDPAddr{IP: net.ParseIP(c.dialerIP)}
		}
	}

	// 尝试拨号并自动回落
	tryDial := func(addr string) (net.Conn, error) {
		conn, err := dialer.Dial(network, addr)
		if err != nil && dialer.LocalAddr != nil && atomic.CompareAndSwapUint32(&c.dialerFallback, 0, 1) {
			c.logger.Error("dialWithRotation: fallback to system auto due to dialer failure: %v", err)
			dialer.LocalAddr = nil
			return dialer.Dial(network, addr)
		}
		return conn, err
	}

	// 单目标地址：快速路径
	if addrCount == 1 {
		return tryDial(getAddr(0))
	}

	// 多目标地址：负载均衡 + 故障转移
	startIdx := c.nextTargetIdx()
	var lastErr error
	for i := range addrCount {
		conn, err := tryDial(getAddr((startIdx + i) % addrCount))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("dialWithRotation: all %d targets failed: %w", addrCount, lastErr)
}

// getTunnelKey 从URL中获取隧道密钥
func (c *Common) getTunnelKey(parsedURL *url.URL) {
	if key := parsedURL.User.Username(); key != "" {
		c.tunnelKey = key
	} else {
		hash := fnv.New32a()
		hash.Write([]byte(parsedURL.Port()))
		c.tunnelKey = hex.EncodeToString(hash.Sum(nil))
	}
}

// getPoolCapacity 获取连接池容量设置
func (c *Common) getPoolCapacity(parsedURL *url.URL) {
	if min := parsedURL.Query().Get("min"); min != "" {
		if value, err := strconv.Atoi(min); err == nil && value > 0 {
			c.minPoolCapacity = value
		}
	} else {
		c.minPoolCapacity = defaultMinPool
	}

	if max := parsedURL.Query().Get("max"); max != "" {
		if value, err := strconv.Atoi(max); err == nil && value > 0 {
			c.maxPoolCapacity = value
		}
	} else {
		c.maxPoolCapacity = defaultMaxPool
	}
}

// getRunMode 获取运行模式
func (c *Common) getRunMode(parsedURL *url.URL) {
	if mode := parsedURL.Query().Get("mode"); mode != "" {
		c.runMode = mode
	} else {
		c.runMode = defaultRunMode
	}
}

// getQuicMode 获取QUIC模式
func (c *Common) getQuicMode(parsedURL *url.URL) {
	if quicMode := parsedURL.Query().Get("quic"); quicMode != "" {
		c.quicMode = quicMode
	} else {
		c.quicMode = defaultQuicMode
	}
	if c.quicMode != "0" && c.tlsCode == "0" {
		c.tlsCode = "1"
	}
}

// getDialerIP 获取拨号本地IP设置
func (c *Common) getDialerIP(parsedURL *url.URL) {
	if dialerIP := parsedURL.Query().Get("dial"); dialerIP != "" && dialerIP != "auto" {
		if ip := net.ParseIP(dialerIP); ip != nil {
			c.dialerIP = dialerIP
			return
		} else {
			c.logger.Error("getDialerIP: fallback to system auto due to invalid IP address: %v", dialerIP)
		}
	}
	c.dialerIP = defaultDialerIP
}

// getReadTimeout 获取读取超时设置
func (c *Common) getReadTimeout(parsedURL *url.URL) {
	if timeout := parsedURL.Query().Get("read"); timeout != "" {
		if value, err := time.ParseDuration(timeout); err == nil && value > 0 {
			c.readTimeout = value
		}
	} else {
		c.readTimeout = defaultReadTimeout
	}
}

// getRateLimit 获取速率限制
func (c *Common) getRateLimit(parsedURL *url.URL) {
	if limit := parsedURL.Query().Get("rate"); limit != "" {
		if value, err := strconv.Atoi(limit); err == nil && value > 0 {
			c.rateLimit = value * 125000
		}
	} else {
		c.rateLimit = defaultRateLimit
	}
}

// getSlotLimit 获取连接槽位限制
func (c *Common) getSlotLimit(parsedURL *url.URL) {
	if slot := parsedURL.Query().Get("slot"); slot != "" {
		if value, err := strconv.Atoi(slot); err == nil && value > 0 {
			c.slotLimit = int32(value)
		}
	} else {
		c.slotLimit = defaultSlotLimit
	}
}

// getProxyProtocol 获取代理协议设置
func (c *Common) getProxyProtocol(parsedURL *url.URL) {
	if protocol := parsedURL.Query().Get("proxy"); protocol != "" {
		c.proxyProtocol = protocol
	} else {
		c.proxyProtocol = defaultProxyProtocol
	}
}

// getTCPStrategy 获取TCP策略
func (c *Common) getTCPStrategy(parsedURL *url.URL) {
	if tcpStrategy := parsedURL.Query().Get("notcp"); tcpStrategy != "" {
		c.disableTCP = tcpStrategy
	} else {
		c.disableTCP = defaultTCPStrategy
	}
}

// getUDPStrategy 获取UDP策略
func (c *Common) getUDPStrategy(parsedURL *url.URL) {
	if udpStrategy := parsedURL.Query().Get("noudp"); udpStrategy != "" {
		c.disableUDP = udpStrategy
	} else {
		c.disableUDP = defaultUDPStrategy
	}
}

// initConfig 初始化配置
func (c *Common) initConfig(parsedURL *url.URL) error {
	if err := c.getAddress(parsedURL); err != nil {
		return err
	}

	c.getCoreType(parsedURL)
	c.getTunnelKey(parsedURL)
	c.getPoolCapacity(parsedURL)
	c.getRunMode(parsedURL)
	c.getQuicMode(parsedURL)
	c.getDialerIP(parsedURL)
	c.getReadTimeout(parsedURL)
	c.getRateLimit(parsedURL)
	c.getSlotLimit(parsedURL)
	c.getProxyProtocol(parsedURL)
	c.getTCPStrategy(parsedURL)
	c.getUDPStrategy(parsedURL)

	return nil
}

// sendProxyV1Header 发送PROXY v1
func (c *Common) sendProxyV1Header(ip string, conn net.Conn) error {
	if c.proxyProtocol != "1" {
		return nil
	}

	clientAddr, err := net.ResolveTCPAddr("tcp", ip)
	if err != nil {
		return fmt.Errorf("sendProxyV1Header: resolveTCPAddr failed: %w", err)
	}
	remoteAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("sendProxyV1Header: remote address is not TCPAddr")
	}

	var protocol string
	switch {
	case clientAddr.IP.To4() != nil && remoteAddr.IP.To4() != nil:
		protocol = "TCP4"
	case clientAddr.IP.To16() != nil && remoteAddr.IP.To16() != nil:
		protocol = "TCP6"
	default:
		return fmt.Errorf("sendProxyV1Header: unsupported IP protocol for PROXY v1")
	}

	if _, err = fmt.Fprintf(conn, "PROXY %s %s %s %d %d\r\n",
		protocol,
		clientAddr.IP.String(),
		remoteAddr.IP.String(),
		clientAddr.Port,
		remoteAddr.Port); err != nil {
		return fmt.Errorf("sendProxyV1Header: fprintf failed: %w", err)
	}

	return nil
}

// initRateLimiter 初始化全局限速器
func (c *Common) initRateLimiter() {
	if c.rateLimit > 0 {
		c.rateLimiter = conn.NewRateLimiter(int64(c.rateLimit), int64(c.rateLimit))
	}
}

// initContext 初始化上下文
func (c *Common) initContext() {
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
}

// initTunnelListener 初始化隧道监听器
func (c *Common) initTunnelListener() error {
	if c.tunnelTCPAddr == nil && c.tunnelUDPAddr == nil {
		return fmt.Errorf("initTunnelListener: nil tunnel address")
	}

	// 初始化隧道TCP监听器
	if c.tunnelTCPAddr != nil && (c.disableTCP != "1" || c.coreType != "client") {
		tunnelListener, err := net.ListenTCP("tcp", c.tunnelTCPAddr)
		if err != nil {
			return fmt.Errorf("initTunnelListener: listenTCP failed: %w", err)
		}
		c.tunnelListener = tunnelListener
	}

	// 初始化隧道UDP监听器
	if c.tunnelUDPAddr != nil && (c.disableUDP != "1" || c.coreType != "client") {
		tunnelUDPConn, err := net.ListenUDP("udp", c.tunnelUDPAddr)
		if err != nil {
			return fmt.Errorf("initTunnelListener: listenUDP failed: %w", err)
		}
		c.tunnelUDPConn = &conn.StatConn{Conn: tunnelUDPConn, RX: &c.udpRX, TX: &c.udpTX, Rate: c.rateLimiter}
	}

	return nil
}

// initTargetListener 初始化目标监听器
func (c *Common) initTargetListener() error {
	if len(c.targetTCPAddrs) == 0 && len(c.targetUDPAddrs) == 0 {
		return fmt.Errorf("initTargetListener: no target address")
	}

	// 初始化目标TCP监听器
	if len(c.targetTCPAddrs) > 0 && c.disableTCP != "1" {
		targetListener, err := net.ListenTCP("tcp", c.targetTCPAddrs[0])
		if err != nil {
			return fmt.Errorf("initTargetListener: listenTCP failed: %w", err)
		}
		c.targetListener = targetListener
	}

	// 初始化目标UDP监听器
	if len(c.targetUDPAddrs) > 0 && c.disableUDP != "1" {
		targetUDPConn, err := net.ListenUDP("udp", c.targetUDPAddrs[0])
		if err != nil {
			return fmt.Errorf("initTargetListener: listenUDP failed: %w", err)
		}
		c.targetUDPConn = &conn.StatConn{Conn: targetUDPConn, RX: &c.udpRX, TX: &c.udpTX, Rate: c.rateLimiter}
	}

	return nil
}

// drain 清空通道中的所有元素
func drain[T any](ch <-chan T) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// stop 共用停止服务
func (c *Common) stop() {
	// 取消上下文
	if c.cancel != nil {
		c.cancel()
	}

	// 关闭隧道连接池
	if c.tunnelPool != nil {
		active := c.tunnelPool.Active()
		c.tunnelPool.Close()
		c.logger.Debug("Tunnel connection closed: pool active %v", active)
	}

	// 清理目标UDP会话
	c.targetUDPSession.Range(func(key, value any) bool {
		if conn, ok := value.(*net.UDPConn); ok {
			conn.Close()
		}
		c.targetUDPSession.Delete(key)
		return true
	})

	// 关闭目标UDP连接
	if c.targetUDPConn != nil {
		c.targetUDPConn.Close()
		c.logger.Debug("Target connection closed: %v", c.targetUDPConn.LocalAddr())
	}

	// 关闭隧道UDP连接
	if c.tunnelUDPConn != nil {
		c.tunnelUDPConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelUDPConn.LocalAddr())
	}

	// 关闭隧道TCP连接
	if c.tunnelTCPConn != nil {
		c.tunnelTCPConn.Close()
		c.logger.Debug("Tunnel connection closed: %v", c.tunnelTCPConn.LocalAddr())
	}

	// 关闭目标监听器
	if c.targetListener != nil {
		c.targetListener.Close()
		c.logger.Debug("Target listener closed: %v", c.targetListener.Addr())
	}

	// 关闭隧道监听器
	if c.tunnelListener != nil {
		c.tunnelListener.Close()
		c.logger.Debug("Tunnel listener closed: %v", c.tunnelListener.Addr())
	}

	// 清空通道
	drain(c.signalChan)

	// 重置全局限速器
	if c.rateLimiter != nil {
		c.rateLimiter.Reset()
	}
}

// shutdown 共用优雅关闭
func (c *Common) shutdown(ctx context.Context, stopFunc func()) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		stopFunc()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown: context error: %w", ctx.Err())
	case <-done:
		return nil
	}
}

// commonControl 共用控制逻辑
func (c *Common) commonControl() error {
	errChan := make(chan error, 3)

	// 信号消纳、信号队列和健康检查
	go func() { errChan <- c.commonOnce() }()
	go func() { errChan <- c.commonQueue() }()
	go func() { errChan <- c.healthCheck() }()

	select {
	case <-c.ctx.Done():
		return fmt.Errorf("commonControl: context error: %w", c.ctx.Err())
	case err := <-errChan:
		return fmt.Errorf("commonControl: %w", err)
	}
}

// commonQueue 共用信号队列
func (c *Common) commonQueue() error {
	for c.ctx.Err() == nil {
		// 读取原始信号
		rawSignal, err := c.bufReader.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("commonQueue: readBytes failed: %w", err)
		}

		// 解码信号
		signalData, err := c.decode(rawSignal)
		if err != nil {
			c.logger.Error("commonQueue: decode signal failed: %v", err)
			select {
			case <-c.ctx.Done():
				return fmt.Errorf("commonQueue: context error: %w", c.ctx.Err())
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}
		signal := string(signalData)

		// 将信号发送到通道
		select {
		case c.signalChan <- signal:
		default:
			c.logger.Error("commonQueue: queue limit reached: %v", semaphoreLimit)
			select {
			case <-c.ctx.Done():
				return fmt.Errorf("commonQueue: context error: %w", c.ctx.Err())
			case <-time.After(50 * time.Millisecond):
			}
		}
	}

	return fmt.Errorf("commonQueue: context error: %w", c.ctx.Err())
}

// healthCheck 共用健康度检查
func (c *Common) healthCheck() error {
	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()

	if c.tlsCode == "1" || c.tlsCode == "2" {
		go func() {
			select {
			case <-c.ctx.Done():
			case <-ticker.C:
				c.incomingVerify()
			}
		}()
	}

	for c.ctx.Err() == nil {
		// 尝试获取锁
		if !c.mu.TryLock() {
			continue
		}

		// 连接池健康度检查
		if c.tunnelPool.ErrorCount() > c.tunnelPool.Active()/2 {
			// 发送刷新信号到对端
			if c.ctx.Err() == nil && c.tunnelTCPConn != nil {
				_, err := c.tunnelTCPConn.Write(c.encode([]byte(c.flushURL.String())))
				if err != nil {
					c.mu.Unlock()
					return fmt.Errorf("healthCheck: write flush signal failed: %w", err)
				}
			}
			c.tunnelPool.Flush()
			c.tunnelPool.ResetError()

			select {
			case <-c.ctx.Done():
				return fmt.Errorf("healthCheck: context error: %w", c.ctx.Err())
			case <-ticker.C:
			}

			c.logger.Debug("Tunnel pool flushed: %v active connections", c.tunnelPool.Active())
		}

		// 发送PING信号
		c.checkPoint = time.Now()
		if c.ctx.Err() == nil && c.tunnelTCPConn != nil {
			_, err := c.tunnelTCPConn.Write(c.encode([]byte(c.pingURL.String())))
			if err != nil {
				c.mu.Unlock()
				return fmt.Errorf("healthCheck: write ping signal failed: %w", err)
			}
		}

		c.mu.Unlock()
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("healthCheck: context error: %w", c.ctx.Err())
		case <-ticker.C:
		}
	}

	return fmt.Errorf("healthCheck: context error: %w", c.ctx.Err())
}

// incomingVerify 入口连接验证
func (c *Common) incomingVerify() {
	for c.ctx.Err() == nil {
		if c.tunnelPool.Ready() {
			break
		}
		select {
		case <-c.ctx.Done():
			continue
		case <-time.After(50 * time.Millisecond):
		}
	}

	if c.tlsConfig == nil || len(c.tlsConfig.Certificates) == 0 {
		return
	}

	cert := c.tlsConfig.Certificates[0]
	if len(cert.Certificate) == 0 {
		return
	}

	// 打印证书指纹
	c.logger.Info("TLS cert verified: %v", c.formatCertFingerprint(cert.Certificate[0]))

	id, testConn, err := c.tunnelPool.IncomingGet(poolGetTimeout)
	if err != nil {
		return
	}
	defer testConn.Close()

	// 构建并发送验证信号
	verifyURL := &url.URL{
		Scheme:   "np",
		Host:     c.tunnelTCPConn.RemoteAddr().String(),
		Path:     url.PathEscape(id),
		Fragment: "v", // TLS验证
	}

	if c.ctx.Err() == nil && c.tunnelTCPConn != nil {
		c.mu.Lock()
		_, err = c.tunnelTCPConn.Write(c.encode([]byte(verifyURL.String())))
		c.mu.Unlock()
		if err != nil {
			return
		}
	}

	c.logger.Debug("TLS verify signal: cid %v -> %v", id, c.tunnelTCPConn.RemoteAddr())
}

// commonLoop 共用处理循环
func (c *Common) commonLoop() {
	for c.ctx.Err() == nil {
		// 等待连接池准备就绪
		if c.tunnelPool.Ready() {
			if c.targetListener != nil || c.disableTCP != "1" {
				go c.commonTCPLoop()
			}
			if c.targetUDPConn != nil || c.disableUDP != "1" {
				go c.commonUDPLoop()
			}
			return
		}

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// commonTCPLoop 共用TCP请求处理循环
func (c *Common) commonTCPLoop() {
	for c.ctx.Err() == nil {
		// 接受来自目标的TCP连接
		targetConn, err := c.targetListener.Accept()
		if err != nil {
			if c.ctx.Err() != nil || err == net.ErrClosed {
				return
			}
			c.logger.Error("commonTCPLoop: accept failed: %v", err)

			select {
			case <-c.ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		targetConn = &conn.StatConn{Conn: targetConn, RX: &c.tcpRX, TX: &c.tcpTX, Rate: c.rateLimiter}
		c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())

		go func(targetConn net.Conn) {
			defer func() {
				if targetConn != nil {
					targetConn.Close()
				}
			}()

			// 尝试获取TCP连接槽位
			if !c.tryAcquireSlot(false) {
				c.logger.Error("commonTCPLoop: TCP slot limit reached: %v/%v", c.tcpSlot, c.slotLimit)
				return
			}

			defer c.releaseSlot(false)

			// 从连接池获取连接
			id, remoteConn, err := c.tunnelPool.IncomingGet(poolGetTimeout)
			if err != nil {
				c.logger.Warn("commonTCPLoop: request timeout: %v", err)
				return
			}

			c.logger.Debug("Tunnel connection: get %v <- pool active %v", id, c.tunnelPool.Active())

			defer func() {
				// 池连接关闭
				if remoteConn != nil {
					remoteConn.Close()
					c.logger.Debug("Tunnel connection: closed %v", id)
				}
			}()

			c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

			// 构建并发送启动信号
			launchURL := &url.URL{
				Scheme:   "np",
				Host:     targetConn.RemoteAddr().String(),
				Path:     url.PathEscape(id),
				Fragment: "1", // TCP模式
			}

			if c.ctx.Err() == nil && c.tunnelTCPConn != nil {
				c.mu.Lock()
				_, err = c.tunnelTCPConn.Write(c.encode([]byte(launchURL.String())))
				c.mu.Unlock()

				if err != nil {
					c.logger.Error("commonTCPLoop: write launch signal failed: %v", err)
					return
				}
			}

			c.logger.Debug("TCP launch signal: cid %v -> %v", id, c.tunnelTCPConn.RemoteAddr())

			buffer1 := c.getTCPBuffer()
			buffer2 := c.getTCPBuffer()
			defer func() {
				c.putTCPBuffer(buffer1)
				c.putTCPBuffer(buffer2)
			}()

			// 交换数据
			c.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
			c.logger.Debug("Exchange complete: %v", conn.DataExchange(remoteConn, targetConn, c.readTimeout, buffer1, buffer2))
		}(targetConn)
	}
}

// commonUDPLoop 共用UDP请求处理循环
func (c *Common) commonUDPLoop() {
	for c.ctx.Err() == nil {
		buffer := c.getUDPBuffer()

		// 读取来自目标的UDP数据
		x, clientAddr, err := c.targetUDPConn.ReadFromUDP(buffer)
		if err != nil {
			if c.ctx.Err() != nil || err == net.ErrClosed {
				c.putUDPBuffer(buffer)
				return
			}
			c.logger.Error("commonUDPLoop: readFromUDP failed: %v", err)
			c.putUDPBuffer(buffer)

			select {
			case <-c.ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		c.logger.Debug("Target connection: %v <-> %v", c.targetUDPConn.LocalAddr(), clientAddr)

		var id string
		var remoteConn net.Conn
		sessionKey := clientAddr.String()

		// 获取或创建UDP会话
		if session, ok := c.targetUDPSession.Load(sessionKey); ok {
			// 复用现有会话
			remoteConn = session.(net.Conn)
			c.logger.Debug("Using UDP session: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())
		} else {
			// 尝试获取UDP连接槽位
			if !c.tryAcquireSlot(true) {
				c.logger.Error("commonUDPLoop: UDP slot limit reached: %v/%v", c.udpSlot, c.slotLimit)
				c.putUDPBuffer(buffer)
				continue
			}

			// 获取池连接
			id, remoteConn, err = c.tunnelPool.IncomingGet(poolGetTimeout)
			if err != nil {
				c.logger.Warn("commonUDPLoop: request timeout: %v", err)
				c.releaseSlot(true)
				c.putUDPBuffer(buffer)
				continue
			}
			c.targetUDPSession.Store(sessionKey, remoteConn)
			c.logger.Debug("Tunnel connection: get %v <- pool active %v", id, c.tunnelPool.Active())
			c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

			go func(remoteConn net.Conn, clientAddr *net.UDPAddr, sessionKey, id string) {
				defer func() {
					// 清理UDP会话和释放槽位
					c.targetUDPSession.Delete(sessionKey)
					c.releaseSlot(true)

					// 池连接关闭
					if remoteConn != nil {
						remoteConn.Close()
						c.logger.Debug("Tunnel connection: closed %v", id)
					}
				}()

				buffer := c.getUDPBuffer()
				defer c.putUDPBuffer(buffer)
				reader := &conn.TimeoutReader{Conn: remoteConn, Timeout: udpReadTimeout}

				for c.ctx.Err() == nil {
					// 从池连接读取数据
					x, err := reader.Read(buffer)
					if err != nil {
						if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
							c.logger.Debug("UDP session abort: %v", err)
						} else if err != io.EOF {
							c.logger.Error("commonUDPLoop: read from tunnel failed: %v", err)
						}
						return
					}

					// 将数据写入目标UDP连接
					_, err = c.targetUDPConn.WriteToUDP(buffer[:x], clientAddr)
					if err != nil {
						if err != io.EOF {
							c.logger.Error("commonUDPLoop: writeToUDP failed: %v", err)
						}
						return
					}
					// 传输完成
					c.logger.Debug("Transfer complete: %v <-> %v", remoteConn.LocalAddr(), c.targetUDPConn.LocalAddr())
				}
			}(remoteConn, clientAddr, sessionKey, id)

			// 构建并发送启动信号
			launchURL := &url.URL{
				Scheme:   "np",
				Host:     clientAddr.String(),
				Path:     url.PathEscape(id),
				Fragment: "2", // UDP模式
			}

			if c.ctx.Err() == nil && c.tunnelTCPConn != nil {
				c.mu.Lock()
				_, err = c.tunnelTCPConn.Write(c.encode([]byte(launchURL.String())))
				c.mu.Unlock()
				if err != nil {
					c.logger.Error("commonUDPLoop: write launch signal failed: %v", err)
					continue
				}
			}

			c.logger.Debug("UDP launch signal: cid %v -> %v", id, c.tunnelTCPConn.RemoteAddr())
			c.logger.Debug("Starting transfer: %v <-> %v", remoteConn.LocalAddr(), c.targetUDPConn.LocalAddr())
		}

		// 将原始数据写入池连接
		_, err = remoteConn.Write(buffer[:x])
		if err != nil {
			if err != io.EOF {
				c.logger.Error("commonUDPLoop: write to tunnel failed: %v", err)
			}
			c.targetUDPSession.Delete(sessionKey)
			remoteConn.Close()
			c.putUDPBuffer(buffer)
			continue
		}

		// 传输完成
		c.logger.Debug("Transfer complete: %v <-> %v", remoteConn.LocalAddr(), c.targetUDPConn.LocalAddr())
		c.putUDPBuffer(buffer)
	}
}

// commonOnce 共用处理单个请求
func (c *Common) commonOnce() error {
	for c.ctx.Err() == nil {
		// 等待连接池准备就绪
		if !c.tunnelPool.Ready() {
			select {
			case <-c.ctx.Done():
				return fmt.Errorf("commonOnce: context error: %w", c.ctx.Err())
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		select {
		case <-c.ctx.Done():
			return fmt.Errorf("commonOnce: context error: %w", c.ctx.Err())
		case signal := <-c.signalChan:
			// 解析信号URL
			signalURL, err := url.Parse(signal)
			if err != nil {
				c.logger.Error("commonOnce: parse signal failed: %v", err)
				select {
				case <-c.ctx.Done():
					return fmt.Errorf("commonOnce: context error: %w", c.ctx.Err())
				case <-time.After(50 * time.Millisecond):
				}
				continue
			}

			// 处理信号
			switch signalURL.Fragment {
			case "v": // 验证
				if c.tlsCode == "1" || c.tlsCode == "2" {
					go c.outgoingVerify(signalURL)
				}
			case "1": // TCP
				if c.disableTCP != "1" {
					go c.commonTCPOnce(signalURL)
				}
			case "2": // UDP
				if c.disableUDP != "1" {
					go c.commonUDPOnce(signalURL)
				}
			case "f": // 连接池刷新
				go func() {
					c.tunnelPool.Flush()
					c.tunnelPool.ResetError()

					select {
					case <-c.ctx.Done():
						return
					case <-time.After(reportInterval):
					}

					c.logger.Debug("Tunnel pool flushed: %v active connections", c.tunnelPool.Active())
				}()
			case "i": // PING
				if c.ctx.Err() == nil && c.tunnelTCPConn != nil {
					c.mu.Lock()
					_, err := c.tunnelTCPConn.Write(c.encode([]byte(c.pongURL.String())))
					c.mu.Unlock()
					if err != nil {
						return fmt.Errorf("commonOnce: write pong signal failed: %w", err)
					}
				}
			case "o": // PONG
				// 发送检查点事件
				c.logger.Event("CHECK_POINT|MODE=%v|PING=%vms|POOL=%v|TCPS=%v|UDPS=%v|TCPRX=%v|TCPTX=%v|UDPRX=%v|UDPTX=%v",
					c.runMode, time.Since(c.checkPoint).Milliseconds(), c.tunnelPool.Active(),
					atomic.LoadInt32(&c.tcpSlot), atomic.LoadInt32(&c.udpSlot),
					atomic.LoadUint64(&c.tcpRX), atomic.LoadUint64(&c.tcpTX),
					atomic.LoadUint64(&c.udpRX), atomic.LoadUint64(&c.udpTX))
			default:
				// 无效信号
			}
		}
	}

	return fmt.Errorf("commonOnce: context error: %w", c.ctx.Err())
}

// outgoingVerify 出口连接验证
func (c *Common) outgoingVerify(signalURL *url.URL) {
	for c.ctx.Err() == nil {
		if c.tunnelPool.Ready() {
			break
		}
		select {
		case <-c.ctx.Done():
			continue
		case <-time.After(50 * time.Millisecond):
		}
	}

	id := strings.TrimPrefix(signalURL.Path, "/")
	if unescapedID, err := url.PathUnescape(id); err != nil {
		c.logger.Error("outgoingVerify: unescape id failed: %v", err)
		return
	} else {
		id = unescapedID
	}
	c.logger.Debug("TLS verify signal: cid %v <- %v", id, c.tunnelTCPConn.RemoteAddr())

	testConn, err := c.tunnelPool.OutgoingGet(id, poolGetTimeout)
	if err != nil {
		c.logger.Error("outgoingVerify: request timeout: %v", err)
		c.tunnelPool.AddError()
		return
	}
	defer testConn.Close()

	if testConn != nil {
		conn, ok := testConn.(interface{ ConnectionState() tls.ConnectionState })
		if !ok {
			return
		}
		state := conn.ConnectionState()

		if len(state.PeerCertificates) == 0 {
			c.logger.Error("outgoingVerify: no peer certificates found")
			return
		}

		// 打印证书指纹
		c.logger.Info("TLS cert verified: %v", c.formatCertFingerprint(state.PeerCertificates[0].Raw))
	}
}

// commonTCPOnce 共用处理单个TCP请求
func (c *Common) commonTCPOnce(signalURL *url.URL) {
	id := strings.TrimPrefix(signalURL.Path, "/")
	if unescapedID, err := url.PathUnescape(id); err != nil {
		c.logger.Error("commonTCPOnce: unescape id failed: %v", err)
		return
	} else {
		id = unescapedID
	}
	c.logger.Debug("TCP launch signal: cid %v <- %v", id, c.tunnelTCPConn.RemoteAddr())

	// 从连接池获取连接
	remoteConn, err := c.tunnelPool.OutgoingGet(id, poolGetTimeout)
	if err != nil {
		c.logger.Error("commonTCPOnce: request timeout: %v", err)
		c.tunnelPool.AddError()
		return
	}

	c.logger.Debug("Tunnel connection: get %v <- pool active %v", id, c.tunnelPool.Active())

	defer func() {
		// 池连接关闭
		if remoteConn != nil {
			remoteConn.Close()
			c.logger.Debug("Tunnel connection: closed %v", id)
		}
	}()

	c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

	// 尝试获取TCP连接槽位
	if !c.tryAcquireSlot(false) {
		c.logger.Error("commonTCPOnce: TCP slot limit reached: %v/%v", c.tcpSlot, c.slotLimit)
		return
	}

	defer c.releaseSlot(false)

	// 连接到目标TCP地址
	targetConn, err := c.dialWithRotation("tcp", tcpDialTimeout)
	if err != nil {
		c.logger.Error("commonTCPOnce: dialWithRotation failed: %v", err)
		return
	}

	defer func() {
		if targetConn != nil {
			targetConn.Close()
		}
	}()

	targetConn = &conn.StatConn{Conn: targetConn, RX: &c.tcpRX, TX: &c.tcpTX, Rate: c.rateLimiter}
	c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())

	// 发送PROXY v1
	if err := c.sendProxyV1Header(signalURL.Host, targetConn); err != nil {
		c.logger.Error("commonTCPOnce: sendProxyV1Header failed: %v", err)
		return
	}

	buffer1 := c.getTCPBuffer()
	buffer2 := c.getTCPBuffer()
	defer func() {
		c.putTCPBuffer(buffer1)
		c.putTCPBuffer(buffer2)
	}()

	// 交换数据
	c.logger.Debug("Starting exchange: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
	c.logger.Debug("Exchange complete: %v", conn.DataExchange(remoteConn, targetConn, c.readTimeout, buffer1, buffer2))
}

// commonUDPOnce 共用处理单个UDP请求
func (c *Common) commonUDPOnce(signalURL *url.URL) {
	id := strings.TrimPrefix(signalURL.Path, "/")
	if unescapedID, err := url.PathUnescape(id); err != nil {
		c.logger.Error("commonUDPOnce: unescape id failed: %v", err)
		return
	} else {
		id = unescapedID
	}
	c.logger.Debug("UDP launch signal: cid %v <- %v", id, c.tunnelTCPConn.RemoteAddr())

	// 获取池连接
	remoteConn, err := c.tunnelPool.OutgoingGet(id, poolGetTimeout)
	if err != nil {
		c.logger.Error("commonUDPOnce: request timeout: %v", err)
		c.tunnelPool.AddError()
		return
	}

	c.logger.Debug("Tunnel connection: get %v <- pool active %v", id, c.tunnelPool.Active())
	c.logger.Debug("Tunnel connection: %v <-> %v", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

	defer func() {
		// 池连接关闭
		if remoteConn != nil {
			remoteConn.Close()
			c.logger.Debug("Tunnel connection: closed %v", id)
		}
	}()

	var targetConn net.Conn
	sessionKey := signalURL.Host
	isNewSession := false

	// 获取或创建目标UDP会话
	if session, ok := c.targetUDPSession.Load(sessionKey); ok {
		targetConn = session.(net.Conn)
		c.logger.Debug("Using UDP session: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
	} else {
		// 创建新的会话
		isNewSession = true

		// 尝试获取UDP连接槽位
		if !c.tryAcquireSlot(true) {
			c.logger.Error("commonUDPOnce: UDP slot limit reached: %v/%v", c.udpSlot, c.slotLimit)
			return
		}

		// 创建新的会话
		newSession, err := c.dialWithRotation("udp", udpDialTimeout)
		if err != nil {
			c.logger.Error("commonUDPOnce: dialWithRotation failed: %v", err)
			c.releaseSlot(true)
			return
		}
		targetConn = &conn.StatConn{Conn: newSession, RX: &c.udpRX, TX: &c.udpTX, Rate: c.rateLimiter}
		c.targetUDPSession.Store(sessionKey, targetConn)
		c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
	}

	if isNewSession {
		defer func() {
			// 清理UDP会话和释放槽位
			c.targetUDPSession.Delete(sessionKey)
			if targetConn != nil {
				targetConn.Close()
			}
			c.releaseSlot(true)
		}()
	}

	c.logger.Debug("Starting transfer: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())

	done := make(chan struct{}, 2)

	go func() {
		defer func() { done <- struct{}{} }()

		buffer := c.getUDPBuffer()
		defer c.putUDPBuffer(buffer)
		reader := &conn.TimeoutReader{Conn: remoteConn, Timeout: udpReadTimeout}

		for c.ctx.Err() == nil {
			// 从隧道连接读取数据
			x, err := reader.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					c.logger.Debug("UDP session abort: %v", err)
				} else if err != io.EOF {
					c.logger.Error("commonUDPOnce: read from tunnel failed: %v", err)
				}
				return
			}

			// 将数据写入目标UDP连接
			_, err = targetConn.Write(buffer[:x])
			if err != nil {
				if err != io.EOF {
					c.logger.Error("commonUDPOnce: write to target failed: %v", err)
				}
				return
			}

			// 传输完成
			c.logger.Debug("Transfer complete: %v <-> %v", remoteConn.LocalAddr(), targetConn.LocalAddr())
		}
	}()

	go func() {
		defer func() { done <- struct{}{} }()

		buffer := c.getUDPBuffer()
		defer c.putUDPBuffer(buffer)
		reader := &conn.TimeoutReader{Conn: targetConn, Timeout: udpReadTimeout}

		for c.ctx.Err() == nil {
			// 从目标UDP连接读取数据
			x, err := reader.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					c.logger.Debug("UDP session abort: %v", err)
				} else if err != io.EOF {
					c.logger.Error("commonUDPOnce: read from target failed: %v", err)
				}
				return
			}

			// 将数据写回隧道连接
			_, err = remoteConn.Write(buffer[:x])
			if err != nil {
				if err != io.EOF {
					c.logger.Error("commonUDPOnce: write to tunnel failed: %v", err)
				}
				return
			}

			// 传输完成
			c.logger.Debug("Transfer complete: %v <-> %v", targetConn.LocalAddr(), remoteConn.LocalAddr())
		}
	}()

	// 等待任一协程完成
	<-done
}

// singleControl 单端控制处理循环
func (c *Common) singleControl() error {
	errChan := make(chan error, 3)

	// 启动单端控制、TCP和UDP处理循环
	if len(c.targetTCPAddrs) > 0 {
		go func() { errChan <- c.singleEventLoop() }()
	}
	if c.tunnelListener != nil || c.disableTCP != "1" {
		go func() { errChan <- c.singleTCPLoop() }()
	}
	if c.tunnelUDPConn != nil || c.disableUDP != "1" {
		go func() { errChan <- c.singleUDPLoop() }()
	}

	select {
	case <-c.ctx.Done():
		return fmt.Errorf("singleControl: context error: %w", c.ctx.Err())
	case err := <-errChan:
		return fmt.Errorf("singleControl: %w", err)
	}
}

// singleEventLoop 单端转发事件循环
func (c *Common) singleEventLoop() error {
	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()

	for c.ctx.Err() == nil {
		ping := 0
		now := time.Now()

		// 尝试连接到目标地址
		if conn, err := net.DialTimeout("tcp", c.targetTCPAddrs[c.nextTargetIdx()].String(), reportInterval); err == nil {
			ping = int(time.Since(now).Milliseconds())
			conn.Close()
		}

		// 发送检查点事件
		c.logger.Event("CHECK_POINT|MODE=%v|PING=%vms|POOL=0|TCPS=%v|UDPS=%v|TCPRX=%v|TCPTX=%v|UDPRX=%v|UDPTX=%v", c.runMode, ping,
			atomic.LoadInt32(&c.tcpSlot), atomic.LoadInt32(&c.udpSlot),
			atomic.LoadUint64(&c.tcpRX), atomic.LoadUint64(&c.tcpTX),
			atomic.LoadUint64(&c.udpRX), atomic.LoadUint64(&c.udpTX))

		// 等待下一个报告间隔
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("singleEventLoop: context error: %w", c.ctx.Err())
		case <-ticker.C:
		}
	}

	return fmt.Errorf("singleEventLoop: context error: %w", c.ctx.Err())
}

// singleTCPLoop 单端转发TCP处理循环
func (c *Common) singleTCPLoop() error {
	for c.ctx.Err() == nil {
		// 接受来自隧道的TCP连接
		tunnelConn, err := c.tunnelListener.Accept()
		if err != nil {
			if c.ctx.Err() != nil || err == net.ErrClosed {
				return fmt.Errorf("singleTCPLoop: context error: %w", c.ctx.Err())
			}
			c.logger.Error("singleTCPLoop: accept failed: %v", err)

			select {
			case <-c.ctx.Done():
				return fmt.Errorf("singleTCPLoop: context error: %w", c.ctx.Err())
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		tunnelConn = &conn.StatConn{Conn: tunnelConn, RX: &c.tcpRX, TX: &c.tcpTX, Rate: c.rateLimiter}
		c.logger.Debug("Tunnel connection: %v <-> %v", tunnelConn.LocalAddr(), tunnelConn.RemoteAddr())

		go func(tunnelConn net.Conn) {
			defer func() {
				if tunnelConn != nil {
					tunnelConn.Close()
				}
			}()

			// 尝试获取TCP连接槽位
			if !c.tryAcquireSlot(false) {
				c.logger.Error("singleTCPLoop: TCP slot limit reached: %v/%v", c.tcpSlot, c.slotLimit)
				return
			}

			defer c.releaseSlot(false)

			// 尝试建立目标连接
			targetConn, err := c.dialWithRotation("tcp", tcpDialTimeout)
			if err != nil {
				c.logger.Error("singleTCPLoop: dialWithRotation failed: %v", err)
				return
			}

			defer func() {
				if targetConn != nil {
					targetConn.Close()
				}
			}()

			c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())

			// 发送PROXY v1
			if err := c.sendProxyV1Header(tunnelConn.RemoteAddr().String(), targetConn); err != nil {
				c.logger.Error("singleTCPLoop: sendProxyV1Header failed: %v", err)
				return
			}
			buffer1 := c.getTCPBuffer()
			buffer2 := c.getTCPBuffer()
			defer func() {
				c.putTCPBuffer(buffer1)
				c.putTCPBuffer(buffer2)
			}()

			// 交换数据
			c.logger.Debug("Starting exchange: %v <-> %v", tunnelConn.LocalAddr(), targetConn.LocalAddr())
			c.logger.Debug("Exchange complete: %v", conn.DataExchange(tunnelConn, targetConn, c.readTimeout, buffer1, buffer2))
		}(tunnelConn)
	}

	return fmt.Errorf("singleTCPLoop: context error: %w", c.ctx.Err())
}

// singleUDPLoop 单端转发UDP处理循环
func (c *Common) singleUDPLoop() error {
	for c.ctx.Err() == nil {
		buffer := c.getUDPBuffer()

		// 读取来自隧道的UDP数据
		x, clientAddr, err := c.tunnelUDPConn.ReadFromUDP(buffer)
		if err != nil {
			if c.ctx.Err() != nil || err == net.ErrClosed {
				c.putUDPBuffer(buffer)
				return fmt.Errorf("singleUDPLoop: context error: %w", c.ctx.Err())
			}
			c.logger.Error("singleUDPLoop: ReadFromUDP failed: %v", err)

			c.putUDPBuffer(buffer)
			select {
			case <-c.ctx.Done():
				return fmt.Errorf("singleUDPLoop: context error: %w", c.ctx.Err())
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		c.logger.Debug("Tunnel connection: %v <-> %v", c.tunnelUDPConn.LocalAddr(), clientAddr)

		var targetConn net.Conn
		sessionKey := clientAddr.String()

		// 获取或创建目标UDP会话
		if session, ok := c.targetUDPSession.Load(sessionKey); ok {
			// 复用现有会话
			targetConn = session.(net.Conn)
			c.logger.Debug("Using UDP session: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())
		} else {
			// 尝试获取UDP连接槽位
			if !c.tryAcquireSlot(true) {
				c.logger.Error("singleUDPLoop: UDP slot limit reached: %v/%v", c.udpSlot, c.slotLimit)
				c.putUDPBuffer(buffer)
				continue
			}

			// 创建新的会话
			newSession, err := c.dialWithRotation("udp", udpDialTimeout)
			if err != nil {
				c.logger.Error("singleUDPLoop: dialWithRotation failed: %v", err)
				c.releaseSlot(true)
				c.putUDPBuffer(buffer)
				continue
			}
			targetConn = newSession
			c.targetUDPSession.Store(sessionKey, newSession)
			c.logger.Debug("Target connection: %v <-> %v", targetConn.LocalAddr(), targetConn.RemoteAddr())

			go func(targetConn net.Conn, clientAddr *net.UDPAddr, sessionKey string) {
				defer func() {
					if targetConn != nil {
						targetConn.Close()
					}
					c.releaseSlot(true)
				}()

				buffer := c.getUDPBuffer()
				defer c.putUDPBuffer(buffer)
				reader := &conn.TimeoutReader{Conn: targetConn, Timeout: udpReadTimeout}

				for c.ctx.Err() == nil {
					// 从UDP读取响应
					x, err := reader.Read(buffer)
					if err != nil {
						if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
							c.logger.Debug("UDP session abort: %v", err)
						} else if err != io.EOF {
							c.logger.Error("singleUDPLoop: read from target failed: %v", err)
						}
						c.targetUDPSession.Delete(sessionKey)
						if targetConn != nil {
							targetConn.Close()
						}
						return
					}

					// 将响应写回隧道UDP连接
					_, err = c.tunnelUDPConn.WriteToUDP(buffer[:x], clientAddr)
					if err != nil {
						if err != io.EOF {
							c.logger.Error("singleUDPLoop: writeToUDP failed: %v", err)
						}
						c.targetUDPSession.Delete(sessionKey)
						if targetConn != nil {
							targetConn.Close()
						}
						return
					}
					// 传输完成
					c.logger.Debug("Transfer complete: %v <-> %v", c.tunnelUDPConn.LocalAddr(), targetConn.LocalAddr())
				}
			}(targetConn, clientAddr, sessionKey)
		}

		// 将初始数据发送到目标UDP连接
		c.logger.Debug("Starting transfer: %v <-> %v", targetConn.LocalAddr(), c.tunnelUDPConn.LocalAddr())
		_, err = targetConn.Write(buffer[:x])
		if err != nil {
			if err != io.EOF {
				c.logger.Error("singleUDPLoop: write to target failed: %v", err)
			}
			c.targetUDPSession.Delete(sessionKey)
			if targetConn != nil {
				targetConn.Close()
			}
			c.putUDPBuffer(buffer)
			return fmt.Errorf("singleUDPLoop: write to target failed: %w", err)
		}

		// 传输完成
		c.logger.Debug("Transfer complete: %v <-> %v", targetConn.LocalAddr(), c.tunnelUDPConn.LocalAddr())
		c.putUDPBuffer(buffer)
	}

	return fmt.Errorf("singleUDPLoop: context error: %w", c.ctx.Err())
}
