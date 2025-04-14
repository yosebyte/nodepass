// 内部包，实现主控模式功能
package internal

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/yosebyte/x/log"
)

// API版本
const openAPIVersion = "v1"

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
	Common                 // 继承通用功能
	prefix    string       // API前缀
	instances sync.Map     // 实例映射表
	server    *http.Server // HTTP服务器
	logLevel  string       // 日志级别
	tlsConfig *tls.Config  // TLS配置
	masterURL *url.URL     // 主控URL
}

// Instance 实例信息
type Instance struct {
	ID         string             `json:"id"`     // 实例ID
	Type       string             `json:"type"`   // 实例类型（client或server）
	Status     string             `json:"status"` // 实例状态
	URL        string             `json:"url"`    // 实例URL
	TCPRX      uint64             `json:"tcprx"`  // TCP接收字节数
	TCPTX      uint64             `json:"tcptx"`  // TCP发送字节数
	UDPRX      uint64             `json:"udprx"`  // UDP接收字节数
	UDPTX      uint64             `json:"udptx"`  // UDP发送字节数
	cmd        *exec.Cmd          `json:"-"`      // 命令对象（不序列化）
	stopped    chan struct{}      `json:"-"`      // 停止信号通道（不序列化）
	cancelFunc context.CancelFunc `json:"-"`      // 取消函数（不序列化）
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
					*stat = v
				}
			}
			w.master.instances.Store(w.instanceID, w.instance)
			continue
		}
		// 输出常规日志
		fmt.Fprintf(w.target, "%s [%s]\n", line, w.instanceID)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(w.target, "%s [%s]", s, w.instanceID)
	}
	return len(p), nil
}

// 设置跨域响应头
func setCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// NewMaster 创建新的主控实例
func NewMaster(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *log.Logger) *Master {
	// 解析主机地址
	host, err := net.ResolveTCPAddr("tcp", parsedURL.Host)
	if err != nil {
		logger.Error("Resolve failed: %v", err)
		return nil
	}

	// 设置API前缀
	prefix := parsedURL.Path
	if prefix == "" || prefix == "/" {
		prefix = "/api"
	} else {
		prefix = strings.TrimRight(prefix, "/")
	}

	master := &Master{
		Common: Common{
			tlsCode: tlsCode,
			logger:  logger,
		},
		prefix:    fmt.Sprintf("%s/%s", prefix, openAPIVersion),
		logLevel:  parsedURL.Query().Get("log"),
		tlsConfig: tlsConfig,
		masterURL: parsedURL,
	}
	master.tunnelAddr = host
	return master
}

// Manage 管理主控生命周期
func (m *Master) Manage() {
	m.logger.Info("Master started: %v%v", m.tunnelAddr, m.prefix)

	// 设置HTTP路由
	mux := http.NewServeMux()
	endpoints := map[string]http.HandlerFunc{
		fmt.Sprintf("%s/instances", m.prefix):    m.handleInstances,
		fmt.Sprintf("%s/instances/", m.prefix):   m.handleInstanceDetail,
		fmt.Sprintf("%s/openapi.json", m.prefix): m.handleOpenAPISpec,
		fmt.Sprintf("%s/docs", m.prefix):         m.handleSwaggerUI,
	}

	// 注册路由处理器
	for path, handler := range endpoints {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			setCorsHeaders(w)
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			handler(w, r)
		})
	}

	// 创建HTTP服务器
	m.server = &http.Server{
		Addr:      m.tunnelAddr.String(),
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
		var wg sync.WaitGroup

		// 停止所有运行中的实例
		m.instances.Range(func(key, value any) bool {
			instance := value.(*Instance)
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

		// 关闭HTTP服务器
		if err := m.server.Shutdown(ctx); err != nil {
			m.logger.Error("ApiSvr shutdown error: %v", err)
		}
	})
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
			httpError(w, "URL scheme must be 'client' or 'server'", http.StatusBadRequest)
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
			stopped: make(chan struct{}),
		}
		m.instances.Store(id, instance)

		// 启动实例
		go m.startInstance(instance)
		writeJSON(w, http.StatusCreated, instance)

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
		// 获取实例信息
		writeJSON(w, http.StatusOK, instance)

	case http.MethodPut:
		// 更新实例状态
		var reqData struct {
			Action string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqData); err == nil {
			switch reqData.Action {
			case "start":
				if instance.Status != "running" {
					go m.startInstance(instance)
				}
			case "stop":
				if instance.Status == "running" {
					m.stopInstance(instance)
				}
			case "restart":
				if instance.Status == "running" {
					m.stopInstance(instance)
				}
				go m.startInstance(instance)
			}
		}
		writeJSON(w, http.StatusOK, instance)

	case http.MethodDelete:
		// 删除实例
		if instance.Status == "running" {
			m.stopInstance(instance)
		}
		m.instances.Delete(id)
		w.WriteHeader(http.StatusNoContent)

	default:
		httpError(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	m.logger.Info("Instance queued: %v [%s]", instance.URL, instance.ID)

	// 启动实例
	if err := cmd.Start(); err != nil {
		m.logger.Error("Instance error: %v [%v]", err, instance.ID)
		instance.Status = "error"
		cancel()
	} else {
		instance.cmd = cmd
		instance.Status = "running"
		go m.monitorInstance(instance, cmd)
	}

	m.instances.Store(instance.ID, instance)
}

