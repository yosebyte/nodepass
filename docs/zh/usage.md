# 使用说明

NodePass创建一个带有未加密TCP控制通道的隧道，并为数据交换提供可配置的TLS加密选项。本指南涵盖三种操作模式并说明如何有效地使用每种模式。

## 命令行语法

NodePass命令的一般语法是：

```bash
nodepass "<core>://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>&mode=<run_mode>&min=<min_pool>&max=<max_pool>&read=<timeout>"
```

其中：
- `<core>`：指定操作模式（`server`、`client`或`master`）
- `<tunnel_addr>`：控制通道通信的隧道端点地址
- `<target_addr>`：业务数据的目标地址，支持双向模式（或在master模式下的API前缀）

### 查询参数说明

通用查询参数：
- `log=<level>`：日志详细级别（`none`、`debug`、`info`、`warn`、`error`或`event`）
- `mode=<run_mode>`：运行模式控制（`0`、`1`或`2`）- 控制操作行为
- `min=<min_pool>`：最小连接池容量（默认：64，仅适用于client模式）
- `max=<max_pool>`：最大连接池容量（默认：1024，仅适用于client模式）
- `read=<timeout>`：数据读取超时时间（默认：300s，支持时间单位如5s、30s、5m等）

TLS相关参数（仅适用于server/master模式）：
- `tls=<mode>`：数据通道的TLS安全级别（`0`、`1`或`2`）
- `crt=<cert_file>`：证书文件路径（当`tls=2`时）
- `key=<key_file>`：私钥文件路径（当`tls=2`时）

## 运行模式

NodePass提供三种互补的运行模式，以适应各种部署场景。

### 服务端模式

服务端模式建立隧道控制通道，并支持双向数据流转发。

```bash
nodepass "server://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>&mode=<run_mode>&read=<timeout>"
```

#### 参数

- `tunnel_addr`：TCP隧道端点地址（控制通道），客户端将连接到此处(例如, 10.1.0.1:10101)
- `target_addr`：业务数据的目标地址，支持双向数据流模式(例如, 10.1.0.1:8080)
- `log`：日志级别(debug, info, warn, error, event)
- `mode`：数据流方向的运行模式控制
  - `0`：自动检测（默认）- 首先尝试本地绑定，如果不可用则回退
  - `1`：强制反向模式 - 服务器本地绑定目标地址并接收流量
  - `2`：强制正向模式 - 服务器连接到远程目标地址
- `tls`：目标数据通道的TLS加密模式 (0, 1, 2)
  - `0`：无TLS加密（明文TCP/UDP）
  - `1`：自签名证书（自动生成）
  - `2`：自定义证书（需要`crt`和`key`参数）
- `crt`：证书文件路径（当`tls=2`时必需）
- `key`：私钥文件路径（当`tls=2`时必需）

#### 服务端模式工作原理

服务端模式通过`mode`参数支持自动模式检测或强制模式选择：

**模式0：自动检测**（默认）
- 首先尝试本地绑定`target_addr`
- 如果成功，以反向模式运行（服务端接收流量）
- 如果绑定失败，以正向模式运行（服务端发送流量）

**模式1：反向模式**（服务端接收流量）
1. 在`tunnel_addr`上监听TCP隧道连接（控制通道）
2. 绑定并在`target_addr`上监听传入的TCP和UDP流量
3. 当`target_addr`收到连接时，通过控制通道向已连接的客户端发送信号
4. 为每个连接创建具有指定TLS加密级别的数据通道

**模式2：正向模式**（服务端发送流量）
1. 在`tunnel_addr`上监听TCP隧道连接（控制通道）
2. 等待客户端在其本地监听，并通过隧道接收连接
3. 建立到远程`target_addr`的连接并转发数据

#### 示例

