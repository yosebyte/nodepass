// 内部包，实现主控模式功能
package internal

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/NodePassProject/logs"
)

// 常量定义
const (
	openAPIVersion = "v1"           // OpenAPI版本
	stateFilePath  = "gob"          // 实例状态持久化文件路径
	stateFileName  = "nodepass.gob" // 实例状态持久化文件名
	sseRetryTime   = 3000           // 重试间隔时间（毫秒）
	apiKeyID       = "********"     // API Key的特殊ID
)

// Swagger UI HTML模板
const swaggerUIHTML = `<!DOCTYPE html>
<html>
<head>
  <title>NodePass API</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => SwaggerUIBundle({
      spec: %s,
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`

// Master 实现主控模式功能
type Master struct {
	Common                            // 继承通用功能
	prefix        string              // API前缀
	version       string              // NP版本
	hostname      string              // 隧道名称
	logLevel      string              // 日志级别
	crtPath       string              // 证书路径
	keyPath       string              // 密钥路径
	instances     sync.Map            // 实例映射表
	server        *http.Server        // HTTP服务器
	tlsConfig     *tls.Config         // TLS配置
	masterURL     *url.URL            // 主控URL
	statePath     string              // 实例状态持久化文件路径
	subscribers   sync.Map            // SSE订阅者映射表
	notifyChannel chan *InstanceEvent // 事件通知通道
	startTime     time.Time           // 启动时间
}

// Instance 实例信息
type Instance struct {
	ID         string             `json:"id"`        // 实例ID
	Alias      string             `json:"alias"`     // 实例别名
	Type       string             `json:"type"`      // 实例类型
	Status     string             `json:"status"`    // 实例状态
	URL        string             `json:"url"`       // 实例URL
	Restart    bool               `json:"restart"`   // 是否自启动
	TCPRX      uint64             `json:"tcprx"`     // TCP接收字节数
	TCPTX      uint64             `json:"tcptx"`     // TCP发送字节数
	UDPRX      uint64             `json:"udprx"`     // UDP接收字节数
	UDPTX      uint64             `json:"udptx"`     // UDP发送字节数
	cmd        *exec.Cmd          `json:"-" gob:"-"` // 命令对象（不序列化）
	stopped    chan struct{}      `json:"-" gob:"-"` // 停止信号通道（不序列化）
	cancelFunc context.CancelFunc `json:"-" gob:"-"` // 取消函数（不序列化）
}

// InstanceEvent 实例事件信息
type InstanceEvent struct {
	Type     string    `json:"type"`           // 事件类型：initial, create, update, delete, shutdown, log
	Time     time.Time `json:"time"`           // 事件时间
	Instance *Instance `json:"instance"`       // 关联的实例
	Logs     string    `json:"logs,omitempty"` // 日志内容，仅当Type为log时有效
}

// InstanceLogWriter 实例日志写入器
type InstanceLogWriter struct {
	instanceID string         // 实例ID
	instance   *Instance      // 实例对象
	target     io.Writer      // 目标写入器
	master     *Master        // 主控对象
	statRegex  *regexp.Regexp // 统计信息正则表达式
}

// NewInstanceLogWriter 创建新的实例日志写入器
func NewInstanceLogWriter(instanceID string, instance *Instance, target io.Writer, master *Master) *InstanceLogWriter {
	return &InstanceLogWriter{
		instanceID: instanceID,
		instance:   instance,
		target:     target,
		master:     master,
		statRegex:  regexp.MustCompile(`TRAFFIC_STATS\|TCP_RX=(\d+)\|TCP_TX=(\d+)\|UDP_RX=(\d+)\|UDP_TX=(\d+)`),
	}
}

// Write 实现io.Writer接口，处理日志输出并解析统计信息
func (w *InstanceLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	scanner := bufio.NewScanner(strings.NewReader(s))

	for scanner.Scan() {
		line := scanner.Text()
		// 解析并处理统计信息
		if matches := w.statRegex.FindStringSubmatch(line); len(matches) == 5 {
			stats := []*uint64{&w.instance.TCPRX, &w.instance.TCPTX, &w.instance.UDPRX, &w.instance.UDPTX}
			for i, stat := range stats {
				if v, err := strconv.ParseUint(matches[i+1], 10, 64); err == nil {
					// 累加新的统计数据
					*stat += v
				}
			}
			w.master.instances.Store(w.instanceID, w.instance)

			// 发送流量更新事件
			w.master.sendSSEEvent("update", w.instance)
		}
		// 输出日志加实例ID
		fmt.Fprintf(w.target, "%s [%s]\n", line, w.instanceID)

		// 发送日志事件
		w.master.sendSSEEvent("log", w.instance, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(w.target, "%s [%s]", s, w.instanceID)
	}
	return len(p), nil
}

// setCorsHeaders 设置跨域响应头
func setCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PATCH, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, Cache-Control")
}

