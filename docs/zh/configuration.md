# 配置选项

NodePass采用极简方法进行配置，所有设置都通过命令行参数和环境变量指定。本指南说明所有可用的配置选项，并为各种部署场景提供建议。

## 日志级别

NodePass提供五种日志详细级别，控制显示的信息量：

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
nodepass server://0.0.0.0:10101/0.0.0.0:8080?tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem
```

## 环境变量

可以使用环境变量微调NodePass行为。以下是所有可用变量的完整列表，包括其描述、默认值以及不同场景的推荐设置。

| 变量 | 描述 | 默认值 | 示例 |
|----------|-------------|---------|---------|
| `NP_SEMAPHORE_LIMIT` | 最大并发连接数 | 1024 | `export NP_SEMAPHORE_LIMIT=2048` |
| `NP_MIN_POOL_CAPACITY` | 最小连接池大小 | 16 | `export NP_MIN_POOL_CAPACITY=32` |
| `NP_MAX_POOL_CAPACITY` | 最大连接池大小 | 1024 | `export NP_MAX_POOL_CAPACITY=4096` |
| `NP_UDP_DATA_BUF_SIZE` | UDP数据包缓冲区大小 | 8192 | `export NP_UDP_DATA_BUF_SIZE=16384` |
| `NP_UDP_READ_TIMEOUT` | UDP读取操作超时 | 15s | `export NP_UDP_READ_TIMEOUT=30s` |
| `NP_TCP_READ_TIMEOUT` | TCP读取操作超时 | 15s | `export NP_TCP_READ_TIMEOUT=30s` |
| `NP_UDP_DIAL_TIMEOUT` | UDP拨号超时 | 15s | `export NP_UDP_DIAL_TIMEOUT=30s` |
| `NP_TCP_DIAL_TIMEOUT` | TCP拨号超时 | 15s | `export NP_TCP_DIAL_TIMEOUT=30s` |
| `NP_MIN_POOL_INTERVAL` | 连接创建之间的最小间隔 | 1s | `export NP_MIN_POOL_INTERVAL=500ms` |
| `NP_MAX_POOL_INTERVAL` | 连接创建之间的最大间隔 | 5s | `export NP_MAX_POOL_INTERVAL=3s` |
| `NP_REPORT_INTERVAL` | 健康检查报告间隔 | 5s | `export NP_REPORT_INTERVAL=10s` |
| `NP_SERVICE_COOLDOWN` | 重启尝试前的冷却期 | 5s | `export NP_SERVICE_COOLDOWN=3s` |
| `NP_SHUTDOWN_TIMEOUT` | 优雅关闭超时 | 5s | `export NP_SHUTDOWN_TIMEOUT=10s` |
| `NP_RELOAD_INTERVAL` | 证书/连接池重载间隔 | 1h | `export NP_RELOAD_INTERVAL=30m` |

### 连接池调优

连接池参数是性能调优中最重要的设置之一：

#### 池容量设置

- `NP_MIN_POOL_CAPACITY`：确保最小可用连接数
  - 太低：流量高峰期延迟增加，因为必须建立新连接
  - 太高：维护空闲连接浪费资源
  - 推荐起点：平均并发连接的25-50%

- `NP_MAX_POOL_CAPACITY`：防止过度资源消耗，同时处理峰值负载
  - 太低：流量高峰期连接失败
  - 太高：潜在资源耗尽影响系统稳定性
  - 推荐起点：峰值并发连接的150-200%

#### 池间隔设置

- `NP_MIN_POOL_INTERVAL`：控制连接创建尝试之间的最小时间
  - 太低：可能以连接尝试压垮网络
  - 推荐范围：根据网络延迟，500ms-2s

- `NP_MAX_POOL_INTERVAL`：控制连接创建尝试之间的最大时间
  - 太高：流量高峰期可能导致池耗尽
  - 推荐范围：根据预期流量模式，3s-10s

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

- `NP_UDP_READ_TIMEOUT`：UDP读取操作超时
  - 对于高延迟网络或响应时间慢的应用增加此值
  - 对于需要快速故障转移的低延迟应用减少此值

- `NP_UDP_DIAL_TIMEOUT`：UDP拨号超时
  - 对于高延迟网络增加此值
  - 对于需要快速连接的应用减少此值

### TCP设置

对于TCP连接的优化：

- `NP_TCP_READ_TIMEOUT`：TCP读取操作超时
  - 对于高延迟网络或响应慢的服务器增加此值
  - 对于需要快速检测断开连接的应用降低此值
  - 影响数据传输过程中的等待时间

- `NP_TCP_DIAL_TIMEOUT`：TCP连接建立超时
  - 对于网络条件不稳定的环境增加此值
  - 对于需要快速判断连接成功与否的应用减少此值
  - 影响初始连接建立阶段

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

## 配置配置文件

以下是常见场景的推荐环境变量配置：

### 高吞吐量配置

对于需要最大吞吐量的应用（如媒体流、文件传输）：

```bash
export NP_MIN_POOL_CAPACITY=64
export NP_MAX_POOL_CAPACITY=4096
export NP_MIN_POOL_INTERVAL=500ms
export NP_MAX_POOL_INTERVAL=3s
export NP_SEMAPHORE_LIMIT=8192
export NP_UDP_DATA_BUF_SIZE=32768
export NP_REPORT_INTERVAL=10s
```

### 低延迟配置

对于需要最小延迟的应用（如游戏、金融交易）：

```bash
export NP_MIN_POOL_CAPACITY=128
export NP_MAX_POOL_CAPACITY=2048
export NP_MIN_POOL_INTERVAL=100ms
export NP_MAX_POOL_INTERVAL=1s
export NP_SEMAPHORE_LIMIT=4096
export NP_UDP_READ_TIMEOUT=10s
export NP_REPORT_INTERVAL=1s
```

### 资源受限配置

对于在资源有限系统上的部署（如IoT设备、小型VPS）：

```bash
export NP_MIN_POOL_CAPACITY=8
export NP_MAX_POOL_CAPACITY=256
export NP_MIN_POOL_INTERVAL=2s
export NP_MAX_POOL_INTERVAL=10s
export NP_SEMAPHORE_LIMIT=512
export NP_REPORT_INTERVAL=30s
export NP_SHUTDOWN_TIMEOUT=3s
```

## 下一步

- 查看[使用说明](/docs/zh/usage.md)了解基本操作命令
- 探索[使用示例](/docs/zh/examples.md)了解部署模式
- 了解[NodePass工作原理](/docs/zh/how-it-works.md)以优化配置
- 如果遇到问题，请查看[故障排除指南](/docs/zh/troubleshooting.md)