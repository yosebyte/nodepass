# 使用说明

NodePass创建一个带有未加密TCP控制通道的隧道，并为数据交换提供可配置的TLS加密选项。本指南涵盖三种操作模式并说明如何有效地使用每种模式。

## 命令行语法

NodePass命令的一般语法是：

```bash
nodepass <core>://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

其中：
- `<core>`：指定操作模式（`server`、`client`或`master`）
- `<tunnel_addr>`：控制通道通信的隧道端点地址
- `<target_addr>`：转发流量的目标地址（或在master模式下的API前缀）
- `<level>`：日志详细级别（`debug`、`info`、`warn`、`error`或`fatal`）
- `<mode>`：数据通道的TLS安全级别（`0`、`1`或`2`）- 仅适用于server/master模式
- `<cert_file>`：证书文件路径（当`tls=2`时）- 仅适用于server/master模式
- `<key_file>`：私钥文件路径（当`tls=2`时）- 仅适用于server/master模式

## 运行模式

NodePass提供三种互补的运行模式，以适应各种部署场景。

### 服务器模式

服务器模式监听客户端连接并通过隧道从目标地址转发流量。

```bash
nodepass server://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

#### 参数

- `tunnel_addr`：TCP隧道端点地址（控制通道），客户端将连接到此处(例如, 10.1.0.1:10101)
- `target_addr`：服务器监听传入连接(TCP和UDP)的地址，这些连接将被隧道传输到客户端(例如, 10.1.0.1:8080)
- `log`：日志级别(debug, info, warn, error, fatal)
- `tls`：目标数据通道的TLS加密模式 (0, 1, 2)
  - `0`：无TLS加密（明文TCP/UDP）
  - `1`：自签名证书（自动生成）
  - `2`：自定义证书（需要`crt`和`key`参数）
- `crt`：证书文件路径（当`tls=2`时必需）
- `key`：私钥文件路径（当`tls=2`时必需）

#### 服务器模式工作原理

在服务器模式下，NodePass：
1. 在`tunnel_addr`上监听TCP隧道连接（控制通道）
2. 在`target_addr`上监听传入的TCP和UDP流量
3. 当`target_addr`收到连接时，通过未加密的TCP隧道向客户端发送信号
4. 为每个连接创建具有指定TLS加密级别的数据通道

#### 示例

```bash
# 数据通道无TLS加密
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=0"

# 自签名证书（自动生成）
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=1"

# 自定义域名证书
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### 客户端模式

客户端模式连接到NodePass服务器并将流量转发到本地目标地址。

```bash
nodepass client://<tunnel_addr>/<target_addr>?log=<level>
```

#### 参数

- `tunnel_addr`：要连接的NodePass服务器隧道端点地址(例如, 10.1.0.1:10101)
- `target_addr`：流量将被转发到的本地地址(例如, 127.0.0.1:8080)
- `log`：日志级别(debug, info, warn, error, fatal)

#### 客户端模式工作原理

在客户端模式下，NodePass：
1. 连接到服务器的未加密TCP隧道端点（控制通道）
2. 通过此控制通道监听来自服务器的信号
3. 当收到信号时，使用服务器指定的TLS安全级别建立数据连接
4. 在`target_addr`建立本地连接并转发流量

#### 示例

```bash
# 连接到NodePass服务器并自动采用其TLS安全策略
nodepass client://server.example.com:10101/127.0.0.1:8080

# 使用调试日志连接
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### 主控模式 (API)

主控模式运行RESTful API服务器，用于集中管理NodePass实例。

```bash
nodepass master://<api_addr>[<prefix>]?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

#### 参数

- `api_addr`：API服务监听的地址（例如，0.0.0.0:9090）
- `prefix`：可选的API前缀路径（例如，/management）。默认为`/api`
- `log`：日志级别(debug, info, warn, error, fatal)
- `tls`：API服务的TLS加密模式(0, 1, 2)
  - `0`：无TLS加密（HTTP）
  - `1`：自签名证书（带自动生成证书的HTTPS）
  - `2`：自定义证书（带提供证书的HTTPS）
- `crt`：证书文件路径（当`tls=2`时必需）
- `key`：私钥文件路径（当`tls=2`时必需）

#### 主控模式工作原理

在主控模式下，NodePass：
1. 运行一个RESTful API服务器，允许动态管理NodePass实例
2. 提供用于创建、启动、停止和监控客户端和服务器实例的端点
3. 包含用于轻松API探索的Swagger UI，位于`{prefix}/v1/docs`
4. 自动继承通过API创建的实例的TLS和日志设置

#### API端点

所有端点都是相对于配置的前缀（默认：`/api`）：

- `GET {prefix}/v1/instances` - 列出所有实例
- `POST {prefix}/v1/instances` - 创建新实例，JSON请求体: `{"url": "server://0.0.0.0:10101/0.0.0.0:8080"}`
- `GET {prefix}/v1/instances/{id}` - 获取实例详情
- `PUT {prefix}/v1/instances/{id}` - 更新实例，JSON请求体: `{"action": "start|stop|restart"}`
- `DELETE {prefix}/v1/instances/{id}` - 删除实例
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

您可以使用标准HTTP请求通过主控API管理NodePass实例：

```bash
# 通过API创建和管理实例（使用默认前缀）
curl -X POST http://localhost:9090/api/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'

# 使用自定义前缀
curl -X POST http://localhost:9090/admin/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'

# 列出所有运行实例
curl http://localhost:9090/api/v1/instances

# 控制实例（用实际实例ID替换{id}）
curl -X PUT http://localhost:9090/api/v1/instances/{id} \
  -H "Content-Type: application/json" \
  -d '{"action":"restart"}'
```

## 下一步

- 了解[配置选项](/docs/zh/configuration.md)以微调NodePass
- 探索常见部署场景的[使用示例](/docs/zh/examples.md)
- 理解NodePass内部[工作原理](/docs/zh/how-it-works.md)
- 如果遇到问题，请查看[故障排除指南](/docs/zh/troubleshooting.md)