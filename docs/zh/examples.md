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
nodepass "client://127.0.0.1:1080/remote.example.com:8080?mode=1"

# 强制客户端以双端握手模式运行（需要服务器协调）
nodepass "client://server.example.com:10101/127.0.0.1:8080?mode=2&log=debug"
```

这些配置：
- **服务器 mode=1**：强制反向模式，服务器本地绑定目标地址
- **客户端 mode=1**：强制单端转发模式，使用直接连接实现高性能
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

### 示例8：服务间通信

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

## 带宽速率限制

### 示例9：带速率限制的文件传输服务器

控制文件传输服务的带宽使用：

```bash
# 服务端：限制文件传输带宽为100 Mbps
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=info&tls=1&rate=100"

# 客户端：连接时限制为50 Mbps
nodepass "client://fileserver.example.com:10101/127.0.0.1:3000?log=info&rate=50"
```

此配置：
- 限制服务器带宽为100 Mbps以防止网络拥塞
- 客户端进一步限制下载速度为50 Mbps以实现公平共享
- 允许文件传输的同时为其他服务保留带宽
- 使用TLS加密确保文件传输安全

### 示例10：物联网传感器数据收集的保守限制

对于带宽有限或按流量计费的物联网设备：

```bash
# 服务器：接受物联网数据，限制为5 Mbps
nodepass "server://0.0.0.0:10101/127.0.0.1:1883?log=warn&rate=5"

# 物联网设备客户端：发送传感器数据，限制为2 Mbps
nodepass "client://iot-gateway.example.com:10101/127.0.0.1:1883?log=error&rate=2"
```

此设置：
- 限制服务器为5 Mbps用于从多个物联网设备收集传感器数据
- 单个物联网客户端限制为2 Mbps以防止单一设备消耗所有带宽
- 最小日志记录（warn/error）以减少物联网设备的资源使用
- 高效适用于MQTT或其他物联网协议

### 示例11：开发环境速率控制

在带宽约束下测试应用程序：

```bash
# 模拟慢速网络条件进行测试
nodepass "client://api.example.com:443/127.0.0.1:8080?log=debug&rate=1"

# 带监控的高速开发服务器
nodepass "server://0.0.0.0:10101/127.0.0.1:3000?log=debug&rate=500"
```

此配置：
- 客户端模拟1 Mbps连接用于测试慢速网络场景
- 开发服务器限制为500 Mbps并提供详细日志记录用于调试
- 帮助识别不同带宽约束下的性能问题

## 物联网设备管理

### 示例12：带STUN NAT穿透的物联网网关

为NAT后的物联网设备创建中央访问点，无需端口转发：

```bash
# 家用路由器后的物联网设备 - 暴露Web界面
nodepass "client://stun.l.google.com:19302/127.0.0.1:80?mode=1&log=info"

# CGNAT后带MQTT代理的物联网设备
nodepass "client://stun1.l.google.com:19302/127.0.0.1:1883?mode=1&log=event"

# 带多个服务的智能家居中枢
nodepass "client://stun2.l.google.com:19302/192.168.1.100:8080?mode=1"
```

此配置：
- 允许从分布式物联网设备到外部客户端的安全连接
- 使用Google的STUN服务器发现公网端点
- 允许嵌入式设备暴露其本地接口而无需路由器配置
- 自动处理运营商级NAT后设备的NAT穿透
- 记录外部端点以便从外部连接：`External endpoint: <公网IP:端口> -> <本地IP:端口> -> <目标>`

### 示例13：通过NAT访问家庭服务器

无需开放路由器端口即可访问家庭服务：

```bash
# 家庭NAS文件服务器
nodepass "client://stun3.l.google.com:19302/localhost:445?mode=1&log=info"

# 家庭媒体服务器（Plex、Jellyfin等）
nodepass "client://stun4.l.google.com:19302/127.0.0.1:32400?mode=1"

# 家庭自动化服务器
nodepass "client://stun.l.google.com:19302/192.168.1.50:8123?mode=1&log=event"
```

此设置：
- 无需DMZ或端口转发即可提供对家庭服务的外部访问
- 通过避免路由器配置更改来保持安全性
- 适用于ISP的动态IP地址
- 适合带NAT的住宅互联网连接

### 示例14：远程开发环境

在NAT后启用对开发服务器的远程访问：

```bash
# 企业NAT后的开发Web服务器
nodepass "client://stun1.l.google.com:19302/localhost:3000?mode=1&log=debug"

# 数据库开发实例
nodepass "client://stun2.l.google.com:19302/127.0.0.1:5432?mode=1"