```bash
# 自动模式检测，无TLS加密
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=0"

# 强制反向模式，自签名证书
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=1&mode=1"

# 强制正向模式，自定义证书
nodepass "server://10.1.0.1:10101/192.168.1.100:8080?log=debug&tls=2&mode=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### 客户端模式

客户端模式连接到NodePass服务端并支持双向数据流转发。

```bash
nodepass "client://<tunnel_addr>/<target_addr>?log=<level>&mode=<run_mode>&min=<min_pool>&max=<max_pool>&read=<timeout>"
```

#### 参数

- `tunnel_addr`：要连接的NodePass服务端隧道端点地址(例如, 10.1.0.1:10101)
- `target_addr`：业务数据的目标地址，支持双向数据流模式(例如, 127.0.0.1:8080)
- `log`：日志级别(debug, info, warn, error, event)
- `mode`：客户端行为的运行模式控制
  - `0`：自动检测（默认）- 首先尝试本地绑定，如果失败则回退到握手模式
  - `1`：强制单端转发模式 - 带连接池的本地代理
  - `2`：强制双端握手模式 - 需要服务器协调
- `min`：最小连接池容量（默认：64）
- `max`：最大连接池容量（默认：1024）
- `read`：数据读取超时时间（默认：300s，支持时间单位如5s、30s、5m等）

#### 客户端模式工作原理

客户端模式通过`mode`参数支持自动模式检测或强制模式选择：

**模式0：自动检测**（默认）
- 首先尝试本地绑定`tunnel_addr`
- 如果成功，以单端转发模式运行
- 如果绑定失败，以双端握手模式运行

**模式1：单端转发模式**
1. 在本地隧道地址上监听TCP和UDP连接
2. 使用连接池技术预建立到目标地址的TCP连接，消除连接延迟
3. 直接将接收到的流量转发到目标地址，实现高性能转发
4. 无需与服务端握手，实现点对点的直接转发
5. 适用于本地代理和简单转发场景

**模式2：双端握手模式**
- **客户端接收流量**（当服务端发送流量时）
  1. 连接到服务端的TCP隧道端点（控制通道）
  2. 在本地监听端口，等待通过隧道传入的连接
  3. 建立到本地`target_addr`的连接并转发数据

- **客户端发送流量**（当服务端接收流量时）
  1. 连接到服务端的TCP隧道端点（控制通道）
  2. 通过控制通道监听来自服务端的信号
  3. 当收到信号时，使用服务端指定的TLS安全级别建立数据连接
  4. 建立到`target_addr`的连接并转发流量

#### 示例

```bash
# 自动模式检测 - 本地代理监听1080端口，转发到目标服务器
nodepass "client://127.0.0.1:1080/target.example.com:8080?log=debug"

# 强制单端转发模式 - 高性能本地代理
nodepass "client://127.0.0.1:1080/target.example.com:8080?mode=1&log=debug"

# 强制双端握手模式 - 连接到NodePass服务端并采用其TLS安全策略
nodepass "client://server.example.com:10101/127.0.0.1:8080?mode=2"

# 使用调试日志和自定义连接池容量连接
nodepass "client://server.example.com:10101/192.168.1.100:8080?log=debug&min=128&max=4096"

# 强制模式的资源受限配置
nodepass "client://server.example.com:10101/127.0.0.1:8080?mode=2&min=16&max=512&log=info"
```

# 资源受限配置 - 小型连接池
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=16&max=512&log=info"
```

### 主控模式 (API)

主控模式运行RESTful API服务器，用于集中管理NodePass实例。

```bash
nodepass "master://<api_addr>[<prefix>]?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>"
```

#### 参数

- `api_addr`：API服务监听的地址（例如，0.0.0.0:9090）
- `prefix`：可选的API前缀路径（例如，/management）。默认为`/api`
- `log`：日志级别(debug, info, warn, error, event)
- `tls`：API服务的TLS加密模式(0, 1, 2)
  - `0`：无TLS加密（HTTP）
  - `1`：自签名证书（带自动生成证书的HTTPS）
  - `2`：自定义证书（带提供证书的HTTPS）
- `crt`：证书文件路径（当`tls=2`时必需）
- `key`：私钥文件路径（当`tls=2`时必需）

#### 主控模式工作原理

在主控模式下，NodePass：
1. 运行一个RESTful API服务器，允许动态管理NodePass实例
2. 提供用于创建、启动、停止和监控客户端和服务端实例的端点
3. 包含用于轻松API探索的Swagger UI，位于`{prefix}/v1/docs`
4. 自动继承通过API创建的实例的TLS和日志设置

#### API端点

所有端点都是相对于配置的前缀（默认：`/api`）：

**受保护的端点（需要API Key）：**
- `GET {prefix}/v1/instances` - 列出所有实例
- `POST {prefix}/v1/instances` - 创建新实例，JSON请求体: `{"url": "server://0.0.0.0:10101/0.0.0.0:8080"}`
- `GET {prefix}/v1/instances/{id}` - 获取实例详情
- `PATCH {prefix}/v1/instances/{id}` - 更新实例，JSON请求体: `{"action": "start|stop|restart"}`
- `DELETE {prefix}/v1/instances/{id}` - 删除实例
- `GET {prefix}/v1/events` - 服务端发送事件流（SSE）
- `GET {prefix}/v1/info` - 获取系统信息