// monitorInstance 监控实例状态
func (m *Master) monitorInstance(instance *Instance, cmd *exec.Cmd) {
	select {
	case <-instance.stopped:
		return
	default:
		if err := cmd.Wait(); err != nil && instance.Status != "stopped" {
			if value, exists := m.instances.Load(instance.ID); exists {
				instance = value.(*Instance)
				if instance.Status != "stopped" {
					m.logger.Error("Instance error: %v [%v]", err, instance.ID)
					instance.Status = "error"
					m.instances.Store(instance.ID, instance)
				}
			}
		} else if value, exists := m.instances.Load(instance.ID); exists {
			instance = value.(*Instance)
			if instance.Status != "stopped" {
				instance.Status = "stopped"
				m.instances.Store(instance.ID, instance)
			}
		}
	}
}

// stopInstance 停止实例
func (m *Master) stopInstance(instance *Instance) {
	// 如果没有命令或进程，直接设为已停止
	if instance.cmd == nil || instance.cmd.Process == nil {
		instance.Status = "stopped"
		m.instances.Store(instance.ID, instance)
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

	// 为服务器实例设置TLS配置
	if instanceType == "server" && m.tlsCode != "0" {
		if query.Get("tls") == "" {
			query.Set("tls", m.tlsCode)
		}

		// 为TLS code-2设置证书和密钥
		if m.tlsCode == "2" {
			masterQuery := m.masterURL.Query()
			for _, param := range []string{"crt", "key"} {
				if val := masterQuery.Get(param); val != "" && query.Get(param) == "" {
					query.Set(param, val)
				}
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
  "openapi": "3.0.0",
  "info": {
    "title": "NodePass API",
    "description": "API for managing NodePass server and client instances",
    "version": "%s"
  },
  "servers": [{"url": "/{prefix}/v1", "variables": {"prefix": {"default": "api", "description": "API prefix path"}}}],
  "paths": {
    "/instances": {
      "get": {
        "summary": "List all instances",
        "responses": {"200": {"description": "Success", "content": {"application/json": {"schema": {"type": "array", "items": {"$ref": "#/components/schemas/Instance"}}}}}}
      },
      "post": {
        "summary": "Create a new instance",
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreateInstanceRequest"}}}},
        "responses": {
          "201": {"description": "Created", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Instance"}}}},
          "400": {"description": "Invalid input"},
		  "404": {"description": "Not found"}
        }
      }
    },
    "/instances/{id}": {
      "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}],
      "get": {
        "summary": "Get instance details",
        "responses": {
          "200": {"description": "Success", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Instance"}}}},
          "404": {"description": "Not found"}
        }
      },
      "put": {
        "summary": "Update instance",
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/UpdateInstanceRequest"}}}},
        "responses": {
          "200": {"description": "Success", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Instance"}}}},
          "404": {"description": "Not found"}
        }
      },
      "delete": {
        "summary": "Delete instance",
        "responses": {"204": {"description": "Deleted"}, "404": {"description": "Not found"}}
      }
    }
  },
  "components": {
    "schemas": {
      "Instance": {
        "type": "object",
        "properties": {
          "id": {"type": "string", "description": "Unique identifier"},
          "type": {"type": "string", "enum": ["client", "server"], "description": "Type of instance"},
          "status": {"type": "string", "enum": ["running", "stopped", "error"], "description": "Instance status"},
          "url": {"type": "string", "description": "Command string"},
          "tcprx": {"type": "integer", "format": "int64", "description": "TCP bytes received"},
          "tcptx": {"type": "integer", "format": "int64", "description": "TCP bytes sent"},
          "udprx": {"type": "integer", "format": "int64", "description": "UDP bytes received"},
          "udptx": {"type": "integer", "format": "int64", "description": "UDP bytes sent"}
        }
      },
      "CreateInstanceRequest": {
        "type": "object",
        "required": ["url"],
        "properties": {"url": {"type": "string", "description": "Command string(scheme://host:port/host:port)"}}
      },
      "UpdateInstanceRequest": {
        "type": "object",
        "properties": {"action": {"type": "string", "enum": ["start", "stop", "restart"], "description": "Action"}}
      }
    }
  }
}`, openAPIVersion)
}
