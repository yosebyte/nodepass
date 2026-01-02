// 内部包，提供共享功能
package internal

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
	parsedURL        *url.URL           // 解析后的URL
	logger           *logs.Logger       // 日志记录器
	dnsCacheTTL      time.Duration      // DNS缓存TTL
	dnsCacheEntries  sync.Map           // DNS缓存条目
	tlsCode          string             // TLS模式代码
	tlsConfig        *tls.Config        // TLS配置
	coreType         string             // 核心类型
	runMode          string             // 运行模式
	poolType         string             // 连接池类型
	dataFlow         string             // 数据流向
	serverName       string             // 服务器名称
	serverPort       string             // 服务器端口
	clientIP         string             // 客户端地址
	dialerIP         string             // 拨号本地IP
	dialerFallback   uint32             // 拨号回落标志
	tunnelKey        string             // 隧道密钥
	tunnelAddr       string             // 原始隧道地址
	tunnelTCPAddr    *net.TCPAddr       // 隧道TCP地址
	tunnelUDPAddr    *net.UDPAddr       // 隧道UDP地址
	targetAddrs      []string           // 原始目标地址组
	targetTCPAddrs   []*net.TCPAddr     // 目标TCP地址组
	targetUDPAddrs   []*net.UDPAddr     // 目标UDP地址组
	targetIdx        uint64             // 目标地址索引
	targetListener   *net.TCPListener   // 目标监听器
	tunnelListener   net.Listener       // 隧道监听器
	controlConn      net.Conn           // 隧道控制连接
	tunnelUDPConn    *conn.StatConn     // 隧道UDP连接
	targetUDPConn    *conn.StatConn     // 目标UDP连接
	targetUDPSession sync.Map           // 目标UDP会话
	tunnelPool       TransportPool      // 隧道连接池
	minPoolCapacity  int                // 最小池容量
	maxPoolCapacity  int                // 最大池容量
	proxyProtocol    string             // 代理协议
	blockProtocol    string             // 屏蔽协议
	blockSOCKS       bool               // 屏蔽SOCKS协议
	blockHTTP        bool               // 屏蔽HTTP协议
	blockTLS         bool               // 屏蔽TLS协议
	disableTCP       string             // 禁用TCP
	disableUDP       string             // 禁用UDP
	rateLimit        int                // 速率限制
	rateLimiter      *conn.RateLimiter  // 全局限速器
	readTimeout      time.Duration      // 读取超时
	bufReader        *bufio.Reader      // 缓冲读取器
	tcpBufferPool    *sync.Pool         // TCP缓冲区池
	udpBufferPool    *sync.Pool         // UDP缓冲区池
	signalChan       chan Signal        // 信号通道
	writeChan        chan []byte        // 写入通道
	verifyChan       chan struct{}      // 证书验证通道
	handshakeStart   time.Time          // 握手开始时间
	checkPoint       time.Time          // 检查点时间
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

// dnsCacheEntry DNS缓存条目
type dnsCacheEntry struct {
	tcpAddr   *net.TCPAddr
	udpAddr   *net.UDPAddr
	expiredAt time.Time
}

// readerConn 包装自定义读取器
type readerConn struct {
	net.Conn
	reader io.Reader
}

