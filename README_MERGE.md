# NodePass 合并报告

## 合并概述

本次合并将分支项目 https://github.com/HappyLeslieAlexander/nodepass 的改动合并到最新版本 https://github.com/yosebyte/nodepass 中，同时进行了代码精简和安全性增强。

主要改进包括：

1. **多协议支持**：增加了QUIC和WebSocket协议支持
2. **安全性增强**：添加了安全管理器、防重放攻击和连接验证功能
3. **代码优化**：统一了命名规范，优化了代码结构
4. **文档更新**：保留了CHANGELOG和SECURITY.md文件

## 合并内容详情

### 1. 目录结构变化

原始仓库的基本结构得到保留，同时增加了以下目录和文件：

```
internal/
  ├── quic/         # QUIC协议支持
  ├── websocket/    # WebSocket协议支持
  ├── security/     # 安全功能
  ├── tls/          # TLS配置
  ├── test/         # 测试文件
  ├── quic_client.go
  ├── quic_server.go
  ├── ws_client.go
  ├── ws_server.go
  └── security_manager.go
```

### 2. 功能增强

#### 2.1 多协议支持

- **QUIC协议**：增加了基于QUIC协议的通信支持，提供更低延迟和更可靠的连接
- **WebSocket协议**：增加了WebSocket协议支持，便于在Web环境中使用

#### 2.2 安全性增强

- **安全管理器**：实现了SecurityManager，统一管理安全相关功能
- **防重放攻击**：添加了NonceManager，防止重放攻击
- **连接验证**：实现了ConnectionVerifier，验证连接的合法性
- **TLS 1.3**：强制使用TLS 1.3，提高安全性

### 3. 代码优化

- **命名规范统一**：将所有结构体名称统一为大写开头（如Common、Server、Client等）
- **错误处理优化**：增强了错误处理和日志记录
- **代码结构优化**：改进了代码组织，增加了模块化设计

### 4. 依赖更新

- 保留了原始仓库的Go 1.24.1版本
- 合并了分支仓库的额外依赖：
  - github.com/gorilla/websocket v1.5.1
  - github.com/quic-go/quic-go v0.40.1

## 使用说明

### 环境要求

- Go 1.24.1或更高版本
- 支持的操作系统：Linux、macOS、Windows

### 编译方法

```bash
# 克隆仓库
git clone <repository-url>
cd nodepass

# 编译
go build -o nodepass ./cmd/nodepass
```

### 基本用法

NodePass现在支持TCP、UDP、QUIC和WebSocket四种协议：

```
# 服务端模式
./nodepass server://<tunnel-address>/<target-address>?log=info

# 客户端模式
./nodepass client://<tunnel-address>/<target-address>?log=info
```

### 协议选择

NodePass会自动协商使用最适合的协议。服务端会告知客户端其支持的协议，客户端会根据服务端的能力选择合适的协议。

### 安全配置

可以通过以下参数配置TLS安全性：

```
# 不使用TLS
?tls=0

# 使用内存中生成的证书
?tls=1

# 使用自定义证书
?tls=2&crt=<path/to/cert>&key=<path/to/key>
```

## 测试方法

由于环境限制，无法直接运行测试。请在具有Go环境的机器上执行以下测试：

```bash
# 单元测试
go test ./...

# 集成测试
go test ./internal/test -v
```

## 已知问题

- 在某些网络环境下，QUIC协议可能被防火墙阻止
- WebSocket在某些代理服务器后可能需要额外配置

## 后续改进方向

1. 增加更多协议支持（如HTTP/3）
2. 增强安全性（如添加双向认证）
3. 优化性能（如改进连接池管理）
4. 增加监控和统计功能
