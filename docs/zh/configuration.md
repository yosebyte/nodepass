# 配置选项

NodePass采用极简方法进行配置，所有设置都通过命令行参数和环境变量指定。本指南说明所有可用的配置选项，并为各种部署场景提供建议。

## 日志级别

NodePass提供六种日志详细级别，控制显示的信息量：

- `none`：禁用日志记录 - 不显示任何日志信息
- `debug`：详细调试信息 - 显示所有操作和连接
- `info`：一般操作信息(默认) - 显示启动、关闭和关键事件
- `warn`：警告条件 - 仅显示不影响核心功能的潜在问题
- `error`：错误条件 - 仅显示影响功能的问题
- `event`：事件记录 - 显示重要的操作事件和流量统计

您可以在命令URL中设置日志级别：

```bash
nodepass server://0.0.0.0:10101/0.0.0.0:8080?log=debug
```

## TLS加密模式

对于服务器和主控模式，NodePass为数据通道提供三种TLS安全级别：

- **模式0**：无TLS加密（明文TCP/UDP）
  - 最快性能，无开销
  - 数据通道无安全保护（仅在受信任网络中使用）
  
- **模式1**：自签名证书（自动生成）
  - 设置简单的良好安全性
  - 证书自动生成且不验证
  - 防止被动窃听
  
- **模式2**：自定义证书（需要`crt`和`key`参数）
  - 具有证书验证的最高安全性
  - 需要提供证书和密钥文件
  - 适用于生产环境

TLS模式1示例（自签名）：
```bash
nodepass server://0.0.0.0:10101/0.0.0.0:8080?tls=1
```

TLS模式2示例（自定义证书）：
```bash
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

## 运行模式控制

NodePass支持通过`mode`查询参数配置运行模式，以控制客户端和服务端实例的行为。这在自动模式检测不适合的部署场景中提供了灵活性。

### 客户端模式控制

对于客户端实例，`mode`参数控制连接策略：

- **模式0**（默认）：自动模式检测
  - 首先尝试本地绑定隧道地址
  - 如果成功，以单端转发模式运行
  - 如果绑定失败，以双端握手模式运行
  
- **模式1**：强制单端转发模式
  - 本地绑定隧道地址并直接转发流量到目标
  - 使用直接连接建立实现高性能
  - 无需与服务器握手
  
- **模式2**：强制双端握手模式
  - 始终连接到远程服务器建立隧道
  - 数据传输前需要与服务器握手
  - 支持双向数据流协调

示例：
```bash
# 强制客户端以单端转发模式运行
nodepass "client://127.0.0.1:1080/target.example.com:8080?mode=1"

# 强制客户端以双端握手模式运行
nodepass "client://server.example.com:10101/127.0.0.1:8080?mode=2"
```

### 服务端模式控制

对于服务端实例，`mode`参数控制数据流方向：

- **模式0**（默认）：自动流向检测
  - 首先尝试本地绑定目标地址
  - 如果成功，以反向模式运行（服务器接收流量）
  - 如果绑定失败，以正向模式运行（服务器发送流量）
  
- **模式1**：强制反向模式
  - 服务器本地绑定目标地址并接收流量
  - 入站连接转发到已连接的客户端
  - 数据流：外部 → 服务器 → 客户端 → 目标
  
- **模式2**：强制正向模式
  - 服务器连接到远程目标地址
  - 客户端连接转发到远程目标
  - 数据流：客户端 → 服务器 → 外部目标

示例：
```bash
# 强制服务器以反向模式运行
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?mode=1"

