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

English | [简体中文](README_zh.md)
</div>

**NodePass** is an enterprise-grade TCP/UDP network tunneling tool featuring a modern three-tier architecture that separates control and data channels, with flexible, high-performance instance management capabilities. It offers zero-configuration deployment with intelligent connection pooling, tiered TLS encryption, and seamless protocol conversion capabilities. Designed for DevOps professionals and system administrators to effortlessly handle complex network scenarios including firewall traversal, NAT bypassing, and tunnel interconnection.

## 💎 Key Features

- **🔀 Multiple Operating Modes**
  - Server mode accepting incoming tunnels with configurable security
  - Client mode for establishing outbound connections to tunnel servers
  - Master mode with RESTful API for dynamic instance management

- **🌍 Protocol Support**
  - TCP tunneling with persistent connection handling
  - UDP datagram forwarding with configurable buffer sizes
  - Intelligent routing mechanisms for both protocols

- **🛡️ Security Options**
  - TLS Mode 0: Unencrypted mode for maximum speed in trusted networks
  - TLS Mode 1: Self-signed certificates for quick secure setup
  - TLS Mode 2: Custom certificate validation for enterprise security

- **⚡ Performance Features**
  - Smart connection pooling with real-time capacity adaptation
  - Dynamic interval adjustment based on network conditions
  - Minimal resource footprint even under heavy load

- **🧰 Simple Configuration**
  - Zero configuration files required
  - Simple command-line parameters
  - Environment variables for fine-tuning performance

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

## 🌐 Ecosystem

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
    <td width="200" align="center">
      <a href="https://whmcs.as211392.com"><img src="https://cdn.yobc.de/assets/dreamcloud.png"></a>
    </td>
    <td width="200" align="center">
      <a href="https://zmto.com"><img src="https://cdn.yobc.de/assets/zmto.png"></a>
    </td>
  </tr>
</table>

## ⭐ Stargazers

[![Stargazers over time](https://starchart.cc/yosebyte/nodepass.svg?variant=adaptive)](https://starchart.cc/yosebyte/nodepass)