**公共端点（无需API Key）：**
- `GET {prefix}/v1/openapi.json` - OpenAPI规范
- `GET {prefix}/v1/docs` - Swagger UI文档

#### 示例

```bash
# 启动HTTP主控服务（使用默认API前缀/api）
nodepass "master://0.0.0.0:9090?log=info"

# 启动带有自定义API前缀的主控服务（/management）
nodepass "master://0.0.0.0:9090/management?log=info"

# 启动HTTPS主控服务（自签名证书）
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"

# 启动HTTPS主控服务（自定义证书）
nodepass "master://0.0.0.0:9090?log=info&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

## 管理NodePass实例

### 通过API创建和管理

NodePass主控模式提供RESTful API来管理实例，所有API请求都需要使用API Key进行身份验证。

#### API Key获取

启动主控模式后，系统会自动生成API Key并在日志中显示：

```bash
# 启动主控模式
nodepass "master://0.0.0.0:9090?log=info"

# 日志输出中会显示：
# INFO: API Key created: abc123def456...
```

#### API请求示例

所有受保护的API端点都需要在请求头中包含`X-API-Key`：

```bash
# 获取API Key (假设为: abc123def456789)

# 通过API创建实例（使用默认前缀）
curl -X POST http://localhost:9090/api/v1/instances \
  -H "X-API-Key: abc123def456789" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'

# 使用自定义前缀
curl -X POST http://localhost:9090/admin/v1/instances \
  -H "X-API-Key: abc123def456789" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'

# 列出所有运行实例
curl http://localhost:9090/api/v1/instances \
  -H "X-API-Key: abc123def456789"

# 控制实例（用实际实例ID替换{id}）
curl -X PATCH http://localhost:9090/api/v1/instances/{id} \
  -H "X-API-Key: abc123def456789" \
  -d '{"action":"restart"}'
```

## 双向数据流说明

NodePass支持灵活的双向数据流配置：

### 客户端单端转发模式
- **客户端**：在本地隧道地址监听，使用连接池技术直接转发到目标地址
- **连接池优化**：预建立TCP连接，消除连接延迟，提供高性能转发
- **无需服务端**：独立运行，不依赖服务端握手
- **使用场景**：本地代理、简单端口转发、测试环境、高性能转发

### 服务端接收模式
- **服务端**：在target_addr监听传入连接，通过隧道转发到客户端
- **客户端**：连接到本地target_addr提供服务
- **使用场景**：将内网服务暴露给外网访问

### 服务端发送模式
- **服务端**：连接到远程target_addr获取数据，通过隧道发送到客户端
- **客户端**：在本地监听，接收来自服务端的连接
- **使用场景**：通过隧道代理访问远程服务

系统会根据隧道地址和目标地址自动选择合适的操作模式：
- 如果客户端的隧道地址为本地地址，启用单端转发模式
- 如果目标地址是本地地址，使用服务端接收模式
- 如果目标地址是远程地址，使用服务端发送模式

## 隧道密钥（Tunnel Key）

NodePass使用隧道密钥来验证客户端和服务端之间的连接。密钥可以通过两种方式指定：

### 密钥获取规则

1. **显式密钥**：在URL中指定用户名部分作为密钥
   ```bash
   # 使用"mypassword"作为隧道密钥
   nodepass server://mypassword@10.1.0.1:10101/10.1.0.1:8080
   nodepass client://mypassword@10.1.0.1:10101/127.0.0.1:8080
   ```

2. **端口派生密钥**：如果未指定用户名，系统将使用端口号的十六进制值作为密钥
   ```bash
   # 端口10101的十六进制值为"2775"，将作为隧道密钥
   nodepass server://10.1.0.1:10101/10.1.0.1:8080
   nodepass client://10.1.0.1:10101/127.0.0.1:8080
   ```

### 握手流程

客户端与服务端的握手过程如下：

1. **客户端连接**：客户端连接到服务端的隧道地址
2. **密钥验证**：客户端发送XOR加密的隧道密钥
3. **服务端验证**：服务端解密并验证密钥是否匹配
4. **配置同步**：验证成功后，服务端发送隧道配置信息（包括TLS模式）
5. **连接确立**：握手完成，开始数据传输

这种设计确保了只有拥有正确密钥的客户端才能建立隧道连接。

## 下一步

- 了解[配置选项](/docs/zh/configuration.md)以微调NodePass
- 探索常见部署场景的[使用示例](/docs/zh/examples.md)
- 理解NodePass内部[工作原理](/docs/zh/how-it-works.md)
- 如果遇到问题，请查看[故障排除指南](/docs/zh/troubleshooting.md)