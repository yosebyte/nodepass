# NodePass 改进记录

## 概述

本次更新为NodePass项目添加了全面的安全增强措施，主要解决两类安全威胁：

1. **中间人攻击防护**：通过证书固定、证书验证和安全握手协议，确保客户端只与合法服务器建立连接
2. **重放攻击防护**：通过时间戳、nonce和消息认证码，防止攻击者重放捕获的通信数据

同时保留了之前添加的功能：
- TLS1.3加密支持
- QUIC协议支持
- WebSocket全双工连接

## 文件变更

### 新增文件

- `internal/security/anti_replay.go` - 防重放攻击机制实现
- `internal/security/handshake.go` - 安全握手协议实现
- `internal/security/connection_verifier.go` - 连接验证器实现
- `internal/security_manager.go` - 安全管理器，整合所有安全组件
- `internal/test/security_test.go` - 安全特性测试用例
- `SECURITY.md` - 安全特性详细文档

### 修改文件

- `internal/tls/config.go` - 增强TLS配置，添加证书固定机制
- `internal/client.go` - 集成安全握手和消息验证
- `internal/server.go` - 集成安全握手和消息验证
- `internal/common.go` - 添加安全管理器字段

## 安全特性详解

### 1. 证书固定机制

- 实现了证书指纹计算和验证功能
- 客户端可以预先存储受信任服务器的证书指纹
- 连接时验证服务器证书指纹，防止中间人攻击

### 2. 安全握手协议

- 实现了双向认证的握手流程
- 交换加密参数和支持的协议信息
- 验证服务器身份和证书有效性

### 3. 防重放攻击机制

- 使用NonceManager跟踪已使用的nonce
- 每条消息包含唯一nonce和时间戳
- 拒绝处理重复nonce或过期时间戳的消息

### 4. 消息完整性验证

- 使用HMAC-SHA256计算消息认证码
- 验证消息完整性，防止消息被篡改
- 确保消息来源可信

### 5. 连接验证机制

- 实现连接令牌生成和验证
- 跟踪已验证的连接
- 防止会话劫持攻击

## 使用方法

安全特性默认启用，无需额外配置。当使用TLS加密模式（tls=1或tls=2）时，系统将自动应用所有安全增强措施。

```bash
# 使用自签名证书和全部安全特性
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# 使用自定义证书和全部安全特性
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

详细的安全特性说明和配置指南请参考 `SECURITY.md` 文档。