# 带热重载的API开发服务器
nodepass "client://stun3.l.google.com:19302/localhost:8080?mode=1&log=info"
```

此配置：
- 允许团队成员远程访问开发环境
- 适用于企业NAT和防火墙限制
- 无需VPN或复杂的网络配置
- 适合分布式开发团队

### 示例15：传统物联网网关（双端模式）

为传统隧道场景创建中央管理服务器：

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

## STUN NAT穿透用于远程访问

### 示例16：NAT后的SSH访问

为NAT后的机器提供SSH访问，无需端口转发：

```bash
# 家用计算机SSH访问
nodepass "client://stun.l.google.com:19302/localhost:22?mode=1&log=info"

# 运营商级NAT后的树莓派SSH
nodepass "client://stun1.l.google.com:19302/127.0.0.1:22?mode=1"

# 企业防火墙后的远程服务器
nodepass "client://stun2.l.google.com:19302/192.168.1.10:22?mode=1&log=event"
```

此设置：
- 无需路由器配置或VPN即可启用SSH访问
- 适用于阻止端口转发的运营商级NAT（CGNAT）
- 使用SSH的内置身份验证保持安全性
- 适合远程系统管理和故障排除

### 示例17：NAT后的游戏服务器托管

从家庭网络托管游戏服务器，无需端口转发：

```bash
# NAT后的Minecraft服务器
nodepass "client://stun3.l.google.com:19302/localhost:25565?mode=1&log=info"

# 反恐精英服务器
nodepass "client://stun4.l.google.com:19302/127.0.0.1:27015?mode=1"

# 带NAT穿透的泰拉瑞亚服务器
nodepass "client://stun.l.google.com:19302/192.168.1.100:7777?mode=1&log=event"
```

此配置：
- 允许从住宅连接托管游戏服务器
- 无需路由器端口转发或DMZ设置
- 适用于动态IP地址
- 玩家连接到发现的公网端点

### 示例18：VoIP和实时通信

在NAT后启用点对点VoIP和通信服务：

```bash
# NAT后的SIP服务器
nodepass "client://stun1.l.google.com:19302/localhost:5060?mode=1&log=info"

# Mumble语音聊天服务器
nodepass "client://stun2.l.google.com:19302/127.0.0.1:64738?mode=1"

# 带NAT穿透的TeamSpeak服务器
nodepass "client://stun3.l.google.com:19302/192.168.1.50:9987?mode=1&log=event"
```

此设置：
- 提供对VoIP和语音聊天服务器的外部访问
- 消除应用程序本身复杂的NAT穿透需求
- 适用于现有的SIP电话和语音聊天客户端
- 适合家庭办公室和小型企业

## 多环境开发

### 示例19：开发环境访问

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

## 高可用性与负载均衡

### 示例20：多后端服务器负载均衡

使用目标地址组实现流量均衡分配和自动故障转移：

```bash
# 服务端：配置3个后端Web服务器
nodepass "server://0.0.0.0:10101/web1.internal:8080,web2.internal:8080,web3.internal:8080?mode=2&tls=1&log=info"

# 客户端：连接到服务端
nodepass "client://server.example.com:10101/127.0.0.1:8080?log=info"
```

此配置：
- 流量自动轮询分配到3个后端服务器，实现负载均衡
- 当某个后端服务器故障时，自动切换到其他可用服务器
- 故障服务器恢复后自动重新接入流量
- 使用TLS加密确保隧道安全

### 示例15：数据库主从切换

为数据库配置主从实例，实现高可用访问：

```bash
# 客户端：配置主从数据库地址（单端转发模式）
nodepass "client://127.0.0.1:3306/db-primary.local:3306,db-secondary.local:3306?mode=1&log=warn"
```

此设置：
- 优先连接主数据库，主库故障时自动切换到从库
- 单端转发模式提供高性能本地代理
- 应用程序无需修改，透明地实现故障转移
- 仅记录警告和错误，减少日志输出

### 示例16：API网关后端池

为API网关配置多个后端服务实例：

```bash
# 服务端：配置4个API服务实例
nodepass "server://0.0.0.0:10101/api1.backend:8080,api2.backend:8080,api3.backend:8080,api4.backend:8080?mode=2&tls=1&rate=200&slot=5000"

