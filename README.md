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

English | [简体中文](README_zh.md)
</div>

**NodePass** is an open-source, lightweight, enterprise-grade TCP/UDP network tunneling solution featuring an all-in-one architecture with separation of control and data channels, along with flexible and high-performance instance control. It supports zero-configuration deployment, intelligent connection pooling, tiered TLS encryption, and seamless protocol conversion. Designed for DevOps professionals and system administrators to effortlessly handle complex network scenarios.

## 💎 Key Features

- **🌐 Universal Functionality**
  - Basic TCP/UDP tunneling and protocol conversion across diverse networks.
  - Compatible with port mapping, NAT traversal, and traffic relay.
  - Cross-platform, multi-architecture, single binary or container.

- **🚀 Connection Pool**
  - Pre-established connections for zero-latency switching and forwarding.
  - Eliminates handshake delays, boosts performance.
  - Auto-scaling with real-time capacity adjustment.

- **🧬 Innovative Architecture**
  - Integrated S/C/M architecture, flexible mode switching.
  - Full decoupling of control/data channels.
  - API-instance management, multi-instance collaboration.

- **🔐 Multi-level Security**
  - Three TLS modes: plaintext, self-signed, strict validation.
  - Covers development to enterprise security needs.
  - Hot-reload certificates with zero downtime.

- **⚙️ Minimal Configuration**
  - No config files required, ready to use via CLI.
  - Optimized for CI/CD and containers.
  - Advanced parameters like timeouts and rate limits.

- **📈 Performance**
  - Intelligent scheduling, auto-tuning, ultra-low resource usage.
  - Stable under high concurrency and heavy load.
  - Health checks, auto-reconnect, self-healing.

- **💡 Visualization**
  - Rich cross-platform visual frontends.
  - One-click deployment scripts, easy management.
  - Real-time monitoring, API-instance management, traffic stats.

## 📋 Quick Start

### 📥 Installation

- **Pre-built Binaries**: Download from [releases page](https://github.com/yosebyte/nodepass/releases).
- **Container Image**: `docker pull ghcr.io/yosebyte/nodepass:latest`

### 🚀 Basic Usage

**Server Mode**
```bash
nodepass "server://:10101/127.0.0.1:8080?log=debug&tls=1"
```

**Client Mode**
```bash
nodepass "client://server:10101/127.0.0.1:8080?min=128"
```

**Master Mode (API)**
```bash
nodepass "master://:10101/api?log=debug&tls=1"
```

## 📚 Documentation

Explore the complete documentation to learn more about NodePass:

- [Installation Guide](/docs/en/installation.md)
- [Usage Instructions](/docs/en/usage.md)
- [Configuration Options](/docs/en/configuration.md)
- [API Reference](/docs/en/api.md)
- [Examples](/docs/en/examples.md)
- [How It Works](/docs/en/how-it-works.md)
- [Troubleshooting](/docs/en/troubleshooting.md)

## 🌱 Ecosystem

The [NodePassProject](https://github.com/NodePassProject) organization develops various frontend applications and auxiliary tools to enhance the NodePass experience:

- **[NodePassDash](https://github.com/NodePassProject/NodePassDash)**: A modern NodePass management interface that provides master management, instance management, traffic statistics, history records, and more.

- **[NodePanel](https://github.com/NodePassProject/NodePanel)**: A lightweight frontend panel that provides visual tunnel management, deployable on Vercel or Cloudflare Pages.

- **[npsh](https://github.com/NodePassProject/npsh)**: A collection of one-click scripts that provide simple deployment for API or Dashboard with flexible configuration and management.

## 💬 Discussion

- Follow our [Telegram Channel](https://t.me/NodePassChannel) for updates and community support.

- Join our [Discord](https://discord.gg/2cnXcnDMGc) and [Telegram Group](https://t.me/NodePassGroup) to share experiences and ideas.

## 📄 License

Project `NodePass` is licensed under the [BSD 3-Clause License](LICENSE).

## ⚖️ Disclaimer

This project is provided "as is" without any warranties. Users assume all risks and must comply with local laws for legal use only. Developers are not liable for any direct, indirect, incidental, or consequential damages. Secondary development requires commitment to legal use and self-responsibility for legal compliance. Developers reserve the right to modify software features and this disclaimer at any time. Final interpretation rights belong to developers.

## 🤝 Sponsors

<table>
  <tr>
    <td width="220" align="center">
      <a href="https://whmcs.as211392.com"><img src="https://cdn.yobc.de/assets/dreamcloud.png"></a>
    </td>
    <td width="220" align="center">
      <a href="https://t.me/xiao_bai_xue_zhang"><img src="https://cdn.yobc.de/assets/xuezhang.png"></a>
    </td>
  </tr>
  <tr>
    <td width="220" align="center">
      <a href="https://sharon.io"><img src="https://cdn.yobc.de/assets/sharon.png"></a>
    </td>
    <td width="220" align="center">
      <a href="https://zmto.com"><img src="https://cdn.yobc.de/assets/zmto.png"></a>
    </td>
  </tr>
</table>

## ⭐ Stargazers

[![Stargazers over time](https://starchart.cc/yosebyte/nodepass.svg?variant=adaptive)](https://starchart.cc/yosebyte/nodepass)