// Read 实现自定义读
func (rc *readerConn) Read(b []byte) (int, error) {
	return rc.reader.Read(b)
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

// Signal 操作信号结构体
type Signal struct {
	ActionType  string `json:"action"`           // 操作类型
	RemoteAddr  string `json:"remote,omitempty"` // 远程地址
	PoolConnID  string `json:"id,omitempty"`     // 池连接ID
	Fingerprint string `json:"fp,omitempty"`     // TLS指纹
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

// 常量定义
const (
	contextCheckInterval = 50 * time.Millisecond // 上下文检查间隔
	defaultDNSTTL        = 5 * time.Minute       // 默认DNS缓存TTL
	defaultMinPool       = 64                    // 默认最小池容量
	defaultMaxPool       = 1024                  // 默认最大池容量
	defaultServerName    = "none"                // 默认服务器名称
	defaultRunMode       = "0"                   // 默认运行模式
	defaultPoolType      = "0"                   // 默认连接池类型
	defaultDialerIP      = "auto"                // 默认拨号本地IP
	defaultReadTimeout   = 0 * time.Second       // 默认读取超时
	defaultRateLimit     = 0                     // 默认速率限制
	defaultSlotLimit     = 65536                 // 默认槽位限制
	defaultProxyProtocol = "0"                   // 默认代理协议
	defaultBlockProtocol = "0"                   // 默认协议屏蔽
	defaultTCPStrategy   = "0"                   // 默认TCP策略
	defaultUDPStrategy   = "0"                   // 默认UDP策略
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

// generateAuthToken 生成认证令牌
func (c *Common) generateAuthToken() string {
	return hex.EncodeToString(hmac.New(sha256.New, []byte(c.tunnelKey)).Sum(nil))
}

// verifyAuthToken 验证认证令牌
func (c *Common) verifyAuthToken(token string) bool {
	return hmac.Equal([]byte(token), []byte(c.generateAuthToken()))
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

// resolve 解析地址并缓存
func (c *Common) resolve(network, address string) (any, error) {
	now := time.Now()

	// 快速路径：检查缓存
	if val, ok := c.dnsCacheEntries.Load(address); ok {
		entry := val.(*dnsCacheEntry)
		if now.Before(entry.expiredAt) {
			if network == "tcp" {
				return entry.tcpAddr, nil
			}
			return entry.udpAddr, nil
		}
		// 删除过期缓存
		c.dnsCacheEntries.Delete(address)
	}

	// 慢速路径：系统解析
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("resolve: resolveTCPAddr failed: %w", err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("resolve: resolveUDPAddr failed: %w", err)
	}

	// 存储新的缓存
	entry := &dnsCacheEntry{
		tcpAddr:   tcpAddr,
		udpAddr:   udpAddr,
		expiredAt: now.Add(c.dnsCacheTTL),
	}
	c.dnsCacheEntries.LoadOrStore(address, entry)

	if network == "tcp" {
		return tcpAddr, nil
	}
	return udpAddr, nil
}

// clearCache 清空DNS缓存
func (c *Common) clearCache() {
	c.dnsCacheEntries.Range(func(key, value any) bool {
		c.dnsCacheEntries.Delete(key)
		return true
	})
}

// resolveAddr 解析单个地址
func (c *Common) resolveAddr(network, address string) (any, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %s: %w", address, err)
	}

	if host == "" || net.ParseIP(host) != nil {
		if network == "tcp" {
			return net.ResolveTCPAddr("tcp", address)
		}
		return net.ResolveUDPAddr("udp", address)
	}

	return c.resolve(network, address)
}

// resolveTarget 动态解析目标地址
func (c *Common) resolveTarget(network string, idx int) (any, error) {
	if idx < 0 || idx >= len(c.targetAddrs) {
		return nil, fmt.Errorf("resolveTarget: index %d out of range", idx)
	}

	addr, err := c.resolveAddr(network, c.targetAddrs[idx])
	if err != nil {
		if network == "tcp" {
			return c.targetTCPAddrs[idx], err
		}
		return c.targetUDPAddrs[idx], err
	}
	return addr, nil
}

// getTunnelTCPAddr 动态解析隧道TCP地址
func (c *Common) getTunnelTCPAddr() (*net.TCPAddr, error) {
	addr, err := c.resolveAddr("tcp", c.tunnelAddr)
	if err != nil {
		return c.tunnelTCPAddr, err
	}
	return addr.(*net.TCPAddr), nil
}

// getTunnelUDPAddr 动态解析隧道UDP地址
func (c *Common) getTunnelUDPAddr() (*net.UDPAddr, error) {
	addr, err := c.resolveAddr("udp", c.tunnelAddr)
	if err != nil {
		return c.tunnelUDPAddr, err
	}
	return addr.(*net.UDPAddr), nil
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
	addrCount := len(c.targetAddrs)

	getAddr := func(i int) string {
		addr, _ := c.resolveTarget(network, i)
		if tcpAddr, ok := addr.(*net.TCPAddr); ok {
			return tcpAddr.String()
		}
		if udpAddr, ok := addr.(*net.UDPAddr); ok {
			return udpAddr.String()
		}
		return ""
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
		if addr := getAddr(0); addr != "" {
			return tryDial(addr)
		}
		return nil, fmt.Errorf("dialWithRotation: invalid target address")
	}

	// 多目标地址：负载均衡 + 故障转移
	startIdx := c.nextTargetIdx()
	var lastErr error
	for i := range addrCount {
		addr := getAddr((startIdx + i) % addrCount)
		if addr == "" {
			continue
		}
		conn, err := tryDial(addr)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("dialWithRotation: all %d targets failed: %w", addrCount, lastErr)
}

// getAddress 解析和设置地址信息
func (c *Common) getAddress() error {
	// 解析隧道地址
	tunnelAddr := c.parsedURL.Host
	if tunnelAddr == "" {
		return fmt.Errorf("getAddress: no valid tunnel address found")
	}

	// 保存原始隧道地址
	c.tunnelAddr = tunnelAddr
	if name, port, err := net.SplitHostPort(tunnelAddr); err == nil {
		c.serverName, c.serverPort = name, port
	}

	// 解析隧道TCP地址
	tcpAddr, err := c.resolveAddr("tcp", tunnelAddr)
	if err != nil {
		return fmt.Errorf("getAddress: resolveTCPAddr failed: %w", err)
	}
	c.tunnelTCPAddr = tcpAddr.(*net.TCPAddr)

	// 解析隧道UDP地址
	udpAddr, err := c.resolveAddr("udp", tunnelAddr)
	if err != nil {
		return fmt.Errorf("getAddress: resolveUDPAddr failed: %w", err)
	}
	c.tunnelUDPAddr = udpAddr.(*net.UDPAddr)

	// 处理目标地址组
	targetAddr := strings.TrimPrefix(c.parsedURL.Path, "/")
	if targetAddr == "" {
		return fmt.Errorf("getAddress: no valid target address found")
	}

	addrList := strings.Split(targetAddr, ",")
	tempTCPAddrs := make([]*net.TCPAddr, 0, len(addrList))
	tempUDPAddrs := make([]*net.UDPAddr, 0, len(addrList))
	tempRawAddrs := make([]string, 0, len(addrList))

	for _, addr := range addrList {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}

		// 解析目标TCP地址
		tcpAddr, err := c.resolveAddr("tcp", addr)
		if err != nil {
			return fmt.Errorf("getAddress: resolveTCPAddr failed for %s: %w", addr, err)
		}

		// 解析目标UDP地址
		udpAddr, err := c.resolveAddr("udp", addr)
		if err != nil {
			return fmt.Errorf("getAddress: resolveUDPAddr failed for %s: %w", addr, err)
		}

		tempTCPAddrs = append(tempTCPAddrs, tcpAddr.(*net.TCPAddr))
		tempUDPAddrs = append(tempUDPAddrs, udpAddr.(*net.UDPAddr))
		tempRawAddrs = append(tempRawAddrs, addr)
	}

	if len(tempTCPAddrs) == 0 || len(tempUDPAddrs) == 0 || len(tempTCPAddrs) != len(tempUDPAddrs) {
		return fmt.Errorf("getAddress: no valid target address found")
	}

	// 设置目标地址组
	c.targetAddrs = tempRawAddrs
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
func (c *Common) getCoreType() {
	c.coreType = c.parsedURL.Scheme
}

// getTunnelKey 从URL中获取隧道密钥
func (c *Common) getTunnelKey() {
	if key := c.parsedURL.User.Username(); key != "" {
		c.tunnelKey = key
	} else {
		hash := fnv.New32a()
		hash.Write([]byte(c.parsedURL.Port()))
		c.tunnelKey = hex.EncodeToString(hash.Sum(nil))
	}
}

// getDNSTTL 获取DNS缓存TTL
func (c *Common) getDNSTTL() {
	if dns := c.parsedURL.Query().Get("dns"); dns != "" {
		if ttl, err := time.ParseDuration(dns); err == nil && ttl > 0 {
			c.dnsCacheTTL = ttl
		}
	} else {
		c.dnsCacheTTL = defaultDNSTTL
	}
}

// getServerName 获取服务器名称
func (c *Common) getServerName() {
	if serverName := c.parsedURL.Query().Get("sni"); serverName != "" {
		c.serverName = serverName
		return
	}
	if c.serverName == "" || net.ParseIP(c.serverName) != nil {
		c.serverName = defaultServerName
	}
}

// getPoolCapacity 获取连接池容量设置
func (c *Common) getPoolCapacity() {
	if min := c.parsedURL.Query().Get("min"); min != "" {
		if value, err := strconv.Atoi(min); err == nil && value > 0 {
			c.minPoolCapacity = value
		}
	} else {
		c.minPoolCapacity = defaultMinPool
	}

	if max := c.parsedURL.Query().Get("max"); max != "" {
		if value, err := strconv.Atoi(max); err == nil && value > 0 {
			c.maxPoolCapacity = value
		}
	} else {
		c.maxPoolCapacity = defaultMaxPool
	}
}

// getRunMode 获取运行模式
func (c *Common) getRunMode() {
	if mode := c.parsedURL.Query().Get("mode"); mode != "" {
		c.runMode = mode
	} else {
		c.runMode = defaultRunMode
	}
}

// getPoolType 获取连接池类型
func (c *Common) getPoolType() {
	if poolType := c.parsedURL.Query().Get("type"); poolType != "" {
		c.poolType = poolType
	} else {
		c.poolType = defaultPoolType
	}
	if c.poolType == "1" && c.tlsCode == "0" {
		c.tlsCode = "1"
	}
}

// getDialerIP 获取拨号本地IP设置
func (c *Common) getDialerIP() {
	if dialerIP := c.parsedURL.Query().Get("dial"); dialerIP != "" && dialerIP != "auto" {
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
func (c *Common) getReadTimeout() {
	if timeout := c.parsedURL.Query().Get("read"); timeout != "" {
		if value, err := time.ParseDuration(timeout); err == nil && value > 0 {
			c.readTimeout = value
		}
	} else {
		c.readTimeout = defaultReadTimeout
	}
}

// getRateLimit 获取速率限制
func (c *Common) getRateLimit() {
	if limit := c.parsedURL.Query().Get("rate"); limit != "" {
		if value, err := strconv.Atoi(limit); err == nil && value > 0 {
			c.rateLimit = value * 125000
		}
	} else {
		c.rateLimit = defaultRateLimit
	}
}

// getSlotLimit 获取连接槽位限制
func (c *Common) getSlotLimit() {
	if slot := c.parsedURL.Query().Get("slot"); slot != "" {
		if value, err := strconv.Atoi(slot); err == nil && value > 0 {
			c.slotLimit = int32(value)
		}
	} else {
		c.slotLimit = defaultSlotLimit
	}
}

// getProxyProtocol 获取代理协议设置
func (c *Common) getProxyProtocol() {
	if protocol := c.parsedURL.Query().Get("proxy"); protocol != "" {
		c.proxyProtocol = protocol
	} else {
		c.proxyProtocol = defaultProxyProtocol
	}
}

// getBlockProtocol 获取屏蔽协议设置
func (c *Common) getBlockProtocol() {
	if protocol := c.parsedURL.Query().Get("block"); protocol != "" {
		c.blockProtocol = protocol
	} else {
		c.blockProtocol = defaultBlockProtocol
	}
	c.blockSOCKS = strings.Contains(c.blockProtocol, "1")
	c.blockHTTP = strings.Contains(c.blockProtocol, "2")
	c.blockTLS = strings.Contains(c.blockProtocol, "3")
}

// getTCPStrategy 获取TCP策略
func (c *Common) getTCPStrategy() {
	if tcpStrategy := c.parsedURL.Query().Get("notcp"); tcpStrategy != "" {
		c.disableTCP = tcpStrategy
	} else {
		c.disableTCP = defaultTCPStrategy
	}
}

// getUDPStrategy 获取UDP策略
func (c *Common) getUDPStrategy() {
	if udpStrategy := c.parsedURL.Query().Get("noudp"); udpStrategy != "" {
		c.disableUDP = udpStrategy
	} else {
		c.disableUDP = defaultUDPStrategy
	}
}

// initConfig 初始化配置
func (c *Common) initConfig() error {
	if err := c.getAddress(); err != nil {
		return err
	}

	c.getCoreType()
	c.getDNSTTL()
	c.getTunnelKey()
	c.getPoolCapacity()
	c.getServerName()
	c.getRunMode()
	c.getPoolType()
	c.getDialerIP()
	c.getReadTimeout()
	c.getRateLimit()
	c.getSlotLimit()
	c.getProxyProtocol()
	c.getBlockProtocol()
	c.getTCPStrategy()
	c.getUDPStrategy()

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

// detectBlockProtocol 检测屏蔽协议
func (c *Common) detectBlockProtocol(conn net.Conn) (string, net.Conn) {
	if !c.blockSOCKS && !c.blockHTTP && !c.blockTLS {
		return "", conn
	}

	reader := bufio.NewReader(conn)
	b, err := reader.Peek(8)
	if err != nil || len(b) < 1 {
		return "", &readerConn{Conn: conn, reader: reader}
	}

	// 检测SOCKS
	if c.blockSOCKS && len(b) >= 2 {
		if b[0] == 0x04 && (b[1] == 0x01 || b[1] == 0x02) {
			return "SOCKS4", &readerConn{Conn: conn, reader: reader}
		}
		if b[0] == 0x05 && b[1] >= 0x01 && b[1] <= 0x03 {
			return "SOCKS5", &readerConn{Conn: conn, reader: reader}
		}
	}

	// 检测HTTP
	if c.blockHTTP && len(b) >= 4 && b[0] >= 'A' && b[0] <= 'Z' {
		for i, c := range b[1:] {
			if c == ' ' {
				return "HTTP", &readerConn{Conn: conn, reader: reader}
			}
			if c < 'A' || c > 'Z' || i >= 7 {
				break
			}
		}
	}

	// 检测TLS
	if c.blockTLS && b[0] == 0x16 {
		return "TLS", &readerConn{Conn: conn, reader: reader}
	}

	return "", &readerConn{Conn: conn, reader: reader}
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
	if len(c.targetAddrs) == 0 {
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

	// 关闭隧道控制连接
	if c.controlConn != nil {
		c.controlConn.Close()
		c.logger.Debug("Control connection closed: %v", c.controlConn.LocalAddr())
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
	drain(c.writeChan)
	drain(c.verifyChan)

	// 重置全局限速器
	if c.rateLimiter != nil {
		c.rateLimiter.Reset()
	}

	// 清空DNS缓存
	c.clearCache()
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

// setControlConn 设置控制连接
func (c *Common) setControlConn() error {
	start := time.Now()
	for c.ctx.Err() == nil {
		if c.tunnelPool.Ready() && c.tunnelPool.Active() > 0 {
			break
		}
		if time.Since(start) > handshakeTimeout {
			return fmt.Errorf("setControlConn: handshake timeout")
		}
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("setControlConn: context error: %w", c.ctx.Err())
		case <-time.After(contextCheckInterval):
		}
	}

	poolConn, err := c.tunnelPool.OutgoingGet("00000000", poolGetTimeout)
	if err != nil {
		return fmt.Errorf("setControlConn: outgoingGet failed: %w", err)
	}
	c.controlConn = poolConn
	c.bufReader = bufio.NewReader(&conn.TimeoutReader{Conn: c.controlConn, Timeout: 3 * reportInterval})
	c.logger.Info("Marking tunnel handshake as complete in %vms", time.Since(c.handshakeStart).Milliseconds())

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case data := <-c.writeChan:
				_, err := c.controlConn.Write(data)
				if err != nil {
					c.logger.Error("startWriter: write failed: %v", err)
				}
			}
		}
	}()

	if c.tlsCode == "1" || c.tlsCode == "2" {
		c.logger.Info("TLS certificate fingerprint verifying...")
	}
	return nil
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
			case <-time.After(contextCheckInterval):
			}
			continue
		}

		// 解析JSON信号
		var signal Signal
		if err := json.Unmarshal(signalData, &signal); err != nil {
			c.logger.Error("commonQueue: unmarshal signal failed: %v", err)
			select {
			case <-c.ctx.Done():
				return fmt.Errorf("commonQueue: context error: %w", c.ctx.Err())
			case <-time.After(contextCheckInterval):
			}
			continue
		}

		// 将信号发送到通道
		select {
		case c.signalChan <- signal:
		default:
			c.logger.Error("commonQueue: queue limit reached: %v", semaphoreLimit)
			select {
			case <-c.ctx.Done():
				return fmt.Errorf("commonQueue: context error: %w", c.ctx.Err())
			case <-time.After(contextCheckInterval):
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
		// 连接池健康度检查
		if c.tunnelPool.ErrorCount() > c.tunnelPool.Active()/2 {
			// 发送刷新信号到对端
			if c.ctx.Err() == nil && c.controlConn != nil {
				signalData, _ := json.Marshal(Signal{ActionType: "flush"})
				c.writeChan <- c.encode(signalData)
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
		if c.ctx.Err() == nil && c.controlConn != nil {
			signalData, _ := json.Marshal(Signal{ActionType: "ping"})
			c.writeChan <- c.encode(signalData)
		}
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
		if c.tunnelPool.Ready() && c.tunnelPool.Active() > 0 {
			break
		}
		select {
		case <-c.ctx.Done():
			continue
		case <-time.After(contextCheckInterval):
		}
	}

	id, testConn, err := c.tunnelPool.IncomingGet(poolGetTimeout)
	if err != nil {
		c.logger.Error("incomingVerify: incomingGet failed: %v", err)
		c.cancel()
		return
	}
	defer testConn.Close()

	// 获取证书指纹
	var fingerprint string
	switch c.coreType {
	case "server":
		if c.tlsConfig != nil && len(c.tlsConfig.Certificates) > 0 {
			cert := c.tlsConfig.Certificates[0]
			if len(cert.Certificate) > 0 {
				fingerprint = c.formatCertFingerprint(cert.Certificate[0])
			}
		}
	case "client":
		if conn, ok := testConn.(interface{ ConnectionState() tls.ConnectionState }); ok {
			state := conn.ConnectionState()
			if len(state.PeerCertificates) > 0 {
				fingerprint = c.formatCertFingerprint(state.PeerCertificates[0].Raw)
			}
		}
	}

	// 构建并发送验证信号
	if c.ctx.Err() == nil && c.controlConn != nil {
		signalData, _ := json.Marshal(Signal{
			ActionType:  "verify",
			PoolConnID:  id,
			Fingerprint: fingerprint,
		})
		c.writeChan <- c.encode(signalData)
	}

	c.logger.Debug("TLS verify signal: cid %v -> %v", id, c.controlConn.RemoteAddr())
}

// commonLoop 共用处理循环
func (c *Common) commonLoop() {
	for c.ctx.Err() == nil {
		// 等待连接池准备就绪
		if c.tunnelPool.Ready() {
			if c.verifyChan != nil {
				select {
				case <-c.verifyChan:
					// 证书验证完成
				case <-c.ctx.Done():
					return
				}
			}

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
		case <-time.After(contextCheckInterval):
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
			case <-time.After(contextCheckInterval):
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

			// 阻止屏蔽协议
			protocol, wrappedConn := c.detectBlockProtocol(targetConn)
			if protocol != "" {
				c.logger.Warn("commonTCPLoop: blocked %v protocol from %v", protocol, targetConn.RemoteAddr())
				return
			}
			targetConn = wrappedConn

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
			if c.ctx.Err() == nil && c.controlConn != nil {
				signalData, _ := json.Marshal(Signal{
					ActionType: "tcp",
					RemoteAddr: targetConn.RemoteAddr().String(),
					PoolConnID: id,
				})
				c.writeChan <- c.encode(signalData)
			}

			c.logger.Debug("TCP launch signal: cid %v -> %v", id, c.controlConn.RemoteAddr())

			buffer1 := c.getTCPBuffer()
			buffer2 := c.getTCPBuffer()
			defer func() {
				c.putTCPBuffer(buffer1)
				c.putTCPBuffer(buffer2)
			}()

			// 交换数据
			c.logger.Info("Starting exchange: %v <-> %v", targetConn.RemoteAddr(), remoteConn.RemoteAddr())
			c.logger.Info("Exchange complete: %v", conn.DataExchange(targetConn, remoteConn, c.readTimeout, buffer1, buffer2))
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
			case <-time.After(contextCheckInterval):
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
			if c.ctx.Err() == nil && c.controlConn != nil {
				signalData, _ := json.Marshal(Signal{
					ActionType: "udp",
					RemoteAddr: clientAddr.String(),
					PoolConnID: id,
				})
				c.writeChan <- c.encode(signalData)
			}

			c.logger.Debug("UDP launch signal: cid %v -> %v", id, c.controlConn.RemoteAddr())
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
			case <-time.After(contextCheckInterval):
			}
			continue
		}

		select {
		case <-c.ctx.Done():
			return fmt.Errorf("commonOnce: context error: %w", c.ctx.Err())
		case signal := <-c.signalChan:
			// 处理信号
			switch signal.ActionType {
			case "verify":
				if c.tlsCode == "1" || c.tlsCode == "2" {
					go c.outgoingVerify(signal)
				}
			case "tcp":
				if c.disableTCP != "1" {
					go c.commonTCPOnce(signal)
				}
			case "udp":
				if c.disableUDP != "1" {
					go c.commonUDPOnce(signal)
				}
			case "flush":
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
			case "ping":
				if c.ctx.Err() == nil && c.controlConn != nil {
					signalData, _ := json.Marshal(Signal{ActionType: "pong"})
					c.writeChan <- c.encode(signalData)
				}
			case "pong":
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
func (c *Common) outgoingVerify(signal Signal) {
	for c.ctx.Err() == nil {
		if c.tunnelPool.Ready() {
			break
		}
		select {
		case <-c.ctx.Done():
			continue
		case <-time.After(contextCheckInterval):
		}
	}

	fingerPrint := signal.Fingerprint
	if fingerPrint == "" {
		c.logger.Error("outgoingVerify: no fingerprint in signal")
		c.cancel()
		return
	}

	id := signal.PoolConnID
	c.logger.Debug("TLS verify signal: cid %v <- %v", id, c.controlConn.RemoteAddr())

	testConn, err := c.tunnelPool.OutgoingGet(id, poolGetTimeout)
	if err != nil {
		c.logger.Error("outgoingVerify: request timeout: %v", err)
		c.cancel()
		return
	}
	defer testConn.Close()

	// 验证证书指纹
	var serverFingerprint, clientFingerprint string
	switch c.coreType {
	case "server":
		if c.tlsConfig == nil || len(c.tlsConfig.Certificates) == 0 {
			c.logger.Error("outgoingVerify: no local certificate")
			c.cancel()
			return
		}

		cert := c.tlsConfig.Certificates[0]
		if len(cert.Certificate) == 0 {
			c.logger.Error("outgoingVerify: empty local certificate")
			c.cancel()
			return
		}

		serverFingerprint = c.formatCertFingerprint(cert.Certificate[0])
		clientFingerprint = fingerPrint
	case "client":
		conn, ok := testConn.(interface{ ConnectionState() tls.ConnectionState })
		if !ok {
			return
		}
		state := conn.ConnectionState()

		if len(state.PeerCertificates) == 0 {
			c.logger.Error("outgoingVerify: no peer certificates found")
			c.cancel()
			return
		}

		clientFingerprint = c.formatCertFingerprint(state.PeerCertificates[0].Raw)
		serverFingerprint = fingerPrint
	}

	// 验证指纹匹配
	if serverFingerprint != clientFingerprint {
		c.logger.Error("outgoingVerify: certificate fingerprint mismatch: server: %v - client: %v", serverFingerprint, clientFingerprint)
		c.cancel()
		return
	}

	c.logger.Info("TLS certificate fingerprint verified: %v", fingerPrint)

	// 通知验证完成
	if c.verifyChan != nil {
		c.verifyChan <- struct{}{}
	}
}

// commonTCPOnce 共用处理单个TCP请求
func (c *Common) commonTCPOnce(signal Signal) {
	id := signal.PoolConnID
	c.logger.Debug("TCP launch signal: cid %v <- %v", id, c.controlConn.RemoteAddr())

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
	if err := c.sendProxyV1Header(signal.RemoteAddr, targetConn); err != nil {
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
	c.logger.Info("Starting exchange: %v <-> %v", remoteConn.RemoteAddr(), targetConn.RemoteAddr())
	c.logger.Info("Exchange complete: %v", conn.DataExchange(remoteConn, targetConn, c.readTimeout, buffer1, buffer2))
}

// commonUDPOnce 共用处理单个UDP请求
// commonUDPOnce 共用处理单个UDP请求
func (c *Common) commonUDPOnce(signal Signal) {
	id := signal.PoolConnID
	c.logger.Debug("UDP launch signal: cid %v <- %v", id, c.controlConn.RemoteAddr())

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
	sessionKey := signal.RemoteAddr
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

		// 尝试连接到目标地址检测延迟
		idx := c.nextTargetIdx()
		if addr, _ := c.resolveTarget("tcp", idx); addr != nil {
			if tcpAddr, ok := addr.(*net.TCPAddr); ok {
				if conn, err := net.DialTimeout("tcp", tcpAddr.String(), reportInterval); err == nil {
					ping = int(time.Since(now).Milliseconds())
					conn.Close()
				}
			}
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
			case <-time.After(contextCheckInterval):
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

			// 阻止屏蔽协议
			protocol, wrappedConn := c.detectBlockProtocol(tunnelConn)
			if protocol != "" {
				c.logger.Warn("singleTCPLoop: blocked %v protocol from %v", protocol, tunnelConn.RemoteAddr())
				return
			}
			tunnelConn = wrappedConn

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
			c.logger.Info("Starting exchange: %v <-> %v", tunnelConn.RemoteAddr(), targetConn.RemoteAddr())
			c.logger.Info("Exchange complete: %v", conn.DataExchange(tunnelConn, targetConn, c.readTimeout, buffer1, buffer2))
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
			case <-time.After(contextCheckInterval):
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
