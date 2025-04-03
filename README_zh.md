# 🔗 NodePass

[![GitHub release](https://img.shields.io/github/v/release/yosebyte/nodepass)](https://github.com/yosebyte/nodepass/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yosebyte/nodepass)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/yosebyte/nodepass)](https://goreportcard.com/report/github.com/yosebyte/nodepass)

<div align="center">
  <img src="https://cdn.yobc.de/assets/nodepass.png" alt="nodepass">
</div>

**语言**: [English](README.md) | [简体中文](README_zh.md)

NodePass是一个优雅、高效的TCP隧道解决方案，可在网络端点之间创建安全的通信桥梁。通过建立一个未加密的TCP控制通道，NodePass能够在受限网络环境中实现无缝数据传输，同时为数据通道提供可配置的安全选项。其服务器-客户端架构允许灵活部署，使服务能够穿越防火墙、NAT和其他网络障碍。凭借智能连接池、最小资源占用和简洁的命令语法，NodePass为开发人员和系统管理员提供了一个强大且易用的工具，可以解决复杂的网络挑战，同时不影响安全性或性能。

## 📋 目录

- [功能特点](#-功能特点)
- [系统要求](#-系统要求)
- [安装方法](#-安装方法)
  - [方式1: 预编译二进制文件](#-方式1-预编译二进制文件)
  - [方式2: 使用Go安装](#-方式2-使用go安装)
  - [方式3: 从源代码构建](#️-方式3-从源代码构建)
  - [方式4: 使用容器镜像](#-方式4-使用容器镜像)
  - [方式5: 使用管理脚本](#-方式5-使用管理脚本)
- [使用方法](#-使用方法)
  - [服务器模式](#️-服务器模式)
  - [客户端模式](#-客户端模式)
- [配置选项](#️-配置选项)
  - [日志级别](#-日志级别)
  - [环境变量](#-环境变量)
- [使用示例](#-使用示例)
  - [基本服务器设置与TLS选项](#-基本服务器设置与tls选项)
  - [连接到NodePass服务器](#-连接到nodepass服务器)
  - [通过防火墙访问数据库](#-通过防火墙访问数据库)
  - [安全的微服务通信](#-安全的微服务通信)
  - [物联网设备管理](#-物联网设备管理)
  - [多环境开发](#-多环境开发)
  - [容器部署](#-容器部署)
- [工作原理](#-工作原理)
- [数据传输流程](#-数据传输流程)
- [信号通信机制](#-信号通信机制)
- [连接池架构](#-连接池架构)
- [常见使用场景](#-常见使用场景)
- [故障排除](#-故障排除)
  - [连接问题](#-连接问题)
  - [性能优化](#-性能优化)
- [贡献指南](#-贡献指南)
- [社区讨论](#-社区讨论)
- [许可协议](#-许可协议)
- [Star趋势](#-Star趋势)

## ✨ 功能特点

<div align="center">
  <img src="https://cdn.yobc.de/assets/np-cli.png" alt="nodepass">
</div>

- **🔄 双重操作模式**: 可作为服务器接受连接或作为客户端发起连接
- **🌐 TCP/UDP协议支持**: 支持TCP和UDP流量隧道传输，确保完整的应用程序兼容性
- **🔒 灵活的TLS选项**: 数据通道加密的三种安全模式
- **🔐 自动TLS策略采用**: 客户端自动采用服务器的TLS安全策略
- **🔌 高效连接池**: 优化的连接管理，支持可配置的池大小
- **📊 灵活的日志系统**: 可配置的五种不同日志级别
- **🛡️ 弹性错误处理**: 自动连接恢复和优雅关闭
- **📦 单一二进制部署**: 简单分发和安装，依赖项极少
- **⚙️ 零配置文件**: 所有设置通过命令行参数和环境变量指定
- **🚀 低资源占用**: 即使在高负载下也能保持最小的CPU和内存使用
- **♻️ 自动重连**: 从网络中断中无缝恢复
- **🧩 模块化架构**: 客户端、服务器和公共组件之间清晰分离
- **🔍 全面调试**: 详细的连接追踪和信号监控
- **⚡ 高性能数据交换**: 优化的双向数据传输机制
- **🧠 智能连接管理**: 智能处理连接状态和生命周期
- **📈 可扩展信号量系统**: 防止高流量期间资源耗尽
- **🛠️ 可配置池动态**: 根据工作负载调整连接池行为
- **🔌 一次性连接模式**: 通过非重用连接增强安全性
- **📡 动态端口分配**: 自动管理安全通信的端口分配

## 📋 系统要求

- Go 1.24或更高版本(从源代码构建时需要)
- 服务器和客户端端点之间的网络连接
- 绑定1024以下端口可能需要管理员权限

## 📥 安装方法

### 💾 方式1: 预编译二进制文件

从我们的[发布页面](https://github.com/yosebyte/nodepass/releases)下载适合您平台的最新版本。

### 🔧 方式2: 使用Go安装

```bash
go install github.com/yosebyte/nodepass/cmd/nodepass@latest
```

### 🛠️ 方式3: 从源代码构建

```bash
# 克隆仓库
git clone https://github.com/yosebyte/nodepass.git

# 构建二进制文件
cd nodepass
go build -o nodepass ./cmd/nodepass

# 可选: 安装到GOPATH/bin
go install ./cmd/nodepass
```

### 🐳 方式4: 使用容器镜像

NodePass在GitHub容器注册表中提供容器镜像:

```bash
# 拉取容器镜像
docker pull ghcr.io/yosebyte/nodepass:latest

# 服务器模式运行
docker run -d --name nodepass-server -p 10101:10101 -p 8080:8080 \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/0.0.0.0:8080

# 客户端模式运行
docker run -d --name nodepass-client \
  -e MIN_POOL_CAPACITY=32 \
  -e MAX_POOL_CAPACITY=512 \
  -p 8080:8080 \
  ghcr.io/yosebyte/nodepass client://server.example.com:10101/127.0.0.1:8080
```

### 📜 方式5: 使用管理脚本

对于Linux系统，您可以使用我们的交互式管理脚本进行简单的安装和服务管理：

```bash
bash <(curl -sL https://cdn.yobc.de/shell/nodepass.sh)
```

此脚本提供了一个交互式菜单，可以：
- 安装或更新NodePass
- 创建和配置多个nodepass服务
- 管理（启动/停止/重启/删除）nodepass服务
- 自动设置systemd服务
- 使用可自定义选项配置客户端和服务器模式

## 🚀 使用方法

NodePass创建一个带有未加密TCP控制通道的隧道，并为数据交换通道提供可配置的TLS加密选项。它有两种互补的运行模式：

### 🖥️ 服务器模式

```bash
nodepass server://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

- `tunnel_addr`: TCP隧道端点地址（控制通道），客户端将连接到此处(例如, 10.1.0.1:10101)
- `target_addr`: 服务器监听传入连接(TCP和UDP)的地址，这些连接将被隧道传输到客户端(例如, 10.1.0.1:8080)
- `log`: 日志级别(debug, info, warn, error, fatal)
- `tls`: 目标数据通道的TLS加密模式 (0, 1, 2)
  - `0`: 无TLS加密（明文TCP/UDP）
  - `1`: 自签名证书（自动生成）
  - `2`: 自定义证书（需要`crt`和`key`参数）
- `crt`: 证书文件路径（当`tls=2`时必需）
- `key`: 私钥文件路径（当`tls=2`时必需）

在服务器模式下，NodePass:
1. 在`tunnel_addr`上监听TCP隧道连接（控制通道）
2. 在`target_addr`上监听传入的TCP和UDP流量
3. 当`target_addr`收到连接时，通过未加密的TCP隧道向客户端发送信号
4. 为每个连接创建具有指定TLS加密级别的数据通道

示例:
```bash
# 数据通道无TLS加密
nodepass server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=0

# 自签名证书（自动生成）
nodepass server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=1

# 自定义域名证书
nodepass server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem
```

### 📱 客户端模式

```bash
nodepass client://<tunnel_addr>/<target_addr>?log=<level>
```

- `tunnel_addr`: 要连接的NodePass服务器隧道端点地址(例如, 10.1.0.1:10101)
- `target_addr`: 流量将被转发到的本地地址(例如, 127.0.0.1:8080)
- `log`: 日志级别(debug, info, warn, error, fatal)

在客户端模式下，NodePass:
1. 连接到服务器的未加密TCP隧道端点（控制通道）
2. 通过此控制通道监听来自服务器的信号
3. 当收到信号时，使用服务器指定的TLS安全级别建立数据连接
4. 在`target_addr`建立本地连接并转发流量

示例:
```bash
# 连接到NodePass服务器并自动采用其TLS安全策略
nodepass client://10.1.0.1:10101/127.0.0.1:8080?log=info
```

## ⚙️ 配置选项

NodePass采用命令行参数和环境变量的极简方法:

### 📝 日志级别

- `debug`: 详细调试信息 - 显示所有操作和连接
- `info`: 一般操作信息(默认) - 显示启动、关闭和关键事件
- `warn`: 警告条件 - 仅显示不影响核心功能的潜在问题
- `error`: 错误条件 - 仅显示影响功能的问题
- `fatal`: 致命条件 - 仅显示导致终止的严重错误

### 🔧 环境变量

| 变量 | 描述 | 默认值 | 示例 |
|----------|-------------|---------|---------|
| `SEMAPHORE_LIMIT` | 最大并发连接数 | 1024 | `export SEMAPHORE_LIMIT=2048` |
| `MIN_POOL_CAPACITY` | 最小连接池大小 | 16 | `export MIN_POOL_CAPACITY=32` |
| `MAX_POOL_CAPACITY` | 最大连接池大小 | 1024 | `export MAX_POOL_CAPACITY=4096` |
| `UDP_DATA_BUF_SIZE` | UDP数据包缓冲区大小 | 8192 | `export UDP_DATA_BUF_SIZE=16384` |
| `UDP_READ_TIMEOUT` | UDP读取操作超时 | 5s | `export UDP_READ_TIMEOUT=10s` |
| `REPORT_INTERVAL` | 健康检查报告间隔 | 5s | `export REPORT_INTERVAL=10s` |
| `SERVICE_COOLDOWN` | 重启尝试前的冷却期 | 5s | `export SERVICE_COOLDOWN=3s` |
| `SHUTDOWN_TIMEOUT` | 优雅关闭超时 | 5s | `export SHUTDOWN_TIMEOUT=10s` |

## 📚 使用示例

### 🔐 基本服务器设置与TLS选项

```bash
# 启动一个数据通道无TLS加密的服务器
nodepass server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=0

# 启动一个使用自动生成的自签名证书的服务器
nodepass server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1

# 启动一个使用自定义域名证书的服务器
nodepass server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem
```

### 🔌 连接到NodePass服务器

```bash
# 连接到服务器（自动采用服务器的TLS安全策略）
nodepass client://server.example.com:10101/127.0.0.1:8080

# 使用调试日志连接
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### 🗄 通过防火墙访问数据库

```bash
# 服务器端(位于安全网络外部)使用TLS加密
nodepass server://:10101/127.0.0.1:5432?tls=1

# 客户端(位于防火墙内部)
nodepass client://server.example.com:10101/127.0.0.1:5432
```

### 🔒 安全的微服务通信

```bash
# 服务A(提供API)使用自定义证书
nodepass server://0.0.0.0:10101/127.0.0.1:8081?log=warn&tls=2&crt=/path/to/service-a.crt&key=/path/to/service-a.key

# 服务B(消费API)
nodepass client://service-a:10101/127.0.0.1:8082
```

### 📡 物联网设备管理

```bash
# 中央管理服务器
nodepass server://0.0.0.0:10101/127.0.0.1:8888?log=info&tls=1

# 物联网设备
nodepass client://mgmt.example.com:10101/127.0.0.1:80
```

### 🧪 多环境开发

```bash
# 生产API访问隧道
nodepass client://tunnel.example.com:10101/127.0.0.1:3443

# 开发环境
nodepass server://tunnel.example.com:10101/127.0.0.1:3000

# 测试环境
nodepass server://tunnel.example.com:10101/127.0.0.1:3001?log=warn&tls=1
```

### 🐳 容器部署

```bash
# 为容器创建网络
docker network create nodepass-net

# 部署使用自签名证书的NodePass服务器
docker run -d --name nodepass-server \
  --network nodepass-net \
  -p 10101:10101 \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/web-service:80?log=info&tls=1

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

## 🔍 工作原理

NodePass创建具有独立控制和数据通道的网络架构:

1. **控制通道(隧道)**:
   - 客户端和服务器之间的未加密TCP连接
   - 专用于信号传递和协调
   - 在隧道的整个生命周期内保持持久连接

2. **数据通道(目标)**:
   - 可配置的TLS加密选项:
     - **模式0**: 未加密数据传输(最快，最不安全)
     - **模式1**: 自签名证书加密(安全性良好，无验证)
     - **模式2**: 验证证书加密(最高安全性，需要有效证书)
   - 按需为每个连接或数据报创建
   - 用于实际应用数据传输

3. **服务器模式操作**:
   - 在隧道端点监听控制连接
   - 当流量到达目标端点时，通过控制通道向客户端发送信号
   - 在需要时使用指定的TLS模式建立数据通道

4. **客户端模式操作**:
   - 连接到服务器的控制通道
   - 监听表示传入连接的信号
   - 使用服务器指定的TLS安全级别创建数据连接
   - 在安全通道和本地目标之间转发数据

5. **协议支持**:
   - **TCP**: 具有持久连接的完整双向流
   - **UDP**: 具有可配置缓冲区大小和超时的数据报转发

## 🔄 数据传输流程

NodePass通过其隧道架构建立双向数据流，支持TCP和UDP协议:

### 服务器端流程
1. **连接初始化**:
   ```
   [目标客户端] → [目标监听器] → [服务器: 创建目标连接]
   ```
   - 对于TCP: 客户端建立与目标监听器的持久连接
   - 对于UDP: 服务器在绑定到目标地址的UDP套接字上接收数据报

2. **信号生成**:
   ```
   [服务器] → [生成唯一连接ID] → [通过TLS加密隧道向客户端发送信号]
   ```
   - 对于TCP: 生成`tcp://<connection_id>`信号
   - 对于UDP: 当接收到数据报时生成`udp://<connection_id>`信号

3. **连接准备**:
   ```
   [服务器] → [在池中创建未加密的远程连接] → [等待客户端连接]
   ```
   - 两种协议使用相同的连接池机制，具有唯一的连接ID

4. **数据交换**:
   ```
   [目标连接] ⟷ [交换/传输] ⟷ [远程连接(未加密)]
   ```
   - 对于TCP: 使用`conn.DataExchange()`进行持续的双向数据流
   - 对于UDP: 使用可配置的缓冲区大小转发单个数据报

### 客户端流程
1. **信号接收**:
   ```
   [客户端] → [从TLS加密隧道读取信号] → [解析连接ID]
   ```
   - 客户端根据URL方案区分TCP和UDP信号

2. **连接建立**:
   ```
   [客户端] → [从池中检索连接] → [连接到远程端点(未加密)]
   ```
   - 此阶段的连接管理与协议无关

3. **本地连接**:
   ```
   [客户端] → [连接到本地目标] → [建立本地连接]
   ```
   - 对于TCP: 建立与本地目标的持久TCP连接
   - 对于UDP: 创建用于与本地目标进行数据报交换的UDP套接字

4. **数据交换**:
   ```
   [远程连接(未加密)] ⟷ [交换/传输] ⟷ [本地目标连接]
   ```
   - 对于TCP: 使用`conn.DataExchange()`进行持续的双向数据流
   - 对于UDP: 读取单个数据报，转发它，使用超时等待响应，然后返回响应

### 协议特性
- **TCP交换**: 
  - 用于全双工通信的持久连接
  - 持续数据流直到连接终止
  - 错误处理及自动重连

- **UDP交换**:
  - 一次性数据报转发，具有可配置的缓冲区大小(`UDP_DATA_BUF_SIZE`)
  - 响应等待的读取超时控制(`UDP_READ_TIMEOUT`)
  - 为低延迟、无状态通信优化

两种协议都受益于通过TLS隧道的相同安全信号机制，确保协议无关的控制流与协议特定的数据处理。

## 📡 信号通信机制

NodePass通过TLS隧道使用基于URL的复杂信号协议:

### 信号类型
1. **远程信号**:
   - 格式: `remote://<port>`
   - 目的: 通知客户端关于服务器的远程端点端口
   - 时机: 在健康检查期间定期发送

2. **TCP启动信号**:
   - 格式: `tcp://<connection_id>`
   - 目的: 请求客户端为特定ID建立TCP连接
   - 时机: 当收到新的TCP目标服务连接时发送

3. **UDP启动信号**:
   - 格式: `udp://<connection_id>`
   - 目的: 请求客户端处理特定ID的UDP流量
   - 时机: 当在目标端口接收到UDP数据时发送

### 信号流程
1. **信号生成**:
   - 服务器为特定事件创建URL格式的信号
   - 信号以换行符终止以便正确解析

2. **信号传输**:
   - 服务器将信号写入TLS隧道连接
   - 使用互斥锁防止并发写入隧道

3. **信号接收**:
   - 客户端使用缓冲读取器从隧道读取信号
   - 信号被修剪并解析为URL格式

4. **信号处理**:
   - 客户端将有效信号放入缓冲通道(signalChan)
   - 专用goroutine处理通道中的信号
   - 信号量模式防止信号溢出

5. **信号执行**:
   - 远程信号更新客户端的远程地址配置
   - 启动信号触发`clientOnce()`方法建立连接

### 信号弹性
- 具有可配置容量的缓冲通道防止高负载期间信号丢失
- 信号量实现确保受控并发
- 错误处理用于格式错误或意外信号

## 🔌 连接池架构

NodePass实现高效的连接池系统来管理网络连接:

### 池设计
1. **池类型**:
   - **客户端池**: 预先建立到远程端点的连接
   - **服务器池**: 管理来自客户端的传入连接

2. **池组件**:
   - **连接存储**: 线程安全的连接ID到net.Conn对象的映射
   - **ID通道**: 可用连接ID的缓冲通道
   - **容量管理**: 基于使用模式的动态调整
   - **连接工厂**: 可定制的连接创建函数

### 连接生命周期
1. **连接创建**:
   - 创建连接直到配置的容量
   - 每个连接分配唯一ID
   - ID和连接存储在池中

2. **连接获取**:
   - 客户端使用连接ID检索连接
   - 服务器从池中检索下一个可用连接
   - 返回前验证连接

3. **连接使用**:
   - 获取时从池中移除连接
   - 用于端点之间的数据交换
   - 不重用连接(一次性使用模型)

4. **连接终止**:
   - 使用后关闭连接
   - 适当释放资源
   - 错误处理确保清洁终止

### 池管理
1. **容量控制**:
   - `MIN_POOL_CAPACITY`: 确保最小可用连接
   - `MAX_POOL_CAPACITY`: 防止过度资源消耗
   - 基于需求模式的动态缩放

2. **池管理器**:
   - `ClientManager()`: 维护客户端连接池
   - `ServerManager()`: 管理服务器连接池

3. **一次性连接模式**:
   池中的每个连接遵循一次性使用模式:
   - 创建并放入池中
   - 为特定数据交换检索一次
   - 永不返回池(防止潜在数据泄漏)
   - 使用后适当关闭

4. **自动池大小调整**:
   - 池容量根据实时使用模式动态调整
   - 如果连接创建成功率低(<20%)，容量减少以最小化资源浪费
   - 如果连接创建成功率高(>80%)，容量增加以适应更高流量
   - 渐进缩放防止振荡并提供稳定性
   - 尊重配置的最小和最大容量边界
   - 在低活动期间缩小规模以节省资源
   - 流量增加时主动扩展以维持性能
   - 适应不同网络条件的自调节算法
   - 为客户端和服务器池提供单独的调整逻辑以优化不同流量模式

5. **效率考虑**:
   - 预先建立减少连接延迟
   - 连接验证确保只使用健康连接
   - 适当的资源清理防止连接泄漏
   - 基于间隔的池维护平衡资源使用与响应能力
   - 具有最小开销的优化连接验证

## 💡 常见使用场景

- **🚪 远程访问**: 从外部位置访问私有网络上的服务，无需VPN基础设施。适用于从远程工作环境访问开发服务器、内部工具或监控系统。

- **🧱 防火墙绕过**: 通过建立使用常允许端口(如443)的隧道，在限制性网络环境中导航。适合具有严格出站连接策略的企业环境或连接有限的公共Wi-Fi网络。

- **🏛️ 遗留系统集成**: 安全连接现代应用程序到遗留系统，无需修改遗留基础设施。通过在旧应用组件和新应用组件之间提供安全桥梁，实现渐进现代化策略。

- **🔒 安全微服务通信**: 在不同网络或数据中心的分布式组件之间建立加密通道。允许微服务安全通信，即使在公共网络上，无需实现复杂的服务网格解决方案。

- **📱 远程开发**: 从任何地方连接到开发资源，实现无缝编码、测试和调试内部开发环境，无论开发人员位置如何。支持现代分布式团队工作流和远程工作安排。

- **☁️ 云到本地连接**: 无需将内部系统直接暴露给互联网，即可将云服务与本地基础设施连接起来。为需要环境之间保护通信通道的混合云架构创建安全桥梁。

- **🌍 地理分布**: 从不同位置访问特定区域的服务，克服地理限制或测试区域特定功能。对于需要在不同市场一致运行的全球应用程序非常有用。

- **🧪 测试环境**: 创建到隔离测试环境的安全连接，而不影响其隔离性。使QA团队能够安全访问测试系统，同时维护测试数据和配置的完整性。

- **🔄 API网关替代**: 作为特定服务的轻量级API网关替代方案。提供对内部API的安全访问，而无需全面API管理解决方案的复杂性和开销。

- **🔒 数据库保护**: 启用安全数据库访问，同时使数据库服务器完全隔离，免受直接互联网暴露。创建一个安全中间层，保护宝贵的数据资产免受直接网络攻击。

- **🌐 跨网络物联网通信**: 促进部署在不同网络段的物联网设备之间的通信。克服物联网部署中常见的NAT、防火墙和路由挑战，跨多个位置。

- **🛠️ DevOps管道集成**: 将CI/CD管道安全连接到各种环境中的部署目标。确保构建和部署系统可以安全地到达生产、暂存和测试环境，而不影响网络安全。

## 🔧 故障排除

### 📜 连接问题
- 验证防火墙设置允许指定端口上的TCP和UDP流量
- 检查客户端模式下隧道地址是否正确指定
- 确保TLS证书生成正确
- 增加日志级别到debug以获取更详细的连接信息
- 验证客户端和服务器端点之间的网络稳定性
- 对于UDP隧道问题，检查您的应用程序是否需要特定的UDP数据包大小配置
- 对于高容量UDP应用，考虑增加UDP_DATA_BUF_SIZE
- 如果UDP数据包似乎丢失，尝试调整UDP_READ_TIMEOUT值
- 如果在不同网络间运行，检查NAT穿越问题
- 如果在负载下遇到连接失败，检查系统资源限制(文件描述符等)
- 如果使用主机名作为隧道或目标地址，验证DNS解析

### 🚀 性能优化

#### 连接池调优
- 根据预期的最小并发连接调整`MIN_POOL_CAPACITY`
  - 太低: 流量高峰期延迟增加，因为必须建立新连接
  - 太高: 维护空闲连接浪费资源
  - 推荐起点: 平均并发连接的25-50%

- 配置`MAX_POOL_CAPACITY`以处理峰值负载，同时防止资源耗尽
  - 太低: 流量高峰期连接失败
  - 太高: 潜在资源耗尽影响系统稳定性
  - 推荐起点: 峰值并发连接的150-200%

- 根据预期峰值并发隧道会话设置`SEMAPHORE_LIMIT`
  - 太低: 流量高峰期拒绝连接
  - 太高: 太多并发goroutine可能导致内存压力
  - 推荐范围: 大多数应用1000-5000，高吞吐量场景更高

#### 网络配置
- 优化客户端和服务器上的TCP设置:
  - 调整长寿命连接的TCP保活时间间隔
  - 考虑高吞吐量应用的TCP缓冲区大小
  - 如可用，启用TCP BBR拥塞控制算法

#### 资源分配
- 确保客户端和服务器上有足够的系统资源:
  - 监控峰值负载期间的CPU使用率
  - 跟踪连接管理的内存消耗
  - 验证端点之间有足够的网络带宽

#### 监控建议
- 实现连接跟踪以识别瓶颈
- 监控连接建立成功率
- 跟踪数据传输率以识别吞吐量问题
- 测量连接延迟以优化用户体验

#### 高级场景
- 对于高吞吐量应用:
  ```bash
  export MIN_POOL_CAPACITY=64
  export MAX_POOL_CAPACITY=4096
  export SEMAPHORE_LIMIT=8192
  export REPORT_INTERVAL=2s
  ```

- 对于低延迟应用:
  ```bash
  export MIN_POOL_CAPACITY=32
  export MAX_POOL_CAPACITY=1024
  export SEMAPHORE_LIMIT=2048
  export REPORT_INTERVAL=1s
  ```

- 对于资源受限环境:
  ```bash
  export MIN_POOL_CAPACITY=8
  export MAX_POOL_CAPACITY=256
  export SEMAPHORE_LIMIT=512
  export REPORT_INTERVAL=10s
  ```

## 👥 贡献指南

欢迎贡献！请随时提交Pull Request。

1. Fork仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开Pull Request

## 💬 社区讨论

感谢[NodeSeek](https://www.nodeseek.com/post-295115-1)社区各位开发者和用户的意见反馈，有任何技术问题欢迎随时交流。

## 📄 许可协议

本项目根据MIT许可证授权 - 详见[LICENSE](LICENSE)文件。

## ⭐ Star趋势

[![Stargazers over time](https://starchart.cc/yosebyte/nodepass.svg?variant=adaptive)](https://starchart.cc/yosebyte/nodepass)
