# 使用示例

本页提供了NodePass在各种部署场景中的实际示例。这些示例涵盖了常见用例，可以根据您的具体需求进行调整。

## 基本服务器设置与TLS选项

### 示例1：无TLS加密

当速度比安全性更重要时（例如，在受信任网络中）：

```bash
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=0"
```

这会启动一个NodePass服务器，它：
- 在所有接口的10101端口上监听隧道连接
- 将流量转发到localhost:8080
- 使用debug日志记录详细信息
- 不对数据通道使用加密（最快性能）

### 示例2：自签名证书

为了平衡安全性和易于设置（推荐大多数情况）：

```bash
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"
```

此配置：
- 自动生成自签名证书
- 提供加密而无需证书管理
- 保护数据流量免受被动窃听
- 适用于内部或测试环境

### 示例3：自定义域名证书

对于需要验证证书的生产环境：

```bash
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

这一设置：
- 使用您提供的TLS证书和私钥
- 提供具有证书验证的最高安全级别
- 适合生产环境和面向公众的服务
- 允许客户端验证服务器的身份

## 连接到NodePass服务器

### 示例4：基本客户端连接

使用默认设置连接到NodePass服务器：

```bash
nodepass client://server.example.com:10101/127.0.0.1:8080
```

此客户端：
- 连接到server.example.com:10101的NodePass服务器
- 将接收到的流量转发到localhost:8080
- 自动采用服务器的TLS安全策略
- 使用默认的info日志级别

### 示例5：带调试日志的客户端

用于故障排除连接问题：

```bash
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

这启用了详细输出，有助于识别：
- 连接建立问题
- 信号处理
- 数据传输详情
- 错误情况

### 示例6：运行模式控制

通过明确的模式设置控制操作行为：

```bash
# 强制服务器以反向模式运行（服务器接收流量）
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?mode=1&tls=1"

# 强制客户端以单端转发模式运行（高性能本地代理）
nodepass "client://127.0.0.1:1080/remote.example.com:8080?mode=1&min=128&max=1024"

# 强制客户端以双端握手模式运行（需要服务器协调）
nodepass "client://server.example.com:10101/127.0.0.1:8080?mode=2&log=debug"
```

这些配置：
- **服务器 mode=1**：强制反向模式，服务器本地绑定目标地址
- **客户端 mode=1**：强制单端转发模式，使用连接池实现最大性能
- **客户端 mode=2**：强制双端握手模式，适用于需要服务器协调的场景
- 当自动检测不符合部署需求时使用模式控制

## 通过防火墙访问数据库

### 示例7：数据库隧道

启用对防火墙后的数据库服务器的安全访问：

```bash
# 服务器端(位于安全网络外部)使用TLS加密
nodepass server://:10101/127.0.0.1:5432?tls=1

# 客户端(位于防火墙内部)
nodepass client://server.example.com:10101/127.0.0.1:5432
```

此配置：
- 创建到PostgreSQL数据库（端口5432）的加密隧道
- 允许安全访问数据库而不直接将其暴露于互联网
- 使用自签名证书加密所有数据库流量
- 使远程数据库在客户端上显示为本地服务

## 安全的微服务通信

### 示例7：服务间通信

启用微服务之间的安全通信：

```bash
# 服务A(消费API)使用自定义证书
nodepass "server://0.0.0.0:10101/127.0.0.1:8081?log=warn&tls=2&crt=/path/to/service-a.crt&key=/path/to/service-a.key"

# 服务B(提供API)
nodepass client://service-a:10101/127.0.0.1:8082
```

此设置：
- 在两个微服务之间创建安全通道
- 使用自定义证书进行服务身份验证
- 将日志限制为仅警告和错误
- 使服务A的API在服务B上显示为本地服务

## 物联网设备管理

### 示例8：物联网网关

创建物联网设备的中央访问点：

```bash
# 中央管理服务器
nodepass "server://0.0.0.0:10101/127.0.0.1:8888?log=info&tls=1"

# 物联网设备
nodepass client://mgmt.example.com:10101/127.0.0.1:80
```

此配置：
- 使分布式物联网设备能够安全连接到中央服务器
- 使用自签名证书提供足够的安全性
- 允许嵌入式设备安全地暴露其本地Web界面
- 通过单一端点集中设备管理

## 多环境开发

### 示例9：开发环境访问

通过隧道访问不同的开发环境：

```bash
# 生产API访问隧道
nodepass client://tunnel.example.com:10101/127.0.0.1:3443

# 开发环境
nodepass server://tunnel.example.com:10101/127.0.0.1:3000

# 测试环境
nodepass "server://tunnel.example.com:10101/127.0.0.1:3001?log=warn&tls=1"
```

此设置：
- 创建对多个环境（生产、开发、测试）的安全访问
- 根据环境敏感性使用不同级别的日志记录
- 使开发人员能够访问环境而无需直接网络暴露
- 将远程服务映射到不同的本地端口，便于识别

## 容器部署

### 示例10：容器化NodePass

在Docker环境中部署NodePass：

```bash
# 为容器创建网络
docker network create nodepass-net

# 部署使用自签名证书的NodePass服务器
docker run -d --name nodepass-server \
  --network nodepass-net \
  -p 10101:10101 \
  ghcr.io/yosebyte/nodepass "server://0.0.0.0:10101/web-service:80?log=info&tls=1"

# 部署Web服务作为目标
docker run -d --name web-service \
  --network nodepass-net \
  nginx:alpine

# 部署NodePass客户端
docker run -d --name nodepass-client \
  -p 8080:8080 \
  ghcr.io/yosebyte/nodepass client://nodepass-server:10101/127.0.0.1:8080?log=info

# 通过http://localhost:8080访问Web服务
```

此配置：
- 在服务之间创建容器化隧道
- 使用Docker网络连接容器
- 仅向主机公开必要端口
- 提供对内部Web服务的安全访问

## 主控API管理

### 示例11：集中管理

为多个NodePass实例设置中央控制器：

```bash
# 使用自签名证书启动主控API服务
nodepass "master://0.0.0.0:9090?log=info&tls=1"
```

然后您可以通过API调用管理实例：

```bash
# 创建服务器实例
curl -X POST http://localhost:9090/api/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'

# 创建客户端实例
curl -X POST http://localhost:9090/api/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"client://localhost:10101/127.0.0.1:8081"}'

# 列出所有运行实例
curl http://localhost:9090/api/v1/instances

# 控制实例（用实际实例ID替换{id}）
curl -X PUT http://localhost:9090/api/v1/instances/{id} \
  -H "Content-Type: application/json" \
  -d '{"action":"restart"}'
```

此设置：
- 为所有NodePass实例提供中央管理界面
- 允许动态创建和控制隧道
- 提供用于自动化和集成的RESTful API
- 包含内置的Swagger UI，位于http://localhost:9090/api/v1/docs

### 示例12：自定义API前缀

为主控模式使用自定义API前缀：

```bash
# 使用自定义API前缀启动
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"

# 使用自定义前缀创建实例
curl -X POST http://localhost:9090/admin/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'
```

这允许：
- 与现有API网关集成
- 用于安全或组织目的的自定义URL路径
- 在http://localhost:9090/admin/v1/docs访问Swagger UI

## 下一步

现在您已经了解了各种使用示例，您可能想要：

- 了解[配置选项](/docs/zh/configuration.md)以进行微调
- 理解NodePass内部[工作原理](/docs/zh/how-it-works.md)
- 查看[故障排除指南](/docs/zh/troubleshooting.md)了解常见问题