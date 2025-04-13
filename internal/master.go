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

const openAPIVersion = "v1"

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

type Master struct {
	Common
	prefix    string
	instances sync.Map
	server    *http.Server
	logLevel  string
	tlsConfig *tls.Config
	masterURL *url.URL
}

type Instance struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Status     string             `json:"status"`
	URL        string             `json:"url"`
	TCPRX      uint64             `json:"tcprx"`
	TCPTX      uint64             `json:"tcptx"`
	UDPRX      uint64             `json:"udprx"`
	UDPTX      uint64             `json:"udptx"`
	cmd        *exec.Cmd          `json:"-"`
	stopped    chan struct{}      `json:"-"`
	cancelFunc context.CancelFunc `json:"-"`
}

type InstanceLogWriter struct {
	instanceID string
	instance   *Instance
	target     io.Writer
	master     *Master
	statRegex  *regexp.Regexp
}

func NewInstanceLogWriter(instanceID string, instance *Instance, target io.Writer, master *Master) *InstanceLogWriter {
	return &InstanceLogWriter{
		instanceID: instanceID,
		instance:   instance,
		target:     target,
		master:     master,
		statRegex:  regexp.MustCompile(`TRAFFIC_STATS\|TCP_RX=(\d+)\|TCP_TX=(\d+)\|UDP_RX=(\d+)\|UDP_TX=(\d+)`),
	}
}

func (w *InstanceLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
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
		fmt.Fprintf(w.target, "%s [%s]\n", line, w.instanceID)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(w.target, "%s [%s]", s, w.instanceID)
	}
	return len(p), nil
}

func setCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func NewMaster(parsedURL *url.URL, tlsCode string, tlsConfig *tls.Config, logger *log.Logger) *Master {
	host, err := net.ResolveTCPAddr("tcp", parsedURL.Host)
	if err != nil {
		logger.Error("Resolve failed: %v", err)
		return nil
	}
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

func (m *Master) Manage() {
	m.logger.Info("Master started: %v%v", m.tunnelAddr, m.prefix)
	mux := http.NewServeMux()
	endpoints := map[string]http.HandlerFunc{
		fmt.Sprintf("%s/instances", m.prefix):    m.handleInstances,
		fmt.Sprintf("%s/instances/", m.prefix):   m.handleInstanceDetail,
		fmt.Sprintf("%s/openapi.json", m.prefix): m.handleOpenAPISpec,
		fmt.Sprintf("%s/docs", m.prefix):         m.handleSwaggerUI,
	}
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
	m.server = &http.Server{
		Addr:      m.tunnelAddr.String(),
		ErrorLog:  m.logger.StdLogger(),
		Handler:   mux,
		TLSConfig: m.tlsConfig,
	}
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := m.Shutdown(shutdownCtx); err != nil {
		m.logger.Error("Master shutdown error: %v", err)
	} else {
		m.logger.Info("Master shutdown complete")
	}
}

func (m *Master) Shutdown(ctx context.Context) error {
	return m.shutdown(ctx, func() {
		var wg sync.WaitGroup
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
		if err := m.server.Shutdown(ctx); err != nil {
			m.logger.Error("ApiSvr shutdown error: %v", err)
		}
	})
}

func (m *Master) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(generateOpenAPISpec()))
}

func (m *Master) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, swaggerUIHTML, generateOpenAPISpec())
}

func (m *Master) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		instances := []*Instance{}
		m.instances.Range(func(_, value any) bool {
			instances = append(instances, value.(*Instance))
			return true
		})
		writeJSON(w, http.StatusOK, instances)
	case http.MethodPost:
		var reqData struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil || reqData.URL == "" {
			httpError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		parsedURL, err := url.Parse(reqData.URL)
		if err != nil {
			httpError(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		instanceType := parsedURL.Scheme
		if instanceType != "client" && instanceType != "server" {
			httpError(w, "URL scheme must be 'client' or 'server'", http.StatusBadRequest)
			return
		}
		id := generateID()
		if _, exists := m.instances.Load(id); exists {
			httpError(w, "Instance ID already exists", http.StatusConflict)
			return
		}
		instance := &Instance{
			ID:     id,
			Type:   instanceType,
			URL:    m.enhanceURL(reqData.URL, instanceType),
			Status: "stopped", stopped: make(chan struct{}),
		}
		m.instances.Store(id, instance)
		go m.startInstance(instance)
		writeJSON(w, http.StatusCreated, instance)
	default:
		httpError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *Master) handleInstanceDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, fmt.Sprintf("%s/instances/", m.prefix))
	if id == "" || id == "/" {
		httpError(w, "Instance ID is required", http.StatusBadRequest)
		return
	}
	instance, ok := m.findInstance(id)
	if !ok {
		httpError(w, "Instance not found", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, instance)
	case http.MethodPut:
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
		if instance.Status == "running" {
			m.stopInstance(instance)
		}
		m.instances.Delete(id)
		w.WriteHeader(http.StatusNoContent)
	default:
		httpError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *Master) findInstance(id string) (*Instance, bool) {
	value, exists := m.instances.Load(id)
	if !exists {
		return nil, false
	}
	return value.(*Instance), true
}

func (m *Master) startInstance(instance *Instance) {
	if value, exists := m.instances.Load(instance.ID); exists {
		instance = value.(*Instance)
		if instance.Status == "running" {
			return
		}
	}
	execPath, err := os.Executable()
	if err != nil {
		m.logger.Error("Get path failed: %v [%v]", err, instance.ID)
		instance.Status = "error"
		m.instances.Store(instance.ID, instance)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, execPath, instance.URL)
	instance.cancelFunc = cancel
	writer := NewInstanceLogWriter(instance.ID, instance, os.Stdout, m)
	cmd.Stdout, cmd.Stderr = writer, writer
	m.logger.Info("Instance queued: %v [%s]", instance.URL, instance.ID)
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

func (m *Master) stopInstance(instance *Instance) {
	if instance.cmd == nil || instance.cmd.Process == nil {
		instance.Status = "stopped"
		m.instances.Store(instance.ID, instance)
		return
	}
	if instance.cmd.Process != nil {
		if runtime.GOOS == "windows" {
			instance.cmd.Process.Signal(os.Interrupt)
		} else {
			instance.cmd.Process.Signal(syscall.SIGTERM)
		}
		time.Sleep(100 * time.Millisecond)
	}
	close(instance.stopped)
	if instance.cancelFunc != nil {
		instance.cancelFunc()
	} else {
		err := instance.cmd.Process.Kill()
		if err != nil {
			m.logger.Error("Instance error: %v [%v]", err, instance.ID)
		}
	}
	m.logger.Info("Instance stopped [%v]", instance.ID)
	instance.Status = "stopped"
	instance.stopped = make(chan struct{})
	instance.cancelFunc = nil
	m.instances.Store(instance.ID, instance)
}

func (m *Master) enhanceURL(instanceURL string, instanceType string) string {
	parsedURL, err := url.Parse(instanceURL)
	if err != nil {
		m.logger.Error("Invalid URL format: %v", err)
		return instanceURL
	}
	query := parsedURL.Query()
	if m.logLevel != "" && query.Get("log") == "" {
		query.Set("log", m.logLevel)
	}
	if instanceType == "server" && m.tlsCode != "0" {
		if query.Get("tls") == "" {
			query.Set("tls", m.tlsCode)
		}
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

func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func httpError(w http.ResponseWriter, message string, statusCode int) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

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