# 强制服务器以正向模式运行
nodepass "server://0.0.0.0:10101/remote.example.com:8080?mode=2"
```

## 连接池容量参数

连接池容量参数仅适用于双端握手模式，可以通过URL查询参数进行配置：

- `min`: 最小连接池容量（默认: 64）- 仅在双端握手模式下使用
- `max`: 最大连接池容量（默认: 1024）- 仅在双端握手模式下使用

注意：在客户端单端转发模式下，不使用连接池，这些参数被忽略。

示例：
```bash
# 设置最小连接池为32，最大为4096
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=32&max=4096"
```

## 数据读取超时
数据读取超时可以通过URL查询参数`read`设置，单位为秒或分钟：
- `read`: 数据读取超时时间（默认: 10分钟）

示例：
```bash
# 设置数据读取超时为5分钟
nodepass "client://server.example.com:10101/127.0.0.1:8080?read=5m"

# 设置数据读取超时为30秒，适用于快速响应应用
nodepass "client://server.example.com:10101/127.0.0.1:8080?read=30s"

# 设置数据读取超时为30分钟，适用于长时间传输
nodepass "client://server.example.com:10101/127.0.0.1:8080?read=30m"
```

## 速率限制
NodePass支持通过`rate`参数进行带宽速率限制，用于流量控制。此功能有助于防止网络拥塞，确保多个连接间的公平资源分配。

- `rate`: 最大带宽限制，单位为Mbps（兆比特每秒）
  - 值为0或省略：无速率限制（无限带宽）
  - 正整数：以Mbps为单位的速率限制（例如，10表示10 Mbps）
  - 同时应用于上传和下载流量
  - 使用令牌桶算法进行平滑流量整形

示例：
```bash
# 限制带宽为50 Mbps
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?rate=50"

# 客户端100 Mbps速率限制
nodepass "client://server.example.com:10101/127.0.0.1:8080?rate=100"

