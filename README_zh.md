<div align="center">
  <img src="https://cdn.yobc.de/assets/np-poster.png" alt="nodepass" width="500">

[![GitHub release](https://img.shields.io/github/v/release/yosebyte/nodepass)](https://github.com/yosebyte/nodepass/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/yosebyte/nodepass)](https://goreportcard.com/report/github.com/yosebyte/nodepass)
![GitHub last commit](https://img.shields.io/github/last-commit/yosebyte/nodepass)

[English](README.md) | [简体中文](README_zh.md)
</div>

NodePass是一个优雅、高效的TCP隧道解决方案，可在网络端点之间创建安全的通信桥梁。通过建立一个TCP控制通道，NodePass能够在受限网络环境中实现无缝数据传输，同时为数据通道提供可配置的安全选项。其服务器-客户端架构允许灵活部署，使服务能够穿越防火墙、NAT和其他网络障碍。凭借智能连接池、最小资源占用和简洁的命令语法，NodePass为开发人员和系统管理员提供了一个强大且易用的工具，可以解决复杂的网络挑战，同时不影响安全性或性能。

## 📋 核心功能

- **🔀 多种操作模式**
  - 服务器模式接受传入隧道连接并提供可配置的安全选项
  - 客户端模式用于建立与隧道服务器的出站连接
  - 主控模式提供RESTful API进行动态实例管理

- **🌍 协议支持**
  - TCP隧道传输与持久连接管理
  - UDP数据报转发与可配置的缓冲区大小
  - 两种协议的智能路由机制

- **🛡️ 安全选项**
  - TLS模式0：在可信网络中获得最大速度的无加密模式
  - TLS模式1：使用自签名证书提供快速安全设置
  - TLS模式2：使用自定义证书验证实现企业级安全

- **⚡ 性能特性**
  - 智能连接池，具备实时容量自适应功能
  - 基于网络状况的动态间隔调整
  - 高负载下保持最小资源占用
  - 网络中断后自动恢复连接

- **🧰 简单配置**
  - 零配置文件设计
  - 简洁的命令行参数
  - 环境变量支持性能精细调优
  - 为大多数使用场景提供智能默认值

## 📚 文档

探索完整文档以了解更多关于NodePass的信息：

- [安装指南](/docs/zh/installation.md)
- [使用说明](/docs/zh/usage.md)
- [配置选项](/docs/zh/configuration.md)
- [API参考](/docs/zh/api.md)
- [使用示例](/docs/zh/examples.md)
- [工作原理](/docs/zh/how-it-works.md)
- [故障排除](/docs/zh/troubleshooting.md)

## 📋 快速开始

### 📥 安装方法

- **预编译二进制文件**: 从[发布页面](https://github.com/yosebyte/nodepass/releases)下载。
- **Go安装**: `go install github.com/yosebyte/nodepass/cmd/nodepass@latest`
- **容器镜像**: `docker pull ghcr.io/yosebyte/nodepass:latest`
- **Linux管理脚本**: `bash <(curl -sL https://cdn.yobc.de/shell/nodepass.sh)`

### 🚀 基本用法

**服务器模式**
```bash
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"
```

**客户端模式**
```bash
nodepass client://server.example.com:10101/127.0.0.1:8080
```

**主控模式 (API)**
```bash
nodepass "master://0.0.0.0:10101/api?log=debug&tls=1"
```

## 🔧 常见使用场景

- **远程访问**: 从外部位置安全访问内部服务
- **防火墙绕过**: 在限制性网络环境中导航
- **安全微服务**: 在分布式组件之间建立加密通道
- **数据库保护**: 在保持服务器隔离的同时实现安全数据库访问
- **物联网通信**: 连接不同网络段上的设备

## 👥 贡献指南

欢迎贡献！请随时提交Pull Request。

1. Fork仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开Pull Request

## 📄 许可协议

本项目根据MIT许可证授权 - 详见[LICENSE](LICENSE)文件。

## ⭐ Star趋势

[![Stargazers over time](https://starchart.cc/yosebyte/nodepass.svg?variant=adaptive)](https://starchart.cc/yosebyte/nodepass)
