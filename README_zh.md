<div align="center">
  <img src="https://cdn.yobc.de/assets/np-gopher.png" alt="nodepass" width="300">

[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#networking)
[![GitHub release](https://img.shields.io/github/v/release/yosebyte/nodepass)](https://github.com/yosebyte/nodepass/releases)
[![GitHub downloads](https://img.shields.io/github/downloads/yosebyte/nodepass/total.svg)](https://github.com/yosebyte/nodepass/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/yosebyte/nodepass)](https://goreportcard.com/report/github.com/yosebyte/nodepass)
[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![Go Reference](https://pkg.go.dev/badge/github.com/yosebyte/nodepass.svg)](https://pkg.go.dev/github.com/yosebyte/nodepass)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/yosebyte/nodepass)
![GitHub last commit](https://img.shields.io/github/last-commit/yosebyte/nodepass)

[English](README.md) | 简体中文
</div>

**NodePass** 是一款开源、轻量的企业级 TCP/UDP 网络隧道解决方案，采用多合一架构设计，通过控制通道与数据通道分离，实现灵活、高性能的实例管控。支持零配置文件部署，内置智能连接池、分级 TLS 加密和无缝协议转换。专为 DevOps 工程师和系统管理员打造，助力轻松应对复杂网络场景。

## 💎 核心功能

- **🌐 通用网络隧道**
  - 基础 TCP/UDP 隧道，具备协议转换能力，适配多种网络结构。
  - 完整适配端口映射、内网穿透、流量中转等多场景应用需求。
  - 多平台、多架构支持，支持独立二进制文件、容器灵活部署。

- **🚀 内置连接池**
  - 预先建立并维护连接，实现“零延迟”切换与高效流量转发。
  - 消除连接的握手等待，显著提升了性能体验。
  - 支持实时容量自适应，动态调整连接池规模。

- **🧬 创新架构设计**
  - Server-Client-Master 多模式整合架构设计，灵活切换。
  - 将 S/C 控制通道与数据通道完全解耦，相互独立、各司其职。
  - 主控-实例的管理方式，支持动态扩容、多实例协作和集中控制。

- **🔐 多级安全策略**
  - 三种 TLS 模式：明文、自签名、严格验证，适配不同安全等级。
  - 满足从开发测试到企业级高安全部署的全场景需求。
  - 支持证书文件的热重载，免停运、无缝处理证书更新问题。

- **⚙️ 极简配置方式**
  - 无需配置文件，仅命令行参数即可运行，适合自动化和快速迭代。
  - 适配 CI/CD 流程与容器环境，极大提升部署和运维效率。
  - 支持环境变量性能调优，灵活适应不同运行环境。

- **📈 高性能优化**
  - 智能流量调度与自动连接调优，极低资源占用。
  - 高并发、高负载状态下卓越的系统稳定性能。
  - 健康检查、断线重连、故障自愈，确保持续高可用。

- **💡 可视化管理**
  - 配套跨平台、多样化的管理前端应用，具备可视化配置能力。
  - 主流平台支持一键部署脚本，支撑灵活配置和辅助管理。
  - 具备实时隧道监控、实例管理、主控管理、流量统计等丰富功能。

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

## 🌱 生态系统

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