# 与其他参数组合使用
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?log=error&tls=1&rate=50"
```

**速率限制使用场景：**
- **带宽控制**：防止NodePass消耗所有可用带宽
- **公平共享**：确保多个应用程序可以共享网络资源
- **成本管理**：在按流量计费的网络环境中控制数据使用
- **QoS合规**：满足带宽使用的服务级别协议
- **测试**：模拟低带宽环境进行应用程序测试

## URL查询参数配置及作用范围

NodePass支持通过URL查询参数进行灵活配置，不同参数在 server、client、master 模式下的适用性如下表：

| 参数      | 说明                 | server | client | master |
|-----------|----------------------|:------:|:------:|:------:|
| `log`     | 日志级别             |   O    |   O    |   O    |
| `tls`     | TLS加密模式          |   O    |   X    |   O    |
| `crt`     | 自定义证书路径       |   O    |   X    |   O    |
| `key`     | 自定义密钥路径       |   O    |   X    |   O    |
| `min`     | 最小连接池容量       |   X    |   O    |   X    |
| `max`     | 最大连接池容量       |   O    |   O    |   X    |
| `mode`    | 运行模式控制         |   O    |   O    |   X    |
| `read`    | 读取超时时间         |   O    |   O    |   X    |
| `rate`    | 带宽速率限制         |   O    |   O    |   X    |



- O：参数有效，推荐根据实际场景配置
- X：参数无效，忽略设置

**最佳实践：**
- server/master 模式建议配置安全相关参数（如 tls、crt、key），提升数据通道安全性。
- client/server 双端握手模式建议根据流量和资源情况调整连接池容量（min/max），优化性能。
- 当自动检测不符合部署需求时或需要跨环境一致行为时，使用运行模式控制（mode）。
- 配置速率限制（rate）以控制带宽使用，防止共享环境中的网络拥塞。
- 日志级别（log）可在所有模式下灵活调整，便于运维和排查。

## 环境变量

可以使用环境变量微调NodePass行为。以下是所有可用变量的完整列表，包括其描述、默认值以及不同场景的推荐设置。

| 变量 | 描述 | 默认值 | 示例 |
|----------|-------------|---------|---------|
| `NP_SEMAPHORE_LIMIT` | 最大并发连接数 | 1024 | `export NP_SEMAPHORE_LIMIT=2048` |
| `NP_UDP_DATA_BUF_SIZE` | UDP数据包缓冲区大小 | 8192 | `export NP_UDP_DATA_BUF_SIZE=16384` |
| `NP_HANDSHAKE_TIMEOUT` | 握手操作超时 | 10s | `export NP_HANDSHAKE_TIMEOUT=30s` |
| `NP_TCP_DIAL_TIMEOUT` | TCP连接建立超时 | 10s | `export NP_TCP_DIAL_TIMEOUT=30s` |
| `NP_UDP_DIAL_TIMEOUT` | UDP连接建立超时 | 10s | `export NP_UDP_DIAL_TIMEOUT=30s` |
| `NP_POOL_GET_TIMEOUT` | 从连接池获取连接的超时时间 | 30s | `export NP_POOL_GET_TIMEOUT=60s` |
| `NP_MIN_POOL_INTERVAL` | 连接创建之间的最小间隔 | 100ms | `export NP_MIN_POOL_INTERVAL=200ms` |
| `NP_MAX_POOL_INTERVAL` | 连接创建之间的最大间隔 | 1s | `export NP_MAX_POOL_INTERVAL=3s` |
| `NP_REPORT_INTERVAL` | 健康检查报告间隔 | 5s | `export NP_REPORT_INTERVAL=10s` |
| `NP_SERVICE_COOLDOWN` | 重启尝试前的冷却期 | 3s | `export NP_SERVICE_COOLDOWN=5s` |
| `NP_SHUTDOWN_TIMEOUT` | 优雅关闭超时 | 5s | `export NP_SHUTDOWN_TIMEOUT=10s` |
| `NP_RELOAD_INTERVAL` | 证书/连接池重载间隔 | 1h | `export NP_RELOAD_INTERVAL=30m` |

### 连接池调优

连接池参数是双端握手模式下性能调优中的重要设置，在客户端单端转发模式下不适用：

#### 池容量设置

- `min` (URL参数)：确保最小可用连接数
  - 太低：流量高峰期延迟增加，因为必须建立新连接  
  - 太高：维护空闲连接浪费资源
  - 推荐起点：平均并发连接的25-50%

- `max` (URL参数)：防止过度资源消耗，同时处理峰值负载
  - 太低：流量高峰期连接失败
  - 太高：潜在资源耗尽影响系统稳定性
  - 推荐起点：峰值并发连接的150-200%

#### 池间隔设置

- `NP_MIN_POOL_INTERVAL`：控制连接创建尝试之间的最小时间
  - 太低：可能以连接尝试压垮网络
  - 推荐范围：根据网络延迟和预期负载，100ms-500ms

- `NP_MAX_POOL_INTERVAL`：控制连接创建尝试之间的最大时间
  - 太高：流量高峰期可能导致池耗尽
  - 推荐范围：根据预期流量模式，1s-5s

#### 连接管理

- `NP_SEMAPHORE_LIMIT`：控制最大并发隧道操作数
  - 太低：流量高峰期拒绝连接
  - 太高：太多并发goroutine可能导致内存压力
  - 推荐范围：大多数应用1000-5000，高吞吐量场景更高

### UDP设置

对于严重依赖UDP流量的应用：

- `NP_UDP_DATA_BUF_SIZE`：UDP数据包缓冲区大小
  - 对于发送大UDP数据包的应用增加此值
  - 默认值(8192)适用于大多数情况
  - 考虑为媒体流或游戏服务器增加到16384或更高

- `NP_UDP_DIAL_TIMEOUT`：UDP连接建立超时
  - 默认值(10s)为大多数应用提供良好平衡
  - 对于高延迟网络或响应缓慢的应用增加此值
  - 对于需要快速故障切换的低延迟应用减少此值

### TCP设置

对于TCP连接的优化：

- `NP_TCP_DIAL_TIMEOUT`：TCP连接建立超时
  - 默认值(10s)适用于大多数网络条件
  - 对于网络条件不稳定的环境增加此值
  - 对于需要快速判断连接成功与否的应用减少此值
  - 影响初始连接建立阶段

### 连接池管理设置

- `NP_POOL_GET_TIMEOUT`：从连接池获取连接时的最大等待时间
  - 默认值(30s)为连接建立提供充足时间
  - 对于高延迟环境或使用大型连接池时增加此值
  - 对于需要快速故障检测的应用减少此值
  - 在客户端单端转发模式下不使用连接池，此参数被忽略

### 服务管理设置

- `NP_REPORT_INTERVAL`：控制健康状态报告频率
  - 较低值提供更频繁的更新但增加日志量
  - 较高值减少日志输出但提供较少的即时可见性

- `NP_RELOAD_INTERVAL`：控制检查TLS证书变更的频率
  - 较低值更快检测证书变更但增加文件系统操作
  - 较高值减少开销但延迟检测证书更新

- `NP_SERVICE_COOLDOWN`：尝试服务重启前的等待时间
  - 较低值更快尝试恢复但可能在持续性问题情况下导致抖动
  - 较高值提供更多稳定性但从瞬态问题中恢复较慢

- `NP_SHUTDOWN_TIMEOUT`：关闭期间等待连接关闭的最长时间
  - 较低值确保更快关闭但可能中断活动连接
  - 较高值允许连接有更多时间完成但延迟关闭

## 推荐配置

以下是常见场景的推荐环境变量配置：

### 高吞吐量配置

对于需要最大吞吐量的应用（如媒体流、文件传输）：

URL参数：
```bash
# 高吞吐量服务器，1 Gbps速率限制
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?max=8192&rate=1000"

