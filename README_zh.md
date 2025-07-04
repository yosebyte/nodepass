<div align="center">
  <img src="https://cdn.yobc.de/assets/np-poster.png" alt="nodepass" width="448">

[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)
[![GitHub release](https://img.shields.io/github/v/release/yosebyte/nodepass)](https://github.com/yosebyte/nodepass/releases)
[![GitHub downloads](https://img.shields.io/github/downloads/yosebyte/nodepass/total.svg)](https://github.com/yosebyte/nodepass/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/yosebyte/nodepass)](https://goreportcard.com/report/github.com/yosebyte/nodepass)
[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![Go Reference](https://pkg.go.dev/badge/github.com/yosebyte/nodepass.svg)](https://pkg.go.dev/github.com/yosebyte/nodepass)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/yosebyte/nodepass)
![GitHub last commit](https://img.shields.io/github/last-commit/yosebyte/nodepass)

[English](README.md) | 简体中文
</div>

**NodePass**是一款企业级的TCP/UDP网络隧道工具，采用现代化三层架构实现控制与数据通道分离，具备灵活、高性能的实例管理方式。支持零配置部署，具备智能连接池、分级TLS加密、无缝协议转换等核心能力。专为DevOps和系统管理员设计，轻松应对防火墙穿透、NAT绕过、隧道互联等复杂网络场景。

## 💎 核心功能

- **🔀 多种操作模式**
  - 服务端模式接受传入隧道连接并提供可配置的安全选项
  - 客户端模式用于建立与隧道服务端的出站连接
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

- **🧰 简单配置**
  - 零配置文件设计
  - 简洁的命令行参数
  - 环境变量支持性能精细调优

## 📋 快速开始

### 📥 安装方法

- **预编译二进制文件**: 从[发布页面](https://github.com/yosebyte/nodepass/releases)下载。
- **容器镜像**: `docker pull ghcr.io/yosebyte/nodepass:latest`

### 🚀 基本用法

**服务端模式**
```bash
nodepass "server://:10101/127.0.0.1:8080?log=debug&tls=1"
```

**客户端模式**
```bash
nodepass "client://server:10101/127.0.0.1:8080?min=128"
```

**主控模式 (API)**
```bash
nodepass "master://:10101/api?log=debug&tls=1"
```

## 📚 文档

探索完整文档以了解更多关于NodePass的信息：

- [安装指南](/docs/zh/installation.md)
- [使用说明](/docs/zh/usage.md)
- [配置选项](/docs/zh/configuration.md)
- [API参考](/docs/zh/api.md)
- [使用示例](/docs/zh/examples.md)
- [工作原理](/docs/zh/how-it-works.md)
- [故障排除](/docs/zh/troubleshooting.md)

## 🌐 生态系统

[NodePassProject](https://github.com/NodePassProject) 组织开发了各种前端应用和辅助工具来增强 NodePass 体验：

- **[NodePassDash](https://github.com/NodePassProject/NodePassDash)**: 现代化的 NodePass 管理界面，提供主控管理、实例管理、流量统计、历史记录等功能。

- **[NodePanel](https://github.com/NodePassProject/NodePanel)**: 轻量化的前端面板，提供可视化的隧道管理功能，在 Vercel 或 Cloudflare Pages 轻松部署。

- **[npsh](https://github.com/NodePassProject/npsh)**: 简单易用的 NodePass 一键脚本合集，包括 API 主控、Dash 面板的安装部署、灵活配置和辅助管理。

## 💬 讨论

- 关注我们的 [Telegram 频道](https://t.me/NodePassChannel) 获取最新更新和社区支持。

- 加入我们的 [Discord](https://discord.gg/2cnXcnDMGc) 和 [Telegram 群组](https://t.me/NodePassGroup) 分享经验和想法。

## 📄 许可协议

`NodePass`项目根据[BSD 3-Clause许可证](LICENSE)授权。

## ⚖️ 免责声明

本项目以“现状”提供，开发者不提供任何明示或暗示的保证。用户使用风险自担，需遵守当地法律法规，仅限合法用途。开发者对任何直接、间接、偶然或后果性损害概不负责。进行二次开发须承诺合法使用并自负法律责任。开发者保留随时修改软件功能及本声明的权利。最终解释权归开发者所有。

## 🤝 赞助商

<table>
  <tr>
    <td width="200" align="center">
      <a href="https://whmcs.as211392.com"><img src="https://cdn.yobc.de/assets/dreamcloud.png"></a>
    </td>
    <td width="200" align="center">
      <a href="https://zmto.com"><img src="https://cdn.yobc.de/assets/zmto.png"></a>
    </td>
  </tr>
</table>

## ⭐ Star趋势

[![Stargazers over time](https://starchart.cc/yosebyte/nodepass.svg?variant=adaptive)](https://starchart.cc/yosebyte/nodepass)