# 客户端：从API网关连接
nodepass "client://apigateway.example.com:10101/127.0.0.1:8080?rate=100&slot=2000"
```

此配置：
- 4个API服务实例形成后端池，轮询分配请求
- 服务端限制带宽200 Mbps，最大5000并发连接
- 客户端限制带宽100 Mbps，最大2000并发连接
- 单个实例故障不影响整体服务可用性

### 示例17：地域分布式服务

配置多地域服务节点，优化网络延迟：

```bash
# 服务端：配置多地域节点
nodepass "server://0.0.0.0:10101/us-west.service:8080,us-east.service:8080,eu-central.service:8080?mode=2&log=debug"
```

此设置：
- 配置3个不同地域的服务节点
- 轮询算法自动分配流量到各个地域
- Debug日志帮助分析流量分布和故障情况
- 适用于全球分布式应用场景

**目标地址组最佳实践：**
- **地址数量**：建议配置2-5个地址，过多会增加故障检测时间
- **健康检查**：确保后端服务有自己的健康检查机制
- **端口一致性**：所有地址使用相同端口或明确指定每个地址的端口
- **监控告警**：配置监控系统跟踪故障转移事件
- **测试验证**：部署前在测试环境验证故障转移和负载均衡行为

## PROXY协议集成

### 示例18：负载均衡器与PROXY协议集成

启用PROXY协议支持，与负载均衡器和反向代理集成：

```bash
# 服务端：为HAProxy/Nginx集成启用PROXY协议v1
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=info&tls=1&proxy=1"

# 客户端：启用PROXY协议以保留客户端连接信息
nodepass "client://tunnel.example.com:10101/127.0.0.1:3000?log=info&proxy=1"
```

此配置：
- 在数据传输开始前发送PROXY协议v1头部
- 通过隧道保留原始客户端IP和端口信息
- 使后端服务能够看到真实的客户端连接详情
- 兼容HAProxy、Nginx和其他支持PROXY协议的服务
- 有助于维护准确的访问日志和基于IP的访问控制

### 示例19：Web应用的反向代理支持

使NodePass后的Web应用能够接收原始客户端信息：

```bash
# 为Web应用启用PROXY协议的NodePass服务器
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=warn&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem&proxy=1"

# 后端Web服务器（如Nginx）配置以处理PROXY协议
# 在nginx.conf中：
# server {
#     listen 8080 proxy_protocol;
#     real_ip_header proxy_protocol;
#     set_real_ip_from 127.0.0.1;
#     ...
# }
```

此设置：
- Web应用接收原始客户端IP地址而不是NodePass隧道IP
- 启用正确的访问日志记录、分析和安全控制
- 支持连接审计的合规性要求
- 适用于支持PROXY协议的Web服务器（Nginx、HAProxy等）

### 示例20：数据库访问与客户端IP保留

为数据库访问日志记录和安全维护客户端IP信息：

```bash
# 启用PROXY协议的数据库代理服务器
nodepass "server://0.0.0.0:10101/127.0.0.1:5432?log=error&proxy=1"

# 通过隧道连接的应用客户端
nodepass "client://dbproxy.example.com:10101/127.0.0.1:5432?proxy=1"
```

优势：
- 数据库日志显示原始应用服务器IP而不是隧道IP
- 启用基于IP的数据库访问控制正常工作
- 维护安全和合规的审计轨迹
- 兼容支持PROXY协议的数据库（适当配置的PostgreSQL）

**PROXY协议重要说明：**
- 目标服务必须支持PROXY协议v1才能正确处理头部
- PROXY头部仅对TCP连接发送，不支持UDP流量
- 头部包含：协议（TCP4/TCP6）、源IP、目标IP、源端口、目标端口
- 如果目标服务不支持PROXY协议，连接可能失败或行为异常
- 在生产环境部署前，请在非生产环境中充分测试启用PROXY协议的配置

## 容器部署

### 示例21：容器化NodePass

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

### 示例22：集中化管理

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

### 示例23：自定义API前缀

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

### 示例24：实时连接和流量监控

通过主控API监控实例的连接数和流量统计：

```bash
# 获取实例详细信息，包括连接数统计
curl -H "X-API-Key: your-api-key" http://localhost:9090/api/v1/instances/{id}

# 响应示例（包含TCPS和UDPS字段）
{
  "id": "a1b2c3d4",
  "alias": "网站代理",
  "type": "server",
  "status": "running", 
  "url": "server://0.0.0.0:10101/127.0.0.1:8080",
  "restart": true,
  "pool": 64,
  "ping": 25,
  "tcps": 12,
  "udps": 5,
  "tcprx": 1048576,
  "tcptx": 2097152,
  "udprx": 512000,
  "udptx": 256000
}

# 使用SSE实时监控所有实例状态变化
curl -H "X-API-Key: your-api-key" \
  -H "Accept: text/event-stream" \
  http://localhost:9090/api/v1/events
```

此监控设置提供：
- **实时连接数跟踪**：TCPS和UDPS字段显示当前活动连接数
- **性能分析**：通过连接数和流量数据评估系统负载
- **容量规划**：基于历史连接数据进行资源规划
- **故障诊断**：异常的连接数变化可能指示网络问题

## 下一步

现在您已经了解了各种使用示例，您可能想要：

- 了解[配置选项](/docs/zh/configuration.md)以进行微调
- 理解NodePass内部[工作原理](/docs/zh/how-it-works.md)
- 查看[故障排除指南](/docs/zh/troubleshooting.md)了解常见问题