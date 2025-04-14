# 配置选项

NodePass采用极简方法进行配置，所有设置都通过命令行参数和环境变量指定。本指南说明所有可用的配置选项，并为各种部署场景提供建议。

## 日志级别

NodePass提供五种日志详细级别，控制显示的信息量：

- `debug`：详细调试信息 - 显示所有操作和连接
- `info`：一般操作信息(默认) - 显示启动、关闭和关键事件
- `warn`：警告条件 - 仅显示不影响核心功能的潜在问题
- `error`：错误条件 - 仅显示影响功能的问题
- `fatal`：致命条件 - 仅显示导致终止的严重错误

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
| `SEMAPHORE_LIMIT` | 最大并发连接数 | 1024 | `export SEMAPHORE_LIMIT=2048` |
| `MIN_POOL_CAPACITY` | 最小连接池大小 | 16 | `export MIN_POOL_CAPACITY=32` |
| `MAX_POOL_CAPACITY` | 最大连接池大小 | 1024 | `export MAX_POOL_CAPACITY=4096` |
| `MIN_POOL_INTERVAL` | 连接创建之间的最小间隔 | 1s | `export MIN_POOL_INTERVAL=500ms` |
| `MAX_POOL_INTERVAL` | 连接创建之间的最大间隔 | 5s | `export MAX_POOL_INTERVAL=3s` |
| `UDP_DATA_BUF_SIZE` | UDP数据包缓冲区大小 | 8192 | `export UDP_DATA_BUF_SIZE=16384` |
| `UDP_READ_TIMEOUT` | UDP读取操作超时 | 5s | `export UDP_READ_TIMEOUT=10s` |
| `REPORT_INTERVAL` | 健康检查报告间隔 | 5s | `export REPORT_INTERVAL=10s` |
| `RELOAD_INTERVAL` | 证书重载间隔 | 1h | `export RELOAD_INTERVAL=30m` |
| `SERVICE_COOLDOWN` | 重启尝试前的冷却期 | 5s | `export SERVICE_COOLDOWN=3s` |
| `SHUTDOWN_TIMEOUT` | 优雅关闭超时 | 5s | `export SHUTDOWN_TIMEOUT=10s` |

### 连接池调优

连接池参数是性能调优中最重要的设置之一：

#### 池容量设置

- `MIN_POOL_CAPACITY`：确保最小可用连接数
  - 太低：流量高峰期延迟增加，因为必须建立新连接
  - 太高：维护空闲连接浪费资源
  - 推荐起点：平均并发连接的25-50%

- `MAX_POOL_CAPACITY`：防止过度资源消耗，同时处理峰值负载
  - 太低：流量高峰期连接失败
  - 太高：潜在资源耗尽影响系统稳定性
  - 推荐起点：峰值并发连接的150-200%

#### 池间隔设置

- `MIN_POOL_INTERVAL`：控制连接创建尝试之间的最小时间
  - 太低：可能以连接尝试压垮网络
  - 推荐范围：根据网络延迟，500ms-2s

- `MAX_POOL_INTERVAL`：控制连接创建尝试之间的最大时间
  - 太高：流量高峰期可能导致池耗尽
  - 推荐范围：根据预期流量模式，3s-10s

#### 连接管理

- `SEMAPHORE_LIMIT`：控制最大并发隧道操作数
  - 太低：流量高峰期拒绝连接
  - 太高：太多并发goroutine可能导致内存压力
  - 推荐范围：大多数应用1000-5000，高吞吐量场景更高

### UDP设置

对于严重依赖UDP流量的应用：

- `UDP_DATA_BUF_SIZE`：UDP数据包缓冲区大小
  - 对于发送大UDP数据包的应用增加此值
  - 默认值(8192)适用于大多数情况
  - 考虑为媒体流或游戏服务器增加到16384或更高

- `UDP_READ_TIMEOUT`：UDP读取操作超时
  - 对于高延迟网络或响应时间慢的应用增加此值
  - 对于需要快速故障转移的低延迟应用减少此值

### 服务管理设置

- `REPORT_INTERVAL`：控制健康状态报告频率
  - 较低值提供更频繁的更新但增加日志量
  - 较高值减少日志输出但提供较少的即时可见性

- `RELOAD_INTERVAL`：控制检查TLS证书变更的频率
  - 较低值更快检测证书变更但增加文件系统操作
  - 较高值减少开销但延迟检测证书更新

- `SERVICE_COOLDOWN`：尝试服务重启前的等待时间
  - 较低值更快尝试恢复但可能在持续性问题情况下导致抖动
  - 较高值提供更多稳定性但从瞬态问题中恢复较慢

- `SHUTDOWN_TIMEOUT`：关闭期间等待连接关闭的最长时间
  - 较低值确保更快关闭但可能中断活动连接
  - 较高值允许连接有更多时间完成但延迟关闭

## 配置配置文件

以下是常见场景的推荐环境变量配置：

### 高吞吐量配置

对于需要最大吞吐量的应用（如媒体流、文件传输）：

```bash
export MIN_POOL_CAPACITY=64
export MAX_POOL_CAPACITY=4096
export MIN_POOL_INTERVAL=500ms
export MAX_POOL_INTERVAL=3s
export SEMAPHORE_LIMIT=8192
export UDP_DATA_BUF_SIZE=32768
export REPORT_INTERVAL=10s
```

### 低延迟配置

对于需要最小延迟的应用（如游戏、金融交易）：

```bash
export MIN_POOL_CAPACITY=128
export MAX_POOL_CAPACITY=2048
export MIN_POOL_INTERVAL=100ms
export MAX_POOL_INTERVAL=1s
export SEMAPHORE_LIMIT=4096
export UDP_READ_TIMEOUT=2s
export REPORT_INTERVAL=1s
```

### 资源受限配置

对于在资源有限系统上的部署（如IoT设备、小型VPS）：

```bash
export MIN_POOL_CAPACITY=8
export MAX_POOL_CAPACITY=256
export MIN_POOL_INTERVAL=2s
export MAX_POOL_INTERVAL=10s
export SEMAPHORE_LIMIT=512
export REPORT_INTERVAL=30s
export SHUTDOWN_TIMEOUT=3s
```

## 下一步

- 查看[使用说明](/docs/zh/usage.md)了解基本操作命令
- 探索[使用示例](/docs/zh/examples.md)了解部署模式
- 了解[NodePass工作原理](/docs/zh/how-it-works.md)以优化配置
- 如果遇到问题，请查看[故障排除指南](/docs/zh/troubleshooting.md)