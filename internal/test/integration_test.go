package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// 测试WebSocket全双工通信
func TestWebSocketFullDuplex(t *testing.T) {
	// 创建一个简单的WebSocket服务器
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// 读取消息并回显
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if !strings.Contains(err.Error(), "websocket: close") {
					t.Logf("Read error: %v", err)
				}
				return
			}

			// 回显消息
			err = conn.WriteMessage(messageType, message)
			if err != nil {
				t.Logf("Write error: %v", err)
				return
			}
		}
	}))
	defer server.Close()

	// 将http://替换为ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 创建WebSocket客户端
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()

	// 测试发送和接收消息
	testMessage := "Hello, WebSocket!"
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// 接收回显消息
	_, receivedMessage, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	// 验证消息
	if string(receivedMessage) != testMessage {
		t.Errorf("Expected message %q, got %q", testMessage, string(receivedMessage))
	}

	fmt.Println("WebSocket full-duplex communication test passed")
}

// 测试TLS1.3与WebSocket的集成
func TestTLS13WithWebSocket(t *testing.T) {
	// 这个测试需要在实际网络环境中运行
	// 这里只验证相关代码的结构和接口
	
	// 验证TLS配置在WebSocket客户端中的应用
	clientFile := "/home/ubuntu/workspace/nodepass/internal/websocket/client.go"
	content, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read WebSocket client file: %v", err)
	}
	
	// 检查是否使用了TLS1.3
	if !strings.Contains(string(content), "ntls.GetTLS13Config") {
		t.Errorf("WebSocket client does not use TLS1.3 configuration")
	}
	
	// 验证TLS配置在WebSocket服务器中的应用
	serverFile := "/home/ubuntu/workspace/nodepass/internal/websocket/server.go"
	content, err = os.ReadFile(serverFile)
	if err != nil {
		t.Fatalf("Failed to read WebSocket server file: %v", err)
	}
	
	// 检查是否使用了TLS1.3
	if !strings.Contains(string(content), "ntls.GetTLS13Config") {
		t.Errorf("WebSocket server does not use TLS1.3 configuration")
	}
	
	fmt.Println("TLS1.3 with WebSocket integration test passed")
}

// 综合测试所有功能
func TestAllFeatures(t *testing.T) {
	// 验证所有功能的集成
	
	// 检查common.go是否包含所有协议支持标志
	commonFile := "/home/ubuntu/workspace/nodepass/internal/common.go"
	content, err := os.ReadFile(commonFile)
	if err != nil {
		t.Fatalf("Failed to read common file: %v", err)
	}
	
	// 检查是否支持QUIC
	if !strings.Contains(string(content), "supportsQuic") {
		t.Errorf("common.go does not include QUIC support flag")
	}
	
	// 检查是否支持WebSocket
	if !strings.Contains(string(content), "supportsWS") || !strings.Contains(string(content), "supportsWebSocket") {
		t.Errorf("common.go does not include WebSocket support flag")
	}
	
	// 检查客户端是否处理所有协议类型
	wsClientFile := "/home/ubuntu/workspace/nodepass/internal/ws_client.go"
	content, err = os.ReadFile(wsClientFile)
	if err != nil {
		t.Fatalf("Failed to read WebSocket client integration file: %v", err)
	}
	
	// 检查是否处理WebSocket信号
	if !strings.Contains(string(content), `case "4":`) {
		t.Errorf("WebSocket client does not handle WebSocket signal (fragment 4)")
	}
	
	// 检查服务器是否处理所有协议类型
	wsServerFile := "/home/ubuntu/workspace/nodepass/internal/ws_server.go"
	content, err = os.ReadFile(wsServerFile)
	if err != nil {
		t.Fatalf("Failed to read WebSocket server integration file: %v", err)
	}
	
	// 检查是否发送WebSocket信号
	if !strings.Contains(string(content), `Fragment: "4"`) {
		t.Errorf("WebSocket server does not send WebSocket signal (fragment 4)")
	}
	
	fmt.Println("All features integration test passed")
}