# 高吞吐量客户端，500 Mbps速率限制
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=128&max=8192&rate=500"
```

环境变量：
```bash
export NP_MIN_POOL_INTERVAL=50ms
export NP_MAX_POOL_INTERVAL=500ms
export NP_SEMAPHORE_LIMIT=8192
export NP_UDP_DATA_BUF_SIZE=32768
export NP_POOL_GET_TIMEOUT=60s
export NP_REPORT_INTERVAL=10s
```

### 低延迟配置

对于需要最小延迟的应用（如游戏、金融交易）：

URL参数：
```bash
# 低延迟服务器，适度速率限制
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?max=4096&rate=200"

# 低延迟客户端，适度速率限制
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=256&max=4096&rate=200"
```

环境变量：
```bash
export NP_MIN_POOL_INTERVAL=50ms
export NP_MAX_POOL_INTERVAL=500ms
export NP_SEMAPHORE_LIMIT=4096
export NP_TCP_DIAL_TIMEOUT=5s
export NP_UDP_DIAL_TIMEOUT=5s
export NP_POOL_GET_TIMEOUT=15s
export NP_REPORT_INTERVAL=1s
```

### 资源受限配置

对于在资源有限系统上的部署（如IoT设备、小型VPS）：

URL参数：
```bash
# 资源受限服务器，保守速率限制
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?max=512&rate=50"

# 资源受限客户端，保守速率限制
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=16&max=512&rate=50"
```

环境变量：
```bash
export NP_MIN_POOL_INTERVAL=200ms
export NP_MAX_POOL_INTERVAL=2s
export NP_SEMAPHORE_LIMIT=512
export NP_TCP_DIAL_TIMEOUT=20s
export NP_UDP_DIAL_TIMEOUT=20s
export NP_POOL_GET_TIMEOUT=45s
export NP_REPORT_INTERVAL=30s
export NP_SHUTDOWN_TIMEOUT=3s
```

## 下一步

- 查看[使用说明](/docs/zh/usage.md)了解基本操作命令
- 探索[使用示例](/docs/zh/examples.md)了解部署模式
- 了解[NodePass工作原理](/docs/zh/how-it-works.md)以优化配置
- 如果遇到问题，请查看[故障排除指南](/docs/zh/troubleshooting.md)