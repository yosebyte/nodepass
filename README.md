# 🔗 NodePass - Enhanced

[![GitHub release](https://img.shields.io/github/v/release/yosebyte/nodepass)](https://github.com/yosebyte/nodepass/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yosebyte/nodepass)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/yosebyte/nodepass)](https://goreportcard.com/report/github.com/yosebyte/nodepass)

<div align="center">
  <img src="https://cdn.yobc.de/assets/nodepass.png" alt="nodepass">
</div>

**Language**: [English](README.md) | [简体中文](README_zh.md)

NodePass is an elegant, efficient TCP tunneling solution that creates secure communication bridges between network endpoints. By establishing an unencrypted TCP control channel, NodePass facilitates seamless data transfer through otherwise restricted network environments while offering configurable security options for the data channel. Its server-client architecture allows for flexible deployment scenarios, enabling access to services across firewalls, NATs, and other network barriers. With its intelligent connection pooling, minimal resource footprint, and straightforward command syntax, NodePass provides developers and system administrators with a powerful yet easy-to-use tool for solving complex networking challenges without compromising on security or performance.

## 📋 Table of Contents

- [Features](#-features)
- [Requirements](#-requirements)
- [Installation](#-installation)
  - [Option 1: Pre-built Binaries](#-option-1-pre-built-binaries)
  - [Option 2: Using Go Install](#-option-2-using-go-install)
  - [Option 3: Building from Source](#️-option-3-building-from-source)
  - [Option 4: Using Container Image](#-option-4-using-container-image)
  - [Option 5: Using Management Script](#-option-5-using-management-script)
- [Usage](#-usage)
  - [Server Mode](#️-server-mode)
  - [Client Mode](#-client-mode)
  - [Protocol Selection](#-protocol-selection)
- [Configuration](#️-configuration)
  - [Log Levels](#-log-levels)
  - [Environment Variables](#-environment-variables)
- [Examples](#-examples)
  - [Basic Server Setup with TLS Options](#-basic-server-setup-with-tls-options)
  - [Connecting to a NodePass Server](#-connecting-to-a-nodepass-server)
  - [Using QUIC Protocol](#-using-quic-protocol)
  - [Using WebSocket Connection](#-using-websocket-connection)
  - [Database Access Through Firewall](#-database-access-through-firewall)
  - [Secure Microservice Communication](#-secure-microservice-communication)
  - [IoT Device Management](#-iot-device-management)
  - [Multi-environment Development](#-multi-environment-development)
  - [Container Deployment](#-container-deployment)
- [How It Works](#-how-it-works)
- [Data Transmission Flow](#-data-transmission-flow)
- [Signal Communication Mechanism](#-signal-communication-mechanism)
- [Connection Pool Architecture](#-connection-pool-architecture)
- [Common Use Cases](#-common-use-cases)
- [Troubleshooting](#-troubleshooting)
  - [Connection Issues](#-connection-issues)
  - [Performance Optimization](#-performance-optimization)
- [Contributing](#-contributing)
- [Discussion](#-discussion)
- [License](#-license)
- [Stargazers](#-stargazers)

## ✨ Features

<div align="center">
  <img src="https://cdn.yobc.de/assets/np-cli.png" alt="nodepass">
</div>

- **🔄 Dual Operating Modes**: Run as a server to accept connections or as a client to initiate them
- **🌐 Multi-Protocol Support**: Tunnels TCP/UDP/QUIC/WebSocket traffic for complete application compatibility
- **🔒 Flexible TLS Options**: Three security modes for data channel encryption
- **🔐 Enforced TLS1.3 Encryption**: All encrypted connections use TLS1.3 for highest level of security
- **🚀 QUIC Protocol Support**: UDP-based QUIC protocol for lower latency and better multiplexing
- **🌐 WebSocket Full-Duplex Connection**: Support for HTTP-based WebSocket connections for firewall-restricted environments
- **🔌 Efficient Connection Pooling**: Optimized connection management with configurable pool sizes
- **📊 Flexible Logging System**: Configurable verbosity with five distinct logging levels
- **🛡️ Resilient Error Handling**: Automatic connection recovery and graceful shutdowns
- **📦 Single-Binary Deployment**: Simple to distribute and install with minimal dependencies
- **⚙️ Zero Configuration Files**: Everything is specified via command-line arguments and environment variables
- **🚀 Low Resource Footprint**: Minimal CPU and memory usage even under heavy load
- **♻️ Automatic Reconnection**: Seamlessly recovers from network interruptions
- **🧩 Modular Architecture**: Clean separation between client, server, and common components
- **🔍 Comprehensive Debugging**: Detailed connection tracing and signal monitoring
- **⚡ High-Performance Data Exchange**: Optimized bidirectional data transfer mechanism
- **🧠 Smart Connection Management**: Intelligent handling of connection states and lifecycles
- **📈 Scalable Semaphore System**: Prevents resource exhaustion during high traffic
- **🛠️ Configurable Pool Dynamics**: Adjust connection pool behavior based on workload
- **🔌 One-Time Connection Pattern**: Enhanced security through non-reused connections
- **📡 Dynamic Port Allocation**: Automatically manages port assignments for secure communication

## 📋 Requirements

- Go 1.18 or higher (for building from source)
- Network connectivity between server and client endpoints
- Admin privileges may be required for binding to ports below 1024

## 📥 Installation

### 💾 Option 1: Pre-built Binaries

Download the latest release for your platform from our [releases page](https://github.com/yosebyte/nodepass/releases).

### 🔧 Option 2: Using Go Install

```bash
go install github.com/yosebyte/nodepass/cmd/nodepass@latest
```

### 🛠️ Option 3: Building from Source

```bash
# Clone the repository
git clone https://github.com/yosebyte/nodepass.git

# Build the binary
cd nodepass
go build -o nodepass ./cmd/nodepass

# Optional: Install to your GOPATH/bin
go install ./cmd/nodepass
```

### 🐳 Option 4: Using Container Image

NodePass is available as a container image on GitHub Container Registry:

```bash
# Pull the container image
docker pull ghcr.io/yosebyte/nodepass:latest

# Run in server mode
docker run -d --name nodepass-server -p 10101:10101 -p 8080:8080 \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/0.0.0.0:8080

# Run in client mode
docker run -d --name nodepass-client \
  -e MIN_POOL_CAPACITY=32 \
  -e MAX_POOL_CAPACITY=512 \
  -p 8080:8080 \
  ghcr.io/yosebyte/nodepass client://nodepass-server:10101/127.0.0.1:8080
```

### 📜 Option 5: Using Management Script

For Linux systems, you can use our interactive management script for easy installation and service management:

```bash
bash <(curl -sL https://cdn.yobc.de/shell/nodepass.sh)
```

This script provides an interactive menu to:
- Install or update NodePass
- Create and configure multiple nodepass services
- Manage (start/stop/restart/delete) nodepass services
- Set up systemd services automatically
- Configure client and server modes with customizable options

## 🚀 Usage

NodePass creates a tunnel with an unencrypted TCP control channel and configurable TLS encryption options for the data exchange channel. It operates in two complementary modes:

### 🖥️ Server Mode

```bash
nodepass server://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

- `tunnel_addr`: Address for the TCP tunnel endpoint (control channel) that clients will connect to (e.g., 10.1.0.1:10101)
- `target_addr`: Address where the server listens for incoming connections (TCP and UDP) that will be tunneled to clients (e.g., 10.1.0.1:8080)
- `log`: Log level (debug, info, warn, error, fatal)
- `tls`: TLS encryption mode for the target data channel (0, 1, 2)
  - `0`: No TLS encryption (plain TCP/UDP)
  - `1`: Self-signed certificate (automatically generated)
  - `2`: Custom certificate (requires `crt` and `key` parameters)
- `crt`: Path to certificate file (required when `tls=2`)
- `key`: Path to private key file (required when `tls=2`)

In server mode, NodePass:
1. Listens for TCP tunnel connections (control channel) on `tunnel_addr`
2. Listens for incoming TCP and UDP traffic on `target_addr` 
3. When a connection arrives at `target_addr`, it signals the connected client through the unencrypted TCP tunnel
4. Creates a data channel for each connection with the specified TLS encryption level

Example:
```bash
# No TLS encryption for data channel
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=0"

# Self-signed certificate (auto-generated) with TLS1.3
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=1"

# Custom domain certificate with TLS1.3
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### 📱 Client Mode

```bash
nodepass client://<tunnel_addr>/<target_addr>?log=<level>
```

- `tunnel_addr`: Address of the NodePass server's tunnel endpoint to connect to (e.g., 10.1.0.1:10101)
- `target_addr`: Local address where traffic will be forwarded to (e.g., 127.0.0.1:8080)
- `log`: Log level (debug, info, warn, error, fatal)

In client mode, NodePass:
1. Connects to the server's unencrypted TCP tunnel endpoint (control channel) at `tunnel_addr`
2. Listens for signals from the server through this control channel
3. When a signal is received, establishes a data connection with the TLS security level specified by the server
4. Creates a local connection to `target_addr` and forwards traffic

Example:
```bash
# Connect to a NodePass server and automatically adopt its TLS security policy
nodepass client://10.1.0.1:10101/127.0.0.1:8080?log=info
```

### 🌐 Protocol Selection

NodePass now supports multiple protocol types that can be selected based on your needs:

- **TCP**: Default protocol suitable for most scenarios, providing reliable connections
- **UDP**: Suitable for applications requiring low latency, such as real-time communications and gaming
- **QUIC**: Modern UDP-based protocol offering low latency and better multiplexing, especially good for mobile networks
- **WebSocket**: HTTP-based protocol suitable for scenarios requiring firewall traversal, supporting full-duplex communication

The server automatically notifies clients about the protocol types it supports, and clients automatically select the best protocol for communication.

## ⚙️ Configuration

NodePass uses a minimalist approach with command-line parameters and environment variables:

### 📝 Log Levels

- `debug`: Verbose debugging information - shows all operations and connections
- `info`: General operational information (default) - shows startup, shutdown, and key events
- `warn`: Warning conditions - only shows potential issues that don't affect core functionality
- `error`: Error conditions - shows only problems that affect functionality
- `fatal`: Critical conditions - shows only severe errors that cause termination

### 🔧 Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SEMAPHORE_LIMIT` | Maximum number of concurrent connections | 1024 | `export SEMAPHORE_LIMIT=2048` |
| `MIN_POOL_CAPACITY` | Minimum connection pool size | 16 | `export MIN_POOL_CAPACITY=32` |
| `MAX_POOL_CAPACITY` | Maximum connection pool size | 1024 | `export MAX_POOL_CAPACITY=4096` |
| `UDP_DATA_BUF_SIZE` | Buffer size for UDP packets | 8192 | `export UDP_DATA_BUF_SIZE=16384` |
| `UDP_READ_TIMEOUT` | Timeout for UDP read operations | 5s | `export UDP_READ_TIMEOUT=10s` |
| `REPORT_INTERVAL` | Interval for health check reports | 5s | `export REPORT_INTERVAL=10s` |
| `RELOAD_INTERVAL` | Interval for certificate reload | 1h | `export RELOAD_INTERVAL=30m` |
| `SERVICE_COOLDOWN` | Cooldown period before restart attempts | 5s | `export SERVICE_COOLDOWN=3s` |
| `SHUTDOWN_TIMEOUT` | Timeout for graceful shutdown | 5s | `export SHUTDOWN_TIMEOUT=10s` |

## 📚 Examples

### 🔐 Basic Server Setup with TLS Options

```bash
# Start a server with no TLS encryption for data channel
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=0"

# Start a server with auto-generated self-signed certificate (using TLS1.3)
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# Start a server with custom domain certificate (using TLS1.3)
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### 🔌 Connecting to a NodePass Server

```bash
# Connect to a server (automatically adopts the server's TLS security policy)
nodepass client://server.example.com:10101/127.0.0.1:8080

# Connect with debug logging
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### 🚀 Using QUIC Protocol

The server automatically enables QUIC protocol support, and clients automatically detect and use QUIC protocol if available. QUIC protocol provides lower latency and better multiplexing, especially suitable for mobile networks and unstable network environments.

```bash
# Server side (automatically enables QUIC support)
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# Client side (automatically detects and uses QUIC)
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### 🌐 Using WebSocket Connection

WebSocket connections are especially suitable for scenarios requiring firewall traversal, as they are based on HTTP protocol which most firewalls allow.

```bash
# Server side (automatically enables WebSocket support)
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"

# Client side (automatically detects and uses WebSocket)
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### 🗄 Database Access Through Firewall

```bash
# Server side (outside secured network) with TLS1.3 encryption
nodepass server://:10101/127.0.0.1:5432?tls=1

# Client side (inside the firewall)
nodepass client://server.example.com:10101/127.0.0.1:5432
```

### 🔒 Secure Microservice Communication

```bash
# Service A (providing API) with custom certificate and TLS1.3
nodepass "server://0.0.0.0:10101/127.0.0.1:8081?log=warn&tls=2&crt=/path/to/service-a.crt&key=/path/to/service-a.key"

# Service B (consuming API)
nodepass client://service-a:10101/127.0.0.1:8082
```

### 📡 IoT Device Management

```bash
# Central management server (with TLS1.3 and WebSocket support)
nodepass "server://0.0.0.0:10101/127.0.0.1:8888?log=info&tls=1"

# IoT device (automatically uses best available protocol)
nodepass client://mgmt.example.com:10101/127.0.0.1:80
```

### 🧪 Multi-environment Development

```bash
# Production API access tunnel
nodepass client://tunnel.example.com:10101/127.0.0.1:3443

# Development environment
nodepass server://tunnel.example.com:10101/127.0.0.1:3000

# Testing environment (with TLS1.3)
nodepass "server://tunnel.example.com:10101/127.0.0.1:3001?log=warn&tls=1"
```

### 🐳 Container Deployment

```bash
# Create a network for the containers
docker network create nodepass-net

# Deploy NodePass server with self-signed certificate (using TLS1.3)
docker run -d --name nodepass-server \
  --network nodepass-net \
  -p 10101:10101 \
  ghcr.io/yosebyte/nodepass "server://0.0.0.0:10101/web-service:80?log=info&tls=1"

# Deploy a web service as target
docker run -d --name web-service \
  --network nodepass-net \
  nginx:alpine

# Deploy NodePass client
docker run -d --name nodepass-client \
  -p 8080:8080 \
  ghcr.io/yosebyte/nodepass client://nodepass-server:10101/127.0.0.1:8080?log=info

# Access the web service via http://localhost:8080
```

## 🔍 How It Works

NodePass creates a network architecture with separate channels for control and data:

1. **Control Channel (Tunnel)**:
   - Unencrypted TCP connection between client and server
   - Used exclusively for signaling and coordination
   - Maintains persistent connection for the lifetime of the tunnel

2. **Data Channel (Target)**:
   - Configurable TLS encryption options:
     - **Mode 0**: Unencrypted data transfer (fastest, least secure)
     - **Mode 1**: Self-signed certificate encryption (good security, no verification)
     - **Mode 2**: Verified certificate encryption (highest security, requires valid certificates)
   - All TLS encrypted connections use TLS1.3 for highest level of security
   - Created on-demand for each connection or datagram
   - Used for actual application data transfer
   - Supports multiple protocols: TCP, UDP, QUIC, and WebSocket

3. **Server Mode Operation**:
   - Listens for control connections on the tunnel endpoint
   - When traffic arrives at the target endpoint, signals the client via the control channel
   - Establishes data channels with the specified TLS mode when needed
   - Automatically enables QUIC and WebSocket support and notifies clients

4. **Client Mode Operation**:
   - Connects to the server's control channel
   - Listens for signals indicating incoming connections
   - Creates data connections using the TLS security level specified by the server
   - Forwards data between the local target and secure channel
   - Automatically detects and uses the best available protocol (TCP, QUIC, or WebSocket)

5. **Protocol Support**:
   - **TCP**: Full bidirectional streaming with persistent connections
   - **UDP**: Datagram forwarding with configurable buffer sizes and timeouts
   - **QUIC**: Modern UDP-based protocol offering low latency and better multiplexing
   - **WebSocket**: HTTP-based protocol supporting full-duplex communication, suitable for firewall-restricted environments

## 🔄 Data Transmission Flow

NodePass establishes a bidirectional data flow through its tunnel architecture, supporting TCP, UDP, QUIC, and WebSocket protocols:

### Server-Side Flow
1. **Connection Initiation**:
   ```
   [Target Client] → [Target Listener] → [Server: Target Connection Created]
   ```
   - For TCP: Client establishes persistent connection to target listener
   - For UDP: Server receives datagrams on UDP socket bound to target address
   - For QUIC: Server accepts QUIC connection requests
   - For WebSocket: Server accepts WebSocket upgrade requests

2. **Signal Generation**:
   ```
   [Server] → [Generate Unique Connection ID] → [Signal Client via Unencrypted TCP Tunnel]
   ```
   - For TCP: Generates a `//<connection_id>#1` signal
   - For UDP: Generates a `//<connection_id>#2` signal
   - For QUIC: Generates a `//<connection_id>#3` signal
   - For WebSocket: Generates a `//<connection_id>#4` signal

3. **Connection Preparation**:
   ```
   [Server] → [Create Remote Connection in Pool with Configured TLS Mode] → [Wait for Client Connection]
   ```
   - All protocols use the same connection pool mechanism with unique connection IDs
   - TLS configuration applied based on the specified mode (0, 1, or 2)
   - All TLS encrypted connections use TLS1.3

4. **Data Exchange**:
   ```
   [Target Connection] ⟷ [Exchange/Transfer] ⟷ [Remote Connection]
   ```
   - For TCP and WebSocket: Uses `conn.DataExchange()` for continuous bidirectional data streaming
   - For UDP: Individual datagrams are forwarded
   - For QUIC: Uses QUIC streams for efficient multiplexed data transfer

### Client-Side Flow
1. **Signal Reception**:
   ```
   [Client] ← [Receive Signal via Unencrypted TCP Tunnel] ← [Server]
   ```
   - Client parses signal to determine connection ID and protocol type

2. **Connection Establishment**:
   ```
   [Client] → [Get Remote Connection from Pool] → [Connect to Local Target]
   ```
   - Client retrieves pre-established connection from pool
   - Uses appropriate connection method based on protocol type (TCP, UDP, QUIC, or WebSocket)

3. **Data Exchange**:
   ```
   [Remote Connection] ⟷ [Exchange/Transfer] ⟷ [Local Target]
   ```
   - Uses same data exchange mechanism as server side
   - All protocols use the same interface for data transfer

4. **Connection Termination**:
   ```
   [Client] → [Close Connection] → [Return Statistics]
   ```
   - When either end closes the connection, the other end also closes
   - Bytes transferred and connection duration are recorded

## 🔄 Signal Communication Mechanism

NodePass uses a simple yet powerful signaling system to coordinate between client and server:

1. **Tunnel Establishment**:
   ```
   [Client] → [Connect to Server Tunnel Endpoint] → [Server]
   [Client] ← [Receive Port and TLS Mode Information] ← [Server]
   ```
   - Server sends URL-formatted signal containing remote port and TLS mode
   - Client parses signal and configures its connection parameters

2. **Connection Signals**:
   ```
   [Server] → [//<connection_id>#<protocol_type>] → [Client]
   ```
   - `<connection_id>` is a unique identifier
   - `<protocol_type>` indicates protocol: 1=TCP, 2=UDP, 3=QUIC, 4=WebSocket

3. **Health Checks**:
   ```
   [Server] → [Periodically Send Newline] → [Client]
   ```
   - Server periodically sends newline character to verify tunnel is still active
   - If error is detected, both ends attempt to re-establish tunnel

## 🔌 Connection Pool Architecture

NodePass uses an efficient connection pool system to optimize performance and resource usage:

1. **Server Pool**:
   - Pre-allocated connections waiting for client requests
   - Allocated when target connections arrive
   - Supports TCP, UDP, QUIC, and WebSocket connections

2. **Client Pool**:
   - Pre-established connections to server
   - Allocated based on signals
   - Dynamically sized to maintain optimal performance

3. **Pool Management**:
   - Automatically expands and contracts to accommodate load
   - Periodic health checks and connection refreshes
   - Intelligent resource allocation to prevent exhaustion

4. **Connection Lifecycle**:
   ```
   [Creation] → [Pool] → [Allocation] → [Usage] → [Closure/Return]
   ```
   - Connections pre-established before use
   - Closed after use to ensure security
   - Pool size configurable via environment variables

## 🔍 Common Use Cases

NodePass is suitable for various networking scenarios:

1. **Firewall Traversal**:
   - Access services behind firewalls
   - Provide multiple services through a single open port

2. **Secure Remote Access**:
   - Secure remote access with TLS1.3 encryption
   - Internal service access without VPN

3. **Microservice Communication**:
   - Service-to-service communication across network boundaries
   - Efficient service mesh using QUIC protocol

4. **IoT Device Connectivity**:
   - Remote device management and monitoring
   - Low-bandwidth device communication using WebSocket

5. **Development and Testing**:
   - Secure bridge from local development to production environments
   - Cross-environment testing and debugging

6. **Database Access**:
   - Secure remote database connections
   - Cross-network database replication

7. **API Proxying**:
   - Secure API gateway
   - Cross-domain API access

8. **Container Networking**:
   - Cross-host container communication
   - Container-to-external service bridging

## 🔧 Troubleshooting

### 🔌 Connection Issues

1. **Tunnel Establishment Failure**:
   - Check network connectivity and firewall rules
   - Verify server is running and listening on specified port
   - Use `log=debug` for detailed information

2. **Data Transfer Errors**:
   - Check TLS configuration
   - Verify target service is accessible
   - Check connection pool size and semaphore limits

3. **Performance Issues**:
   - Adjust connection pool sizes
   - Consider using QUIC protocol for better performance
   - Monitor resource usage

### 📊 Performance Optimization

1. **Connection Pool Tuning**:
   ```bash
   export MIN_POOL_CAPACITY=32
   export MAX_POOL_CAPACITY=2048
   ```
   - Increase minimum pool capacity for better responsiveness
   - Increase maximum pool capacity for higher concurrency

2. **Protocol Selection**:
   - Use QUIC protocol for low-latency requirements
   - Use WebSocket for firewall traversal
   - Use standard TCP for general purposes

3. **Buffer Sizes**:
   ```bash
   export UDP_DATA_BUF_SIZE=16384
   ```
   - Increase UDP buffer size for handling larger datagrams

4. **Timeout Settings**:
   ```bash
   export UDP_READ_TIMEOUT=10s
   export SHUTDOWN_TIMEOUT=10s
   ```
   - Adjust timeouts to accommodate network conditions

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests.

## 💬 Discussion

Join our [discussions](https://github.com/yosebyte/nodepass/discussions) to share your experiences and ideas.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⭐ Stargazers

[![Star History Chart](https://api.star-history.com/svg?repos=yosebyte/nodepass&type=Date)](https://star-history.com/#yosebyte/nodepass&Date)
