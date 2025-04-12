# NodePass 改进记录

## 概述

本次改进为NodePass项目添加了三个主要功能：

1. **TLS1.3加密支持**：为所有加密连接强制使用TLS1.3，提供最高级别的安全性
2. **QUIC协议支持**：添加了基于UDP的QUIC协议，提供更低延迟和更好的多路复用能力
3. **WebSocket全双工连接**：实现了基于HTTP的WebSocket连接，特别适合在有防火墙限制的环境中使用

## 文件变更

### 新增文件

- `internal/tls/config.go` - TLS1.3配置辅助函数
- `internal/quic/client.go` - QUIC客户端实现
- `internal/quic/server.go` - QUIC服务器实现
- `internal/quic/pool.go` - QUIC连接池实现
- `internal/quic_client.go` - QUIC客户端集成
- `internal/quic_server.go` - QUIC服务器集成
- `internal/websocket/client.go` - WebSocket客户端实现
- `internal/websocket/server.go` - WebSocket服务器实现
- `internal/websocket/pool.go` - WebSocket连接池实现
- `internal/ws_client.go` - WebSocket客户端集成
- `internal/ws_server.go` - WebSocket服务器集成
- `internal/test/tls_quic_ws_test.go` - TLS、QUIC和WebSocket测试
- `internal/test/integration_test.go` - 集成测试

### 修改文件

- `cmd/nodepass/core.go` - 添加TLS1.3、QUIC和WebSocket支持
- `internal/common.go` - 添加协议支持标志和通用功能
- `go.mod` - 更新依赖，添加QUIC和WebSocket相关包
- `README.md` - 更新英文文档，添加新功能说明
- `README_zh.md` - 更新中文文档，添加新功能说明

## 技术细节

### TLS1.3加密

- 所有加密连接均使用TLS1.3，不再支持较低版本的TLS
- 限制了密码套件只使用TLS1.3支持的安全套件
- 提供了专用的TLS配置辅助函数，确保所有TLS连接强制使用TLS1.3

### QUIC协议

- 添加了基于UDP的QUIC协议支持，使用quic-go库
- 实现了完整的QUIC客户端、服务器端和连接池
- 集成到现有架构中，确保与TLS1.3加密兼容
- 添加了协议检测机制，使客户端能够识别服务器是否支持QUIC

### WebSocket全双工连接

- 添加了基于HTTP的WebSocket连接支持，使用gorilla/websocket库
- 实现了完整的WebSocket客户端、服务器端和连接池
- 提供了类似net.Conn的接口以便与现有代码集成
- 支持全双工通信，适用于防火墙限制环境

## 使用方法

### TLS1.3加密

TLS1.3加密已自动启用，无需额外配置。当使用TLS加密模式（tls=1或tls=2）时，系统将自动使用TLS1.3。

```bash
# 使用自签名证书和TLS1.3
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# 使用自定义证书和TLS1.3
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### QUIC协议

服务器会自动启用QUIC协议支持，客户端会自动检测并使用QUIC协议（如果可用）。

```bash
# 服务器端（自动启用QUIC支持）
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# 客户端（自动检测并使用QUIC）
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### WebSocket连接

WebSocket连接特别适合需要穿越防火墙的场景，因为它基于HTTP协议，大多数防火墙允许HTTP流量。

```bash
# 服务器端（自动启用WebSocket支持）
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# 客户端（自动检测并使用WebSocket）
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

## 性能考虑

- QUIC协议在不稳定网络环境中表现更好，特别适合移动网络
- WebSocket连接在穿越防火墙时更可靠，但可能比直接TCP连接有更高的延迟
- TLS1.3加密提供了更好的安全性，同时比TLS1.2有更低的握手延迟

## 兼容性

所有新功能都保持了与现有代码的兼容性，不会破坏现有的使用方式。客户端会自动适应服务器支持的协议类型，无需额外配置。
