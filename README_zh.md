# 🔗 NodePass - 增强版

[![GitHub release](https://img.shields.io/github/v/release/yosebyte/nodepass)](https://github.com/yosebyte/nodepass/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yosebyte/nodepass)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/yosebyte/nodepass)](https://goreportcard.com/report/github.com/yosebyte/nodepass)

<div align="center">
  <img src="https://cdn.yobc.de/assets/nodepass.png" alt="nodepass">
</div>

**Language**: [English](README.md) | [简体中文](README_zh.md)

NodePass是一个优雅、高效的TCP隧道解决方案，可在网络端点之间创建安全的通信桥梁。通过建立非加密的TCP控制通道，NodePass促进了数据在受限网络环境中的无缝传输，同时为数据通道提供可配置的安全选项。其服务器-客户端架构允许灵活的部署场景，使服务能够穿越防火墙、NAT和其他网络障碍。凭借智能连接池、最小资源占用和简洁的命令语法，NodePass为开发人员和系统管理员提供了一个强大且易于使用的工具，用于解决复杂的网络挑战，同时不影响安全性或性能。

## 📋 目录

- [功能特性](#-功能特性)
- [系统要求](#-系统要求)
- [安装方法](#-安装方法)
  - [方法1：预编译二进制文件](#-方法1预编译二进制文件)
  - [方法2：使用Go Install](#-方法2使用go-install)
  - [方法3：从源代码构建](#️-方法3从源代码构建)
  - [方法4：使用容器镜像](#-方法4使用容器镜像)
  - [方法5：使用管理脚本](#-方法5使用管理脚本)
- [使用方法](#-使用方法)
  - [服务器模式](#️-服务器模式)
  - [客户端模式](#-客户端模式)
  - [协议选择](#-协议选择)
- [配置选项](#️-配置选项)
  - [日志级别](#-日志级别)
  - [环境变量](#-环境变量)
- [示例](#-示例)
  - [基本服务器设置与TLS选项](#-基本服务器设置与tls选项)
  - [连接到NodePass服务器](#-连接到nodepass服务器)
  - [使用QUIC协议](#-使用quic协议)
  - [使用WebSocket连接](#-使用websocket连接)
  - [通过防火墙访问数据库](#-通过防火墙访问数据库)
  - [安全的微服务通信](#-安全的微服务通信)
  - [IoT设备管理](#-iot设备管理)
  - [多环境开发](#-多环境开发)
  - [容器部署](#-容器部署)
- [工作原理](#-工作原理)
- [数据传输流程](#-数据传输流程)
- [信号通信机制](#-信号通信机制)
- [连接池架构](#-连接池架构)
- [常见用例](#-常见用例)
- [故障排除](#-故障排除)
  - [连接问题](#-连接问题)
  - [性能优化](#-性能优化)
- [贡献](#-贡献)
- [讨论](#-讨论)
- [许可证](#-许可证)
- [Star历史](#-star历史)

## ✨ 功能特性

<div align="center">
  <img src="https://cdn.yobc.de/assets/np-cli.png" alt="nodepass">
</div>

- **🔄 双操作模式**：可作为服务器接受连接或作为客户端发起连接
- **🌐 多协议支持**：支持TCP/UDP/QUIC/WebSocket协议，提供完整的应用兼容性
- **🔒 灵活的TLS选项**：三种数据通道加密安全模式
- **🔐 强制TLS1.3加密**：所有加密连接均使用TLS1.3，提供最高级别的安全性
- **🚀 QUIC协议支持**：基于UDP的QUIC协议，提供低延迟和更好的多路复用能力
- **🌐 WebSocket全双工连接**：支持基于HTTP的WebSocket连接，适用于防火墙限制环境
- **🔌 高效连接池**：优化的连接管理，具有可配置的池大小
- **📊 灵活的日志系统**：五种不同日志级别的可配置详细程度
- **🛡️ 弹性错误处理**：自动连接恢复和优雅关闭
- **📦 单二进制部署**：简单分发和安装，依赖项最少
- **⚙️ 零配置文件**：所有配置通过命令行参数和环境变量指定
- **🚀 低资源占用**：即使在高负载下也能保持最小的CPU和内存使用
- **♻️ 自动重连**：从网络中断中无缝恢复
- **🧩 模块化架构**：客户端、服务器和公共组件之间的清晰分离
- **🔍 全面调试**：详细的连接跟踪和信号监控
- **⚡ 高性能数据交换**：优化的双向数据传输机制
- **🧠 智能连接管理**：智能处理连接状态和生命周期
- **📈 可扩展信号量系统**：防止高流量期间资源耗尽
- **🛠️ 可配置池动态**：根据工作负载调整连接池行为
- **🔌 一次性连接模式**：通过非重用连接增强安全性
- **📡 动态端口分配**：自动管理安全通信的端口分配

## 📋 系统要求

- Go 1.18或更高版本（从源代码构建时需要）
- 服务器和客户端端点之间的网络连接
- 绑定到1024以下端口可能需要管理员权限

## 📥 安装方法

### 💾 方法1：预编译二进制文件

从我们的[发布页面](https://github.com/yosebyte/nodepass/releases)下载适用于您平台的最新版本。

### 🔧 方法2：使用Go Install

```bash
go install github.com/yosebyte/nodepass/cmd/nodepass@latest
```

### 🛠️ 方法3：从源代码构建

```bash
# 克隆仓库
git clone https://github.com/yosebyte/nodepass.git

# 构建二进制文件
cd nodepass
go build -o nodepass ./cmd/nodepass

# 可选：安装到GOPATH/bin
go install ./cmd/nodepass
```

### 🐳 方法4：使用容器镜像

NodePass在GitHub容器注册表上可用作容器镜像：

```bash
# 拉取容器镜像
docker pull ghcr.io/yosebyte/nodepass:latest

# 以服务器模式运行
docker run -d --name nodepass-server -p 10101:10101 -p 8080:8080 \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/0.0.0.0:8080

# 以客户端模式运行
docker run -d --name nodepass-client \
  -e MIN_POOL_CAPACITY=32 \
  -e MAX_POOL_CAPACITY=512 \
  -p 8080:8080 \
  ghcr.io/yosebyte/nodepass client://nodepass-server:10101/127.0.0.1:8080
```

### 📜 方法5：使用管理脚本

对于Linux系统，您可以使用我们的交互式管理脚本进行简单安装和服务管理：

```bash
bash <(curl -sL https://cdn.yobc.de/shell/nodepass.sh)
```

此脚本提供交互式菜单，可以：
- 安装或更新NodePass
- 创建和配置多个nodepass服务
- 管理（启动/停止/重启/删除）nodepass服务
- 自动设置systemd服务
- 配置具有可自定义选项的客户端和服务器模式

## 🚀 使用方法

NodePass创建一个具有非加密TCP控制通道和可配置TLS加密选项的数据交换通道的隧道。它以两种互补模式运行：

### 🖥️ 服务器模式

```bash
nodepass server://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

- `tunnel_addr`：TCP隧道端点（控制通道）的地址，客户端将连接到该地址（例如，10.1.0.1:10101）
- `target_addr`：服务器监听传入连接（TCP和UDP）的地址，这些连接将被隧道传输到客户端（例如，10.1.0.1:8080）
- `log`：日志级别（debug, info, warn, error, fatal）
- `tls`：目标数据通道的TLS加密模式（0, 1, 2）
  - `0`：无TLS加密（纯TCP/UDP）
  - `1`：自签名证书（自动生成）
  - `2`：自定义证书（需要`crt`和`key`参数）
- `crt`：证书文件路径（当`tls=2`时需要）
- `key`：私钥文件路径（当`tls=2`时需要）

在服务器模式下，NodePass：
1. 在`tunnel_addr`上监听TCP隧道连接（控制通道）
2. 在`target_addr`上监听传入的TCP和UDP流量
3. 当连接到达`target_addr`时，通过非加密TCP隧道向已连接的客户端发送信号
4. 为每个连接创建具有指定TLS加密级别的数据通道

示例：
```bash
# 数据通道无TLS加密
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=0"

# 自签名证书（自动生成）
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=1"

# 自定义域证书
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### 📱 客户端模式

```bash
nodepass client://<tunnel_addr>/<target_addr>?log=<level>
```

- `tunnel_addr`：要连接的NodePass服务器的隧道端点地址（例如，10.1.0.1:10101）
- `target_addr`：流量将被转发到的本地地址（例如，127.0.0.1:8080）
- `log`：日志级别（debug, info, warn, error, fatal）

在客户端模式下，NodePass：
1. 连接到服务器的非加密TCP隧道端点（控制通道）`tunnel_addr`
2. 通过此控制通道监听来自服务器的信号
3. 当收到信号时，建立具有服务器指定的TLS安全级别的数据连接
4. 创建到`target_addr`的本地连接并转发流量

示例：
```bash
# 连接到NodePass服务器并自动采用其TLS安全策略
nodepass client://10.1.0.1:10101/127.0.0.1:8080?log=info
```

### 🌐 协议选择

NodePass现在支持多种协议类型，可以根据需要选择最适合的协议：

- **TCP**：默认协议，适用于大多数场景，提供可靠的连接
- **UDP**：适用于需要低延迟的应用，如实时通信和游戏
- **QUIC**：基于UDP的现代协议，提供低延迟和更好的多路复用能力，特别适合移动网络
- **WebSocket**：基于HTTP的协议，适用于需要穿越防火墙的场景，支持全双工通信

服务器会自动通知客户端其支持的协议类型，客户端会自动选择最佳协议进行通信。

## ⚙️ 配置选项

NodePass使用命令行参数和环境变量的极简方法：

### 📝 日志级别

- `debug`：详细调试信息 - 显示所有操作和连接
- `info`：一般操作信息（默认）- 显示启动、关闭和关键事件
- `warn`：警告条件 - 仅显示不影响核心功能的潜在问题
- `error`：错误条件 - 仅显示影响功能的问题
- `fatal`：严重条件 - 仅显示导致终止的严重错误

### 🔧 环境变量

| 变量 | 描述 | 默认值 | 示例 |
|----------|-------------|---------|---------|
| `SEMAPHORE_LIMIT` | 最大并发连接数 | 1024 | `export SEMAPHORE_LIMIT=2048` |
| `MIN_POOL_CAPACITY` | 最小连接池大小 | 16 | `export MIN_POOL_CAPACITY=32` |
| `MAX_POOL_CAPACITY` | 最大连接池大小 | 1024 | `export MAX_POOL_CAPACITY=4096` |
| `UDP_DATA_BUF_SIZE` | UDP数据包的缓冲区大小 | 8192 | `export UDP_DATA_BUF_SIZE=16384` |
| `UDP_READ_TIMEOUT` | UDP读取操作的超时 | 5s | `export UDP_READ_TIMEOUT=10s` |
| `REPORT_INTERVAL` | 健康检查报告的间隔 | 5s | `export REPORT_INTERVAL=10s` |
| `RELOAD_INTERVAL` | 证书重新加载的间隔 | 1h | `export RELOAD_INTERVAL=30m` |
| `SERVICE_COOLDOWN` | 重启尝试前的冷却期 | 5s | `export SERVICE_COOLDOWN=3s` |
| `SHUTDOWN_TIMEOUT` | 优雅关闭的超时 | 5s | `export SHUTDOWN_TIMEOUT=10s` |

## 📚 示例

### 🔐 基本服务器设置与TLS选项

```bash
# 启动数据通道无TLS加密的服务器
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=0"

# 启动具有自动生成的自签名证书的服务器（使用TLS1.3）
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# 启动具有自定义域证书的服务器（使用TLS1.3）
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### 🔌 连接到NodePass服务器

```bash
# 连接到服务器（自动采用服务器的TLS安全策略）
nodepass client://server.example.com:10101/127.0.0.1:8080

# 使用调试日志连接
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### 🚀 使用QUIC协议

服务器会自动启用QUIC协议支持，客户端会自动检测并使用QUIC协议（如果可用）。QUIC协议提供更低的延迟和更好的多路复用能力，特别适合移动网络和不稳定的网络环境。

```bash
# 服务器端（自动启用QUIC支持）
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# 客户端（自动检测并使用QUIC）
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### 🌐 使用WebSocket连接

WebSocket连接特别适合需要穿越防火墙的场景，因为它基于HTTP协议，大多数防火墙允许HTTP流量。

```bash
# 服务器端（自动启用WebSocket支持）
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# 客户端（自动检测并使用WebSocket）
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### 🗄 通过防火墙访问数据库

```bash
# 服务器端（在安全网络外部）使用TLS1.3加密
nodepass server://:10101/127.0.0.1:5432?tls=1

# 客户端端（在防火墙内部）
nodepass client://server.example.com:10101/127.0.0.1:5432
```

### 🔒 安全的微服务通信

```bash
# 服务A（提供API）使用自定义证书和TLS1.3
nodepass "server://0.0.0.0:10101/127.0.0.1:8081?log=warn&tls=2&crt=/path/to/service-a.crt&key=/path/to/service-a.key"

# 服务B（消费API）
nodepass client://service-a:10101/127.0.0.1:8082
```

### 📡 IoT设备管理

```bash
# 中央管理服务器（使用TLS1.3和WebSocket支持）
nodepass "server://0.0.0.0:10101/127.0.0.1:8888?log=info&tls=1"

# IoT设备（自动使用最佳可用协议）
nodepass client://mgmt.example.com:10101/127.0.0.1:80
```

### 🧪 多环境开发

```bash
# 生产API访问隧道
nodepass client://tunnel.example.com:10101/127.0.0.1:3443

# 开发环境
nodepass server://tunnel.example.com:10101/127.0.0.1:3000

# 测试环境（使用TLS1.3）
nodepass "server://tunnel.example.com:10101/127.0.0.1:3001?log=warn&tls=1"
```

### 🐳 容器部署

```bash
# 为容器创建网络
docker network create nodepass-net

# 部署带有自签名证书的NodePass服务器（使用TLS1.3）
docker run -d --name nodepass-server \
  --network nodepass-net \
  -p 10101:10101 \
  ghcr.io/yosebyte/nodepass "server://0.0.0.0:10101/web-service:80?log=info&tls=1"

# 部署作为目标的Web服务
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

NodePass创建了一个具有单独控制和数据通道的网络架构：

1. **控制通道（隧道）**：
   - 客户端和服务器之间的非加密TCP连接
   - 专门用于信号和协调
   - 在隧道的整个生命周期内保持持久连接

2. **数据通道（目标）**：
   - 可配置的TLS加密选项：
     - **模式0**：非加密数据传输（最快，最不安全）
     - **模式1**：自签名证书加密（良好的安全性，无验证）
     - **模式2**：验证证书加密（最高安全性，需要有效证书）
   - 所有TLS加密连接均使用TLS1.3，提供最高级别的安全性
   - 根据需要为每个连接或数据报创建
   - 用于实际应用数据传输
   - 支持多种协议：TCP、UDP、QUIC和WebSocket

3. **服务器模式操作**：
   - 在隧道端点监听控制连接
   - 当流量到达目标端点时，通过控制通道向客户端发送信号
   - 在需要时使用指定的TLS模式建立数据通道
   - 自动启用QUIC和WebSocket支持，并通知客户端

4. **客户端模式操作**：
   - 连接到服务器的控制通道
   - 监听指示传入连接的信号
   - 使用服务器指定的TLS安全级别创建数据连接
   - 在本地目标和安全通道之间转发数据
   - 自动检测并使用最佳可用协议（TCP、QUIC或WebSocket）

5. **协议支持**：
   - **TCP**：具有持久连接的完整双向流
   - **UDP**：具有可配置缓冲区大小和超时的数据报转发
   - **QUIC**：基于UDP的现代协议，提供低延迟和更好的多路复用能力
   - **WebSocket**：基于HTTP的协议，支持全双工通信，适用于防火墙限制环境

## 🔄 数据传输流程

NodePass通过其隧道架构建立双向数据流，支持TCP、UDP、QUIC和WebSocket协议：

### 服务器端流程
1. **连接初始化**：
   ```
   [目标客户端] → [目标监听器] → [服务器：目标连接已创建]
   ```
   - 对于TCP：客户端与目标监听器建立持久连接
   - 对于UDP：服务器在绑定到目标地址的UDP套接字上接收数据报
   - 对于QUIC：服务器接受QUIC连接请求
   - 对于WebSocket：服务器接受WebSocket升级请求

2. **信号生成**：
   ```
   [服务器] → [生成唯一连接ID] → [通过非加密TCP隧道向客户端发送信号]
   ```
   - 对于TCP：生成`//<connection_id>#1`信号
   - 对于UDP：生成`//<connection_id>#2`信号
   - 对于QUIC：生成`//<connection_id>#3`信号
   - 对于WebSocket：生成`//<connection_id>#4`信号

3. **连接准备**：
   ```
   [服务器] → [在池中创建具有配置的TLS模式的远程连接] → [等待客户端连接]
   ```
   - 所有协议都使用具有唯一连接ID的相同连接池机制
   - 根据指定的模式（0、1或2）应用TLS配置
   - 所有TLS加密连接均使用TLS1.3

4. **数据交换**：
   ```
   [目标连接] ⟷ [交换/传输] ⟷ [远程连接]
   ```
   - 对于TCP和WebSocket：使用`conn.DataExchange()`进行连续的双向数据流
   - 对于UDP：单个数据报被转发
   - 对于QUIC：使用QUIC流进行高效的多路复用数据传输

### 客户端端流程
1. **信号接收**：
   ```
   [客户端] ← [通过非加密TCP隧道接收信号] ← [服务器]
   ```
   - 客户端解析信号以确定连接ID和协议类型

2. **连接建立**：
   ```
   [客户端] → [从池中获取远程连接] → [连接到本地目标]
   ```
   - 客户端从连接池中获取预先建立的连接
   - 根据协议类型（TCP、UDP、QUIC或WebSocket）使用适当的连接方法

3. **数据交换**：
   ```
   [远程连接] ⟷ [交换/传输] ⟷ [本地目标]
   ```
   - 使用与服务器端相同的数据交换机制
   - 所有协议都使用相同的接口进行数据传输

4. **连接终止**：
   ```
   [客户端] → [关闭连接] → [返回统计信息]
   ```
   - 当任一端关闭连接时，另一端也会关闭
   - 记录传输的字节数和连接持续时间

## 🔄 信号通信机制

NodePass使用简单但强大的信号系统在客户端和服务器之间协调：

1. **隧道建立**：
   ```
   [客户端] → [连接到服务器隧道端点] → [服务器]
   [客户端] ← [接收端口和TLS模式信息] ← [服务器]
   ```
   - 服务器发送包含远程端口和TLS模式的URL格式信号
   - 客户端解析信号并配置其连接参数

2. **连接信号**：
   ```
   [服务器] → [//<connection_id>#<protocol_type>] → [客户端]
   ```
   - `<connection_id>`是唯一标识符
   - `<protocol_type>`指示协议：1=TCP，2=UDP，3=QUIC，4=WebSocket

3. **健康检查**：
   ```
   [服务器] → [定期发送换行符] → [客户端]
   ```
   - 服务器定期发送换行符以验证隧道是否仍然活动
   - 如果检测到错误，两端都会尝试重新建立隧道

## 🔌 连接池架构

NodePass使用高效的连接池系统来优化性能和资源使用：

1. **服务器池**：
   - 预先分配的连接，等待客户端请求
   - 当目标连接到达时分配
   - 支持TCP、UDP、QUIC和WebSocket连接

2. **客户端池**：
   - 预先建立到服务器的连接
   - 根据信号分配
   - 动态调整大小以保持最佳性能

3. **池管理**：
   - 自动扩展和收缩以适应负载
   - 定期健康检查和连接刷新
   - 智能资源分配以防止耗尽

4. **连接生命周期**：
   ```
   [创建] → [池] → [分配] → [使用] → [关闭/返回]
   ```
   - 连接在使用前预先建立
   - 使用后关闭以确保安全
   - 池大小通过环境变量可配置

## 🔍 常见用例

NodePass适用于各种网络场景：

1. **防火墙穿透**：
   - 在防火墙后面访问服务
   - 通过单个开放端口提供多个服务

2. **安全远程访问**：
   - 使用TLS1.3加密的安全远程访问
   - 无需VPN的内部服务访问

3. **微服务通信**：
   - 跨网络边界的服务间通信
   - 使用QUIC协议的高效服务网格

4. **IoT设备连接**：
   - 远程设备管理和监控
   - 使用WebSocket的低带宽设备通信

5. **开发和测试**：
   - 本地开发环境到生产环境的安全桥接
   - 跨环境测试和调试

6. **数据库访问**：
   - 安全远程数据库连接
   - 跨网络的数据库复制

7. **API代理**：
   - 安全API网关
   - 跨域API访问

8. **容器网络**：
   - 跨主机容器通信
   - 容器到外部服务的桥接

## 🔧 故障排除

### 🔌 连接问题

1. **隧道建立失败**：
   - 检查网络连接和防火墙规则
   - 验证服务器是否正在运行并监听指定端口
   - 使用`log=debug`获取详细信息

2. **数据传输错误**：
   - 检查TLS配置
   - 验证目标服务是否可访问
   - 检查连接池大小和信号量限制

3. **性能问题**：
   - 调整连接池大小
   - 考虑使用QUIC协议以获得更好的性能
   - 监控资源使用情况

### 📊 性能优化

1. **连接池调整**：
   ```bash
   export MIN_POOL_CAPACITY=32
   export MAX_POOL_CAPACITY=2048
   ```
   - 增加最小池容量以提高响应能力
   - 增加最大池容量以处理更高的并发

2. **协议选择**：
   - 对于低延迟要求，使用QUIC协议
   - 对于防火墙穿透，使用WebSocket
   - 对于一般用途，使用标准TCP

3. **缓冲区大小**：
   ```bash
   export UDP_DATA_BUF_SIZE=16384
   ```
   - 增加UDP缓冲区大小以处理更大的数据报

4. **超时设置**：
   ```bash
   export UDP_READ_TIMEOUT=10s
   export SHUTDOWN_TIMEOUT=10s
   ```
   - 调整超时以适应网络条件

## 🤝 贡献

欢迎贡献！请随时提交问题、功能请求或拉取请求。

## 💬 讨论

加入我们的[讨论](https://github.com/yosebyte/nodepass/discussions)，分享您的经验和想法。

## 📄 许可证

本项目根据MIT许可证授权 - 有关详细信息，请参阅[LICENSE](LICENSE)文件。

## ⭐ Star历史

[![Star History Chart](https://api.star-history.com/svg?repos=yosebyte/nodepass&type=Date)](https://star-history.com/#yosebyte/nodepass&Date)