// NewMaster 创建新的主控实例
func NewMaster(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *logs.Logger, version string) *Master {
	// 解析主机地址
	host, err := net.ResolveTCPAddr("tcp", parsedURL.Host)
	if err != nil {
		logger.Error("Resolve failed: %v", err)
		return nil
	}

	// 获取隧道名称
	var hostname string
	if tlsConfig != nil && tlsConfig.ServerName != "" {
		hostname = tlsConfig.ServerName
	} else {
		hostname = parsedURL.Hostname()
	}

	// 设置API前缀
	prefix := parsedURL.Path
	if prefix == "" || prefix == "/" {
		prefix = "/api"
	} else {
		prefix = strings.TrimRight(prefix, "/")
	}

	// 获取应用程序目录作为状态文件存储位置
	execPath, _ := os.Executable()
	baseDir := filepath.Dir(execPath)

	master := &Master{
		Common: Common{
			tlsCode: tlsCode,
			logger:  logger,
		},
		prefix:        fmt.Sprintf("%s/%s", prefix, openAPIVersion),
		version:       version,
		logLevel:      parsedURL.Query().Get("log"),
		crtPath:       parsedURL.Query().Get("crt"),
		keyPath:       parsedURL.Query().Get("key"),
		hostname:      hostname,
		tlsConfig:     tlsConfig,
		masterURL:     parsedURL,
		statePath:     filepath.Join(baseDir, stateFilePath, stateFileName),
		notifyChannel: make(chan *InstanceEvent, 1024),
		startTime:     time.Now(),
	}
	master.tunnelTCPAddr = host

	// 加载持久化的实例状态
	master.loadState()

	// 启动事件分发器
	go master.startEventDispatcher()

	return master
}

// Run 管理主控生命周期
func (m *Master) Run() {
	m.logger.Info("Master started: %v%v", m.tunnelAddr, m.prefix)

	// 初始化API Key
	apiKey, ok := m.findInstance(apiKeyID)
	if !ok {
		// 如果不存在API Key实例，则创建一个
		apiKey = &Instance{
			ID:  apiKeyID,
			URL: generateAPIKey(),
		}
		m.instances.Store(apiKeyID, apiKey)
		m.saveState()
		m.logger.Info("API Key created: %v", apiKey.URL)
	} else {
		m.logger.Info("API Key loaded: %v", apiKey.URL)
	}

	// 设置HTTP路由
	mux := http.NewServeMux()

	// 创建需要API Key认证的端点
	protectedEndpoints := map[string]http.HandlerFunc{
		fmt.Sprintf("%s/instances", m.prefix):  m.handleInstances,
		fmt.Sprintf("%s/instances/", m.prefix): m.handleInstanceDetail,
		fmt.Sprintf("%s/events", m.prefix):     m.handleSSE,
		fmt.Sprintf("%s/info", m.prefix):       m.handleInfo,
	}

	// 创建不需要API Key认证的端点
	publicEndpoints := map[string]http.HandlerFunc{
		fmt.Sprintf("%s/openapi.json", m.prefix): m.handleOpenAPISpec,
		fmt.Sprintf("%s/docs", m.prefix):         m.handleSwaggerUI,
	}

	// API Key 认证中间件
	apiKeyMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 设置跨域响应头
			setCorsHeaders(w)
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// 读取API Key，如果存在的话
			apiKeyInstance, keyExists := m.findInstance(apiKeyID)
			if keyExists && apiKeyInstance.URL != "" {
				// 检查请求头中的API Key
				reqAPIKey := r.Header.Get("X-API-Key")
				if reqAPIKey == "" {
					// API Key不存在，返回未授权错误
					httpError(w, "Unauthorized: API key required", http.StatusUnauthorized)
					return
				}

				// 验证API Key
				if reqAPIKey != apiKeyInstance.URL {
					httpError(w, "Unauthorized: Invalid API key", http.StatusUnauthorized)
					return
				}
			}

			// 调用原始处理器
			next(w, r)
		}
	}

	// CORS 中间件
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 设置跨域响应头
			setCorsHeaders(w)
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next(w, r)
		}
	}

	// 注册受保护的端点
	for path, handler := range protectedEndpoints {
		mux.HandleFunc(path, apiKeyMiddleware(handler))
	}

	// 注册公共端点
	for path, handler := range publicEndpoints {
		mux.HandleFunc(path, corsMiddleware(handler))
	}

	// 创建HTTP服务器
	m.server = &http.Server{
		Addr:      m.tunnelTCPAddr.String(),
		ErrorLog:  m.logger.StdLogger(),
		Handler:   mux,
		TLSConfig: m.tlsConfig,
	}

	// 启动HTTP服务器
	go func() {
		var err error
		if m.tlsConfig != nil {
			err = m.server.ListenAndServeTLS("", "")
		} else {
			err = m.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			m.logger.Error("Listen failed: %v", err)
		}
	}()

	// 处理系统信号
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := m.Shutdown(shutdownCtx); err != nil {
		m.logger.Error("Master shutdown error: %v", err)
	} else {
		m.logger.Info("Master shutdown complete")
	}
}

// Shutdown 关闭主控
func (m *Master) Shutdown(ctx context.Context) error {
	return m.shutdown(ctx, func() {
		// 声明一个已关闭通道的集合，避免重复关闭
		var closedChannels sync.Map

		var wg sync.WaitGroup

		// 给所有订阅者一个关闭通知
		m.subscribers.Range(func(key, value any) bool {
			subscriberChan := value.(chan *InstanceEvent)
			wg.Add(1)
			go func(ch chan *InstanceEvent) {
				defer wg.Done()
				// 非阻塞的方式发送关闭事件
				select {
				case ch <- &InstanceEvent{
					Type: "shutdown",
					Time: time.Now(),
				}:
				default:
					// 不可用，忽略
				}
			}(subscriberChan)
			return true
		})

		// 等待所有订阅者处理完关闭事件
		time.Sleep(100 * time.Millisecond)

		// 关闭所有订阅者通道
		m.subscribers.Range(func(key, value any) bool {
			subscriberChan := value.(chan *InstanceEvent)
			// 检查通道是否已关闭，如果没有则关闭它
			if _, loaded := closedChannels.LoadOrStore(subscriberChan, true); !loaded {
				wg.Add(1)
				go func(k any, ch chan *InstanceEvent) {
					defer wg.Done()
					close(ch)
					m.subscribers.Delete(k)
				}(key, subscriberChan)
			}
			return true
		})

		// 停止所有运行中的实例
		m.instances.Range(func(key, value any) bool {
			instance := value.(*Instance)
			// 如果实例正在运行，则停止它
			if instance.Status == "running" && instance.cmd != nil && instance.cmd.Process != nil {
				wg.Add(1)
				go func(inst *Instance) {
					defer wg.Done()
					m.stopInstance(inst)
				}(instance)
			}
			return true
		})

		wg.Wait()

		// 关闭事件通知通道，停止事件分发器
		close(m.notifyChannel)

		// 保存实例状态
		if err := m.saveState(); err != nil {
			m.logger.Error("Save gob failed: %v", err)
		} else {
			m.logger.Info("Instances saved: %v", m.statePath)
		}

		// 关闭HTTP服务器
		if err := m.server.Shutdown(ctx); err != nil {
			m.logger.Error("ApiSvr shutdown error: %v", err)
		}
	})
}

// saveState 保存实例状态到文件
func (m *Master) saveState() error {
	// 创建持久化数据
	persistentData := make(map[string]*Instance)

	// 从sync.Map转换数据
	m.instances.Range(func(key, value any) bool {
		instance := value.(*Instance)
		persistentData[key.(string)] = instance
		return true
	})

	// 如果没有实例，直接返回
	if len(persistentData) == 0 {
		// 如果状态文件存在，删除它
		if _, err := os.Stat(m.statePath); err == nil {
			return os.Remove(m.statePath)
		}
		return nil
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0755); err != nil {
		m.logger.Error("Create state dir failed: %v", err)
		return err
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp(filepath.Dir(m.statePath), "np-*.tmp")
	if err != nil {
		m.logger.Error("Create temp failed: %v", err)
		return err
	}
	tempPath := tempFile.Name()

	// 删除临时文件的函数，只在错误情况下使用
	removeTemp := func() {
		if _, err := os.Stat(tempPath); err == nil {
			os.Remove(tempPath)
		}
	}

	// 编码数据
	encoder := gob.NewEncoder(tempFile)
	if err := encoder.Encode(persistentData); err != nil {
		m.logger.Error("Encode instances failed: %v", err)
		tempFile.Close()
		removeTemp()
		return err
	}

	// 关闭文件
	if err := tempFile.Close(); err != nil {
		m.logger.Error("Close temp failed: %v", err)
		removeTemp()
		return err
	}

	// 原子地替换文件
	if err := os.Rename(tempPath, m.statePath); err != nil {
		m.logger.Error("Rename temp failed: %v", err)
		removeTemp()
		return err
	}

	return nil
}

// loadState 从文件加载实例状态
func (m *Master) loadState() {
	// 检查文件是否存在
	if _, err := os.Stat(m.statePath); os.IsNotExist(err) {
		return
	}

	// 打开文件
	file, err := os.Open(m.statePath)
	if err != nil {
		m.logger.Error("Open file failed: %v", err)
		return
	}
	defer file.Close()

	// 解码数据
	var persistentData map[string]*Instance
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&persistentData); err != nil {
		m.logger.Error("Decode file failed: %v", err)
		return
	}

	// 恢复实例
	for id, instance := range persistentData {
		instance.stopped = make(chan struct{})
		m.instances.Store(id, instance)

		// 处理自启动
		if instance.Restart {
			go m.startInstance(instance)
			m.logger.Info("Auto-starting instance: %v [%v]", instance.URL, instance.ID)
		}
	}

	m.logger.Info("Loaded %v instances from %v", len(persistentData), m.statePath)
}

// handleOpenAPISpec 处理OpenAPI规范请求
func (m *Master) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(generateOpenAPISpec()))
}

// handleSwaggerUI 处理Swagger UI请求
func (m *Master) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, swaggerUIHTML, generateOpenAPISpec())
}

// handleInfo 处理系统信息请求
func (m *Master) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := map[string]any{
		"os":     runtime.GOOS,
		"arch":   runtime.GOARCH,
		"ver":    m.version,
		"name":   m.hostname,
		"uptime": uint64(time.Since(m.startTime).Seconds()),
		"log":    m.logLevel,
		"tls":    m.tlsCode,
		"crt":    m.crtPath,
		"key":    m.keyPath,
	}

	writeJSON(w, http.StatusOK, info)
}

// handleInstances 处理实例集合请求
func (m *Master) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 获取所有实例
		instances := []*Instance{}
		m.instances.Range(func(_, value any) bool {
			instances = append(instances, value.(*Instance))
			return true
		})
		writeJSON(w, http.StatusOK, instances)

	case http.MethodPost:
		// 创建新实例
		var reqData struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil || reqData.URL == "" {
			httpError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// 解析URL
		parsedURL, err := url.Parse(reqData.URL)
		if err != nil {
			httpError(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		// 验证实例类型
		instanceType := parsedURL.Scheme
		if instanceType != "client" && instanceType != "server" {
			httpError(w, "Invalid URL scheme", http.StatusBadRequest)
			return
		}

		// 生成实例ID
		id := generateID()
		if _, exists := m.instances.Load(id); exists {
			httpError(w, "Instance ID already exists", http.StatusConflict)
			return
		}

		// 创建实例
		instance := &Instance{
			ID:      id,
			Type:    instanceType,
			URL:     m.enhanceURL(reqData.URL, instanceType),
			Status:  "stopped",
			Restart: false,
			stopped: make(chan struct{}),
		}
		m.instances.Store(id, instance)

		// 启动实例
		go m.startInstance(instance)

		// 保存实例状态
		go func() {
			time.Sleep(100 * time.Millisecond)
			m.saveState()
		}()
		writeJSON(w, http.StatusCreated, instance)

		// 发送创建事件
		m.sendSSEEvent("create", instance)

	default:
		httpError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleInstanceDetail 处理单个实例请求
func (m *Master) handleInstanceDetail(w http.ResponseWriter, r *http.Request) {
	// 获取实例ID
	id := strings.TrimPrefix(r.URL.Path, fmt.Sprintf("%s/instances/", m.prefix))
	if id == "" || id == "/" {
		httpError(w, "Instance ID is required", http.StatusBadRequest)
		return
	}

	// 查找实例
	instance, ok := m.findInstance(id)
	if !ok {
		httpError(w, "Instance not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		m.handleGetInstance(w, instance)
	case http.MethodPatch:
		m.handlePatchInstance(w, r, id, instance)
	case http.MethodPut:
		m.handlePutInstance(w, r, id, instance)
	case http.MethodDelete:
		m.handleDeleteInstance(w, id, instance)
	default:
		httpError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetInstance 处理获取实例信息请求
func (m *Master) handleGetInstance(w http.ResponseWriter, instance *Instance) {
	writeJSON(w, http.StatusOK, instance)
}

// handlePatchInstance 处理更新实例状态请求
func (m *Master) handlePatchInstance(w http.ResponseWriter, r *http.Request, id string, instance *Instance) {
	var reqData struct {
		Alias   string `json:"alias,omitempty"`
		Action  string `json:"action,omitempty"`
		Restart *bool  `json:"restart,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqData); err == nil {
		if id == apiKeyID {
			// API Key实例只允许restart操作
			if reqData.Action == "restart" {
				m.regenerateAPIKey(instance)
				// 只有API Key需要在这里发送事件
				m.sendSSEEvent("update", instance)
			}
		} else {
			// 更新自启动设置
			if reqData.Restart != nil && instance.Restart != *reqData.Restart {
				instance.Restart = *reqData.Restart
				m.instances.Store(id, instance)
				m.saveState()
				m.logger.Info("Restart policy updated: %v [%v]", *reqData.Restart, instance.ID)

				// 发送restart策略变更事件
				m.sendSSEEvent("update", instance)
			}

			// 更新实例别名
			if reqData.Alias != "" && instance.Alias != reqData.Alias {
				instance.Alias = reqData.Alias
				m.instances.Store(id, instance)
				m.saveState()
				m.logger.Info("Alias updated: %v [%v]", reqData.Alias, instance.ID)

				// 发送别名变更事件
				m.sendSSEEvent("update", instance)
			}

			// 处理当前实例操作
			if reqData.Action != "" {
				m.processInstanceAction(instance, reqData.Action)
			}
		}
	}
	writeJSON(w, http.StatusOK, instance)
}

// handlePutInstance 处理更新实例URL请求
func (m *Master) handlePutInstance(w http.ResponseWriter, r *http.Request, id string, instance *Instance) {
	// API Key实例不允许修改URL
	if id == apiKeyID {
		httpError(w, "Forbidden: API Key", http.StatusForbidden)
		return
	}

	var reqData struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil || reqData.URL == "" {
		httpError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 解析URL
	parsedURL, err := url.Parse(reqData.URL)
	if err != nil {
		httpError(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	// 验证实例类型
	instanceType := parsedURL.Scheme
	if instanceType != "client" && instanceType != "server" {
		httpError(w, "Invalid URL scheme", http.StatusBadRequest)
		return
	}

	// 增强URL以便进行重复检测
	enhancedURL := m.enhanceURL(reqData.URL, instanceType)

	// 检查是否与当前实例的URL相同
	if instance.URL == enhancedURL {
		httpError(w, "Instance URL conflict", http.StatusConflict)
		return
	}

	// 如果实例正在运行，先停止它
	if instance.Status == "running" {
		m.stopInstance(instance)
		time.Sleep(100 * time.Millisecond)
	}

	// 更新实例URL和类型
	instance.URL = enhancedURL
	instance.Type = instanceType

	// 清空累计流量统计
	instance.TCPRX = 0
	instance.TCPTX = 0
	instance.UDPRX = 0
	instance.UDPTX = 0

	// 更新实例状态
	instance.Status = "stopped"
	m.instances.Store(id, instance)

	// 启动实例
	go m.startInstance(instance)

	// 保存实例状态
	go func() {
		time.Sleep(100 * time.Millisecond)
		m.saveState()
	}()
	writeJSON(w, http.StatusOK, instance)

	m.logger.Info("Instance URL updated: %v [%v]", instance.URL, instance.ID)
}

// regenerateAPIKey 重新生成API Key
func (m *Master) regenerateAPIKey(instance *Instance) {
	instance.URL = generateAPIKey()
	m.instances.Store(apiKeyID, instance)
	m.saveState()
	m.logger.Info("API Key regenerated: %v", instance.URL)
}

// processInstanceAction 处理实例操作
func (m *Master) processInstanceAction(instance *Instance, action string) {
	switch action {
	case "start":
		if instance.Status != "running" {
			go m.startInstance(instance)
		}
	case "stop":
		if instance.Status == "running" {
			go m.stopInstance(instance)
		}
	case "restart":
		if instance.Status == "running" {
			go func() {
				m.stopInstance(instance)
				time.Sleep(100 * time.Millisecond)
				m.startInstance(instance)
			}()
		} else {
			go m.startInstance(instance)
		}
	}
}

// handleDeleteInstance 处理删除实例请求
func (m *Master) handleDeleteInstance(w http.ResponseWriter, id string, instance *Instance) {
	// API Key实例不允许删除
	if id == apiKeyID {
		httpError(w, "Forbidden: API Key", http.StatusForbidden)
		return
	}

	if instance.Status == "running" {
		m.stopInstance(instance)
	}
	m.instances.Delete(id)
	// 删除实例后保存状态
	m.saveState()
	w.WriteHeader(http.StatusNoContent)

	// 发送删除事件
	m.sendSSEEvent("delete", instance)
}

// handleSSE 处理SSE连接请求
func (m *Master) handleSSE(w http.ResponseWriter, r *http.Request) {
	// 验证是否为GET请求
	if r.Method != http.MethodGet {
		httpError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 设置SSE相关响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建唯一的订阅者ID
	subscriberID := generateID()

	// 创建一个通道用于接收事件
	events := make(chan *InstanceEvent, 10)

	// 注册订阅者
	m.subscribers.Store(subscriberID, events)
	defer m.subscribers.Delete(subscriberID)

	// 发送初始重试间隔
	fmt.Fprintf(w, "retry: %d\n\n", sseRetryTime)

	// 获取当前所有实例并发送初始状态
	m.instances.Range(func(_, value any) bool {
		instance := value.(*Instance)
		event := &InstanceEvent{
			Type:     "initial",
			Time:     time.Now(),
			Instance: instance,
		}

		data, err := json.Marshal(event)
		if err == nil {
			fmt.Fprintf(w, "event: instance\ndata: %s\n\n", data)
			w.(http.Flusher).Flush()
		}
		return true
	})

	// 设置客户端连接超时
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// 客户端连接关闭标志
	connectionClosed := make(chan struct{})

	// 监听客户端连接是否关闭，但不关闭通道，留给Shutdown处理
	go func() {
		<-ctx.Done()
		close(connectionClosed)
		// 只从映射表中移除，但不关闭通道
		m.subscribers.Delete(subscriberID)
	}()

	// 持续发送事件到客户端
	for {
		select {
		case <-connectionClosed:
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			// 序列化事件数据
			data, err := json.Marshal(event)
			if err != nil {
				m.logger.Error("Event marshal error: %v", err)
				continue
			}

			// 发送事件
			fmt.Fprintf(w, "event: instance\ndata: %s\n\n", data)
			w.(http.Flusher).Flush()
		}
	}
}

// sendSSEEvent 发送SSE事件的通用函数
func (m *Master) sendSSEEvent(eventType string, instance *Instance, logs ...string) {
	event := &InstanceEvent{
		Type:     eventType,
		Time:     time.Now(),
		Instance: instance,
	}

	// 如果有日志内容，添加到事件中
	if len(logs) > 0 {
		event.Logs = logs[0]
	}

	// 非阻塞方式发送事件
	select {
	case m.notifyChannel <- event:
	default:
		// 通道已满或关闭，忽略
	}
}

// startEventDispatcher 启动事件分发器
func (m *Master) startEventDispatcher() {
	for event := range m.notifyChannel {
		// 向所有订阅者分发事件
		m.subscribers.Range(func(_, value any) bool {
			eventChan := value.(chan *InstanceEvent)
			// 非阻塞方式发送事件
			select {
			case eventChan <- event:
			default:
				// 不可用，忽略
			}
			return true
		})
	}
}

// findInstance 查找实例
func (m *Master) findInstance(id string) (*Instance, bool) {
	value, exists := m.instances.Load(id)
	if !exists {
		return nil, false
	}
	return value.(*Instance), true
}

// startInstance 启动实例
func (m *Master) startInstance(instance *Instance) {
	// 获取最新实例状态
	if value, exists := m.instances.Load(instance.ID); exists {
		instance = value.(*Instance)
		if instance.Status == "running" {
			return
		}
	}

	// 保存原始流量统计
	originalTCPRX := instance.TCPRX
	originalTCPTX := instance.TCPTX
	originalUDPRX := instance.UDPRX
	originalUDPTX := instance.UDPTX

	// 获取可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		m.logger.Error("Get path failed: %v [%v]", err, instance.ID)
		instance.Status = "error"
		m.instances.Store(instance.ID, instance)
		return
	}

	// 创建上下文和命令
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, execPath, instance.URL)
	instance.cancelFunc = cancel

	// 设置日志输出
	writer := NewInstanceLogWriter(instance.ID, instance, os.Stdout, m)
	cmd.Stdout, cmd.Stderr = writer, writer

	m.logger.Info("Instance starting: %v [%v]", instance.URL, instance.ID)

	// 启动实例
	if err := cmd.Start(); err != nil {
		m.logger.Error("Instance error: %v [%v]", err, instance.ID)
		instance.Status = "error"
		cancel()
	} else {
		instance.cmd = cmd
		instance.Status = "running"

		// 恢复原始流量统计
		instance.TCPRX = originalTCPRX
		instance.TCPTX = originalTCPTX
		instance.UDPRX = originalUDPRX
		instance.UDPTX = originalUDPTX

		go m.monitorInstance(instance, cmd)
	}

	m.instances.Store(instance.ID, instance)

	// 发送启动事件
	m.sendSSEEvent("update", instance)
}

// monitorInstance 监控实例状态
func (m *Master) monitorInstance(instance *Instance, cmd *exec.Cmd) {
	select {
	case <-instance.stopped:
		// 实例被显式停止
		return
	default:
		// 等待进程完成
		err := cmd.Wait()

		// 获取最新的实例状态
		if value, exists := m.instances.Load(instance.ID); exists {
			instance = value.(*Instance)

			// 仅在实例状态为running时才发送事件
			if instance.Status == "running" {
				if err != nil {
					m.logger.Error("Instance error: %v [%v]", err, instance.ID)
					instance.Status = "error"
				} else {
					instance.Status = "stopped"
				}
				m.instances.Store(instance.ID, instance)

				// 安全地发送停止事件，避免向已关闭的通道发送
				m.sendSSEEvent("update", instance)
			}
		}
	}
}

// stopInstance 停止实例
func (m *Master) stopInstance(instance *Instance) {
	// 如果已经是停止状态，不重复操作
	if instance.Status == "stopped" {
		return
	}

	// 如果没有命令或进程，直接设为已停止
	if instance.cmd == nil || instance.cmd.Process == nil {
		instance.Status = "stopped"
		m.instances.Store(instance.ID, instance)
		m.sendSSEEvent("update", instance)
		return
	}

	// 发送终止信号
	if instance.cmd.Process != nil {
		if runtime.GOOS == "windows" {
			instance.cmd.Process.Signal(os.Interrupt)
		} else {
			instance.cmd.Process.Signal(syscall.SIGTERM)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 关闭停止通道
	close(instance.stopped)

	// 取消执行或强制终止
	if instance.cancelFunc != nil {
		instance.cancelFunc()
	} else {
		err := instance.cmd.Process.Kill()
		if err != nil {
			m.logger.Error("Instance error: %v [%v]", err, instance.ID)
		}
	}

	m.logger.Info("Instance stopped [%v]", instance.ID)

	// 重置实例状态
	instance.Status = "stopped"
	instance.stopped = make(chan struct{})
	instance.cancelFunc = nil
	m.instances.Store(instance.ID, instance)

	// 保存状态变更
	m.saveState()

	// 发送停止事件
	m.sendSSEEvent("update", instance)
}

// enhanceURL 增强URL，添加日志级别和TLS配置
func (m *Master) enhanceURL(instanceURL string, instanceType string) string {
	parsedURL, err := url.Parse(instanceURL)
	if err != nil {
		m.logger.Error("Invalid URL format: %v", err)
		return instanceURL
	}

	query := parsedURL.Query()

	// 设置日志级别
	if m.logLevel != "" && query.Get("log") == "" {
		query.Set("log", m.logLevel)
	}

	// 为服务端实例设置TLS配置
	if instanceType == "server" && m.tlsCode != "0" {
		if query.Get("tls") == "" {
			query.Set("tls", m.tlsCode)
		}

		// 为TLS code-2设置证书和密钥
		if m.tlsCode == "2" {
			if m.crtPath != "" && query.Get("crt") == "" {
				query.Set("crt", m.crtPath)
			}
			if m.keyPath != "" && query.Get("key") == "" {
				query.Set("key", m.keyPath)
			}
		}
	}

	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

// generateID 生成随机ID
func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateAPIKey 生成API Key
func generateAPIKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// httpError 返回HTTP错误
func httpError(w http.ResponseWriter, message string, statusCode int) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// writeJSON 写入JSON响应
func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// generateOpenAPISpec 生成OpenAPI规范文档
func generateOpenAPISpec() string {
	return fmt.Sprintf(`{
  "openapi": "3.1.1",
  "info": {
    "title": "NodePass API",
    "description": "API for managing NodePass server and client instances",
    "version": "%s"
  },
  "servers": [{"url": "/{prefix}/v1", "variables": {"prefix": {"default": "api", "description": "API prefix path"}}}],
  "security": [{"ApiKeyAuth": []}],
  "paths": {
    "/instances": {
      "get": {
        "summary": "List all instances",
        "security": [{"ApiKeyAuth": []}],
        "responses": {
          "200": {"description": "Success", "content": {"application/json": {"schema": {"type": "array", "items": {"$ref": "#/components/schemas/Instance"}}}}},
          "401": {"description": "Unauthorized"},
          "405": {"description": "Method not allowed"}
        }
      },
      "post": {
        "summary": "Create a new instance",
        "security": [{"ApiKeyAuth": []}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreateInstanceRequest"}}}},
        "responses": {
          "201": {"description": "Created", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Instance"}}}},
          "400": {"description": "Invalid input"},
          "401": {"description": "Unauthorized"},
          "405": {"description": "Method not allowed"},
          "409": {"description": "Instance ID already exists"}
        }
      }
    },
    "/instances/{id}": {
      "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}],
      "get": {
        "summary": "Get instance details",
        "security": [{"ApiKeyAuth": []}],
        "responses": {
          "200": {"description": "Success", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Instance"}}}},
          "400": {"description": "Instance ID required"},
          "401": {"description": "Unauthorized"},
          "404": {"description": "Not found"},
          "405": {"description": "Method not allowed"}
        }
      },
      "patch": {
        "summary": "Update instance",
        "security": [{"ApiKeyAuth": []}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/UpdateInstanceRequest"}}}},
        "responses": {
          "200": {"description": "Success", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Instance"}}}},
          "400": {"description": "Instance ID required or invalid input"},
          "401": {"description": "Unauthorized"},
          "404": {"description": "Not found"},
          "405": {"description": "Method not allowed"}
        }
      },
      "put": {
        "summary": "Update instance URL",
        "security": [{"ApiKeyAuth": []}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/PutInstanceRequest"}}}},
        "responses": {
          "200": {"description": "Success", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Instance"}}}},
          "400": {"description": "Instance ID required or invalid input"},
          "401": {"description": "Unauthorized"},
          "403": {"description": "Forbidden"},
          "404": {"description": "Not found"},
          "405": {"description": "Method not allowed"},
          "409": {"description": "Instance URL conflict"}
        }
      },
      "delete": {
        "summary": "Delete instance",
        "security": [{"ApiKeyAuth": []}],
        "responses": {
          "204": {"description": "Deleted"},
          "400": {"description": "Instance ID required"},
          "401": {"description": "Unauthorized"},
          "403": {"description": "Forbidden"},
          "404": {"description": "Not found"},
          "405": {"description": "Method not allowed"}
        }
      }
    },
    "/events": {
      "get": {
        "summary": "Subscribe to instance events",
		"security": [{"ApiKeyAuth": []}],
        "responses": {
          "200": {"description": "Success", "content": {"text/event-stream": {}}},
		  "401": {"description": "Unauthorized"},
          "405": {"description": "Method not allowed"}
        }
      }
    },
    "/info": {
      "get": {
        "summary": "Get master information",
        "security": [{"ApiKeyAuth": []}],
        "responses": {
          "200": {"description": "Success", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/MasterInfo"}}}},
          "401": {"description": "Unauthorized"},
          "405": {"description": "Method not allowed"}
        }
      }
    },
    "/openapi.json": {
      "get": {
        "summary": "Get OpenAPI specification",
        "responses": {
          "200": {"description": "Success", "content": {"application/json": {}}}
        }
      }
    },
    "/docs": {
      "get": {
        "summary": "Get Swagger UI",
        "responses": {
          "200": {"description": "Success", "content": {"text/html": {}}}
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "ApiKeyAuth": {
        "type": "apiKey",
        "in": "header",
        "name": "X-API-Key",
        "description": "API Key for authentication"
      }
    },
    "schemas": {
      "Instance": {
        "type": "object",
        "properties": {
          "id": {"type": "string", "description": "Unique identifier"},
          "alias": {"type": "string", "description": "Instance alias"},
          "type": {"type": "string", "enum": ["client", "server"], "description": "Type of instance"},
          "status": {"type": "string", "enum": ["running", "stopped", "error"], "description": "Instance status"},
          "url": {"type": "string", "description": "Command string or API Key"},
          "restart": {"type": "boolean", "description": "Restart policy"},
          "tcprx": {"type": "integer", "description": "TCP received bytes"},
          "tcptx": {"type": "integer", "description": "TCP transmitted bytes"},
          "udprx": {"type": "integer", "description": "UDP received bytes"},
          "udptx": {"type": "integer", "description": "UDP transmitted bytes"}
        }
      },
      "CreateInstanceRequest": {
        "type": "object",
        "required": ["url"],
        "properties": {"url": {"type": "string", "description": "Command string(scheme://host:port/host:port)"}}
      },
      "UpdateInstanceRequest": {
        "type": "object",
        "properties": {
          "alias": {"type": "string", "description": "Instance alias"},
          "action": {"type": "string", "enum": ["start", "stop", "restart"], "description": "Action for the instance"},
          "restart": {"type": "boolean", "description": "Instance restart policy"}
        }
      },
      "PutInstanceRequest": {
        "type": "object",
        "required": ["url"],
        "properties": {"url": {"type": "string", "description": "New command string(scheme://host:port/host:port)"}}
      },
      "MasterInfo": {
        "type": "object",
        "properties": {
          "os": {"type": "string", "description": "Operating system"},
          "arch": {"type": "string", "description": "System architecture"},
          "ver": {"type": "string", "description": "NodePass version"},
		  "name": {"type": "string", "description": "Hostname"},
          "uptime": {"type": "integer", "format": "int64", "description": "Uptime in seconds"},
          "log": {"type": "string", "description": "Log level"},
          "tls": {"type": "string", "description": "TLS code"},
          "crt": {"type": "string", "description": "Certificate path"},
          "key": {"type": "string", "description": "Private key path"}
        }
      }
    }
  }
}`, openAPIVersion)
}
