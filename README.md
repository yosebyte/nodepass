# 🔗 NodePass

[![GitHub release](https://img.shields.io/github/v/release/yosebyte/nodepass)](https://github.com/yosebyte/nodepass/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yosebyte/nodepass)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/yosebyte/nodepass)](https://goreportcard.com/report/github.com/yosebyte/nodepass)

<div align="center">
  <img src="https://cdn.yobc.de/assets/nodepass.png" alt="nodepass">
</div>

**Language**: [English](README.md) | [简体中文](README_zh.md)

NodePass is an elegant, efficient TCP tunneling solution that creates secure communication bridges between network endpoints. By establishing a control channel secured with TLS encryption, NodePass facilitates seamless data transfer through otherwise restricted network environments. Its server-client architecture allows for flexible deployment scenarios, enabling access to services across firewalls, NATs, and other network barriers. With its intelligent connection pooling, minimal resource footprint, and straightforward command syntax, NodePass provides developers and system administrators with a powerful yet easy-to-use tool for solving complex networking challenges without compromising on security or performance.

## 📋 Table of Contents

- [Features](#-features)
- [Requirements](#-requirements)
- [Installation](#-installation)
  - [Option 1: Pre-built Binaries](#-option-1-pre-built-binaries)
  - [Option 2: Using Go Install](#-option-2-using-go-install)
  - [Option 3: Building from Source](#️-option-3-building-from-source)
  - [Option 4: Using Container Image](#-option-4-using-container-image)
- [Usage](#-usage)
  - [Server Mode](#️-server-mode)
  - [Client Mode](#-client-mode)
- [Configuration](#️-configuration)
  - [Log Levels](#-log-levels)
  - [Environment Variables](#-environment-variables)
- [Examples](#-examples)
  - [Basic Server Setup](#-basic-server-setup)
  - [Connecting to a NodePass Server](#-connecting-to-a-nodepass-server)
  - [Database Access Through Firewall](#-database-access-through-firewall)
  - [Secure Microservice Communication](#-secure-microservice-communication)
  - [IoT Device Management](#-iot-device-management)
  - [Multi-environment Development](#-multi-environment-development)
  - [Container Deployment](#-container-deployment)
- [How It Works](#-how-it-works)
- [Architectural Principles](#-architectural-principles)
- [Data Transmission Flow](#-data-transmission-flow)
- [Signal Communication Mechanism](#-signal-communication-mechanism)
- [Connection Pool Architecture](#-connection-pool-architecture)
- [Common Use Cases](#-common-use-cases)
- [Troubleshooting](#-troubleshooting)
  - [Connection Issues](#-connection-issues)
  - [Performance Optimization](#-performance-optimization)
- [Contributing](#-contributing)
- [Special Thanks](#-special-thanks)
- [License](#-license)
- [Stargazers](#-stargazers)

## ✨ Features

- **🔄 Dual Operating Modes**: Run as a server to accept connections or as a client to initiate them
- **🔒 TLS Encrypted Communication**: All tunnel traffic is secured using TLS encryption
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
- **🔄 URL-Based Signaling Protocol**: Elegant and extensible communication between endpoints
- **🛠️ Configurable Pool Dynamics**: Adjust connection pool behavior based on workload
- **🔌 One-Time Connection Pattern**: Enhanced security through non-reused connections
- **📡 Dynamic Port Allocation**: Automatically manages port assignments for secure communication

## 📋 Requirements

- Go 1.24 or higher (for building from source)
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
  ghcr.io/yosebyte/nodepass client://server.example.com:10101/127.0.0.1:8080
```

## 🚀 Usage

NodePass can be run in either server mode or client mode with a single, intuitive URL-style command:

### 🖥️ Server Mode

```bash
nodepass server://<tunnel_addr>/<target_addr>?log=<level>
```

- `tunnel_addr`: Address for the TLS tunnel endpoint (e.g., 10.1.0.1:10101)
- `target_addr`: Address of the service to be tunneled (e.g., 10.1.0.1:8080)
- `log`: Log level (debug, info, warn, error, fatal)

Example:
```bash
nodepass server://10.1.0.1:10101/10.1.0.1:8080?log=debug
```

### 📱 Client Mode

```bash
nodepass client://<tunnel_addr>/<target_addr>?log=<level>
```

- `tunnel_addr`: Address of the NodePass server's tunnel endpoint (e.g., 10.1.0.1:10101)
- `target_addr`: Local address to connect to (e.g., 127.0.0.1:8080)
- `log`: Log level (debug, info, warn, error, fatal)

Example:
```bash
nodepass client://10.1.0.1:10101/127.0.0.1:8080?log=info
```

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
| `REPORT_INTERVAL` | Interval for health check reports | 5s | `export REPORT_INTERVAL=10s` |
| `SERVICE_COOLDOWN` | Cooldown period before restart attempts | 5s | `export SERVICE_COOLDOWN=3s` |
| `SHUTDOWN_TIMEOUT` | Timeout for graceful shutdown | 5s | `export SHUTDOWN_TIMEOUT=10s` |

## 📚 Examples

### 🔐 Basic Server Setup

```bash
# Start a server tunneling traffic to a local web server
nodepass server://0.0.0.0:10101/127.0.0.1:8080?log=debug

# Start a server with increased connection limit
export SEMAPHORE_LIMIT=2048
nodepass server://10.1.0.1:10101/10.1.0.1:5432?log=info
```

### 🔌 Connecting to a NodePass Server

```bash
# Connect to a remote NodePass server and expose a service locally
nodepass client://server.example.com:10101/127.0.0.1:8080

# Connect with optimized pool settings for high-throughput scenarios
export MIN_POOL_CAPACITY=32
export MAX_POOL_CAPACITY=2048
nodepass client://10.1.0.1:10101/127.0.0.1:3000?log=debug
```

### 🗄 Database Access Through Firewall

```bash
# Server side (inside secured network)
nodepass client://server.example.com:10101/db.internal:5432

# Client side (outside the firewall)
nodepass server://:10101/127.0.0.1:5432

# Connect to database locally
psql -h 127.0.0.1 -p 5432 -U dbuser -d mydatabase
```

### 🔒 Secure Microservice Communication

```bash
# Service A (providing API)
nodepass server://0.0.0.0:10101/127.0.0.1:8081?log=warn

# Service B (consuming API)
nodepass client://service-a:10101/127.0.0.1:8082

# Service C (consuming API)
nodepass client://service-a:10101/127.0.0.1:8083

# All services communicate through encrypted channel
```

### 📡 IoT Device Management

```bash
# Central management server
nodepass server://0.0.0.0:10101/127.0.0.1:8888?log=info

# IoT device 1 
nodepass client://mgmt.example.com:10101/127.0.0.1:80

# IoT device 2
nodepass client://mgmt.example.com:10101/127.0.0.1:80

# All devices securely accessible from management interface
```

### 🧪 Multi-environment Development

```bash
# Production API access tunnel
nodepass server://0.0.0.0:10101/api.production:443?log=warn

# Development environment
nodepass client://tunnel.example.com:10101/127.0.0.1:3000

# Testing environment
nodepass client://tunnel.example.com:10101/127.0.0.1:3001

# Both environments can access production API securely
```

### 🐳 Container Deployment

```bash
# Create a network for the containers
docker network create nodepass-net

# Deploy NodePass server
docker run -d --name nodepass-server \
  --network nodepass-net \
  -p 10101:10101 \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/web-service:80?log=info

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

NodePass creates a network tunnel with a secure control channel:

1. **Server Mode**:
   - Sets up three listeners: tunnel (TLS-encrypted), remote (unencrypted), and target
   - Accepts incoming connections on the tunnel endpoint
   - When a client connects to the target, signals the client through the secure tunnel
   - The client then establishes a connection to the remote endpoint (unencrypted)
   - Data is exchanged between the target and remote connections

2. **Client Mode**:
   - Connects to the server's tunnel endpoint using TLS (encrypted control channel)
   - Listens for signals from the server through this secure channel
   - When a signal is received, connects to the server's remote endpoint (unencrypted data channel)
   - Establishes a connection to the local target address
   - Data is exchanged between the remote and local target connections

3. **Security Architecture**:
   - Only the tunnel connection (`tunnelConn`) between server and client is TLS-encrypted
   - The remote connections (`remoteConn`) that carry actual data are unencrypted TCP
   - The signaling and coordination happens over the secure TLS tunnel
   - This design balances security with performance for high-throughput scenarios

## 🏗 Architectural Principles

NodePass is built on several core architectural principles that ensure its reliability, security, and performance:

### 1. Separation of Concerns
The codebase maintains clear separation between:
- **Command Layer**: Handles user input and configuration (in `cmd/nodepass`)
- **Service Layer**: Implements the core client and server logic (in `internal`)
- **Common Layer**: Provides shared functionality between client and server components

### 2. Context-Based Flow Control
- Uses Go's context package for proper cancellation propagation
- Enables clean shutdown of all components when termination is requested
- Prevents resource leaks during service termination

### 3. Resilient Error Handling
- Implements automatic reconnection with configurable cooldown periods
- Gracefully handles network interruptions without user intervention
- Uses comprehensive error logging for troubleshooting

### 4. Security-First Design
- Employs TLS encryption for all tunnel traffic
- Generates in-memory TLS certificates when needed
- Follows principle of least privilege in network communications

### 5. Resource Efficiency
- Uses connection pooling to minimize connection establishment overhead
- Implements semaphore patterns for concurrency control
- Provides configurable limits to prevent resource exhaustion

## 🔄 Data Transmission Flow

NodePass establishes a bidirectional data flow through its tunnel architecture:

### Server-Side Flow
1. **Connection Initiation**:
   ```
   [Target Client] → [Target Listener] → [Server: Target Connection Created]
   ```

2. **Signal Generation**:
   ```
   [Server] → [Generate Unique Connection ID] → [Signal Client via TLS-Encrypted Tunnel]
   ```

3. **Connection Preparation**:
   ```
   [Server] → [Create Unencrypted Remote Connection in Pool] → [Wait for Client Connection]
   ```

4. **Data Exchange**:
   ```
   [Target Connection] ⟷ [conn.DataExchange] ⟷ [Remote Connection (Unencrypted)]
   ```

### Client-Side Flow
1. **Signal Reception**:
   ```
   [Client] → [Read Signal from TLS-Encrypted Tunnel] → [Parse Connection ID]
   ```

2. **Connection Establishment**:
   ```
   [Client] → [Retrieve Connection from Pool] → [Connect to Remote Endpoint (Unencrypted)]
   ```

3. **Local Connection**:
   ```
   [Client] → [Connect to Local Target] → [Establish Local Connection]
   ```

4. **Data Exchange**:
   ```
   [Remote Connection (Unencrypted)] ⟷ [conn.DataExchange] ⟷ [Local Target Connection]
   ```

### Bidirectional Exchange
The `conn.DataExchange()` function implements a concurrent bidirectional data pipe:
- Uses separate goroutines for each direction
- Efficiently handles data transfer in both directions simultaneously
- Properly propagates connection termination events

## 📡 Signal Communication Mechanism

NodePass uses a sophisticated URL-based signaling protocol through the TLS tunnel:

### Signal Types
1. **Remote Signal**:
   - Format: `remote://<port>`
   - Purpose: Informs the client about the server's remote endpoint port
   - Timing: Sent periodically during health checks

2. **Launch Signal**:
   - Format: `launch://<connection_id>`
   - Purpose: Requests the client to establish a connection for a specific ID
   - Timing: Sent when a new connection to the target service is received

### Signal Flow
1. **Signal Generation**:
   - Server creates URL-formatted signals for specific events
   - Signal is terminated with a newline character for proper parsing

2. **Signal Transmission**:
   - Server writes signals to the TLS tunnel connection
   - Uses a mutex to prevent concurrent writes to the tunnel

3. **Signal Reception**:
   - Client uses a buffered reader to read signals from the tunnel
   - Signals are trimmed and parsed into URL format

4. **Signal Processing**:
   - Client places valid signals in a buffered channel (signalChan)
   - A dedicated goroutine processes signals from the channel
   - Semaphore pattern prevents signal overflow

5. **Signal Execution**:
   - Remote signals update the client's remote address configuration
   - Launch signals trigger the `clientOnce()` method to establish connections

### Signal Resilience
- Buffered channel with configurable capacity prevents signal loss during high load
- Semaphore implementation ensures controlled concurrency
- Error handling for malformed or unexpected signals

## 🔌 Connection Pool Architecture

NodePass implements an efficient connection pooling system for managing network connections:

### Pool Design
1. **Pool Types**:
   - **Client Pool**: Pre-establishes connections to the remote endpoint
   - **Server Pool**: Manages incoming connections from clients

2. **Pool Components**:
   - **Connection Storage**: Thread-safe map of connection IDs to net.Conn objects
   - **ID Channel**: Buffered channel for available connection IDs
   - **Capacity Management**: Dynamic adjustment based on usage patterns
   - **Connection Factory**: Customizable connection creation function

### Connection Lifecycle
1. **Connection Creation**:
   - Connections are created up to the configured capacity
   - Each connection is assigned a unique ID
   - IDs and connections are stored in the pool

2. **Connection Acquisition**:
   - Client retrieves connections using connection IDs
   - Server retrieves the next available connection from the pool
   - Connections are validated before being returned

3. **Connection Usage**:
   - Connection is removed from the pool when acquired
   - Used for data exchange between endpoints
   - No connection reuse (one-time use model)

4. **Connection Termination**:
   - Connections are closed after use
   - Resources are properly released
   - Error handling ensures clean termination

### Pool Management
1. **Capacity Control**:
   - `MIN_POOL_CAPACITY`: Ensures minimum available connections
   - `MAX_POOL_CAPACITY`: Prevents excessive resource consumption
   - Dynamic scaling based on demand patterns

2. **Pool Managers**:
   - `ClientManager()`: Maintains the client connection pool
   - `ServerManager()`: Manages the server connection pool

3. **One-Time Connection Pattern**:
   Each connection in the pool follows a one-time use pattern:
   - Created and placed in the pool
   - Retrieved once for a specific data exchange
   - Never returned to the pool (prevents potential data leakage)
   - Properly closed after use

4. **Automatic Pool Size Adjustment**:
   - Pool capacity dynamically adjusts based on real-time usage patterns
   - If connection creation success rate is low (<20%), capacity decreases to minimize resource waste
   - If connection creation success rate is high (>80%), capacity increases to accommodate higher traffic
   - Gradual scaling prevents oscillation and provides stability
   - Respects configured minimum and maximum capacity boundaries
   - Scales down during periods of low activity to conserve resources
   - Scales up proactively when traffic increases to maintain performance
   - Self-tuning algorithm that adapts to varying network conditions
   - Separate adjustment logic for client and server pools to optimize for different traffic patterns

5. **Efficiency Considerations**:
   - Pre-establishment reduces connection latency
   - Connection validation ensures only healthy connections are used
   - Proper resource cleanup prevents connection leaks
   - Interval-based pool maintenance balances resource usage with responsiveness
   - Optimized connection validation with minimal overhead

## 💡 Common Use Cases

- **🚪 Remote Access**: Access services on private networks from external locations without VPN infrastructure. Ideal for accessing development servers, internal tools, or monitoring systems from remote work environments.

- **🧱 Firewall Bypass**: Navigate through restrictive network environments by establishing tunnels that use commonly allowed ports (like 443). Perfect for corporate environments with strict outbound connection policies or public Wi-Fi networks with limited connectivity.

- **🏛️ Legacy System Integration**: Connect modern applications to legacy systems securely without modifying the legacy infrastructure. Enables gradual modernization strategies by providing secure bridges between old and new application components.

- **🔒 Secure Microservice Communication**: Establish encrypted channels between distributed components across different networks or data centers. Allows microservices to communicate securely even across public networks without implementing complex service mesh solutions.

- **📱 Remote Development**: Connect to development resources from anywhere, enabling seamless coding, testing, and debugging against internal development environments regardless of developer location. Supports modern distributed team workflows and remote work arrangements.

- **☁️ Cloud-to-On-Premise Connectivity**: Link cloud services with on-premise infrastructure without exposing internal systems directly to the internet. Creates secure bridges for hybrid cloud architectures that require protected communication channels between environments.

- **🌍 Geographic Distribution**: Access region-specific services from different locations, overcoming geographic restrictions or testing region-specific functionality. Useful for global applications that need to operate consistently across different markets.

- **🧪 Testing Environments**: Create secure connections to isolated testing environments without compromising their isolation. Enables QA teams to access test systems securely while maintaining the integrity of test data and configurations.

- **🔄 API Gateway Alternative**: Serve as a lightweight alternative to full API gateways for specific services. Provides secure access to internal APIs without the complexity and overhead of comprehensive API management solutions.

- **🔒 Database Protection**: Enable secure database access while keeping database servers completely isolated from direct internet exposure. Creates a secure middle layer that protects valuable data assets from direct network attacks.

- **🌐 Cross-Network IoT Communication**: Facilitate communication between IoT devices deployed across different network segments. Overcomes NAT, firewall, and routing challenges common in IoT deployments spanning multiple locations.

- **🛠️ DevOps Pipeline Integration**: Connect CI/CD pipelines securely to deployment targets in various environments. Ensures build and deployment systems can securely reach production, staging, and testing environments without compromising network security.

## 🔧 Troubleshooting

### 📜 Connection Issues
- Verify firewall settings allow traffic on the specified ports
- Check that the tunnel address is correctly specified in client mode
- Ensure TLS certificates are properly generated
- Increase log level to debug for more detailed connection information
- Verify network stability between client and server endpoints
- Check for NAT traversal issues if operating across different networks
- Inspect system resource limits (file descriptors, etc.) if experiencing connection failures under load
- Verify DNS resolution if using hostnames for tunnel or target addresses

### 🚀 Performance Optimization

#### Connection Pool Tuning
- Adjust `MIN_POOL_CAPACITY` based on your minimum expected concurrent connections
  - Too low: Increased latency during traffic spikes as new connections must be established
  - Too high: Wasted resources maintaining idle connections
  - Recommended starting point: 25-50% of your average concurrent connections

- Configure `MAX_POOL_CAPACITY` to handle peak loads while preventing resource exhaustion
  - Too low: Connection failures during traffic spikes
  - Too high: Potential resource exhaustion affecting system stability
  - Recommended starting point: 150-200% of your peak concurrent connections

- Set `SEMAPHORE_LIMIT` based on expected peak concurrent tunneled sessions
  - Too low: Rejected connections during traffic spikes
  - Too high: Potential memory pressure from too many concurrent goroutines
  - Recommended range: 1000-5000 for most applications, higher for high-throughput scenarios

#### Network Configuration
- Optimize TCP settings on both client and server:
  - Adjust TCP keepalive intervals for long-lived connections
  - Consider TCP buffer sizes for high-throughput applications
  - Enable TCP BBR congestion control algorithm if available

#### Resource Allocation
- Ensure sufficient system resources on both client and server:
  - Monitor CPU usage during peak loads
  - Track memory consumption for connection management
  - Verify sufficient network bandwidth between endpoints

#### Monitoring Recommendations
- Implement connection tracking to identify bottlenecks
- Monitor connection establishment success rates
- Track data transfer rates to identify throughput issues
- Measure connection latency to optimize user experience

#### Advanced Scenarios
- For high-throughput applications:
  ```bash
  export MIN_POOL_CAPACITY=64
  export MAX_POOL_CAPACITY=4096
  export SEMAPHORE_LIMIT=8192
  export REPORT_INTERVAL=2s
  ```

- For low-latency applications:
  ```bash
  export MIN_POOL_CAPACITY=32
  export MAX_POOL_CAPACITY=1024
  export SEMAPHORE_LIMIT=2048
  export REPORT_INTERVAL=1s
  ```

- For resource-constrained environments:
  ```bash
  export MIN_POOL_CAPACITY=8
  export MAX_POOL_CAPACITY=256
  export SEMAPHORE_LIMIT=512
  export REPORT_INTERVAL=10s
  ```

## 👥 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 🎉 Special Thanks

Thank you to all the developers and users in the [NodeSeek](https://www.nodeseek.com/post-295115-1) community for your feedbacks. Feel free to reach out anytime with any technical issues.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⭐ Stargazers

[![Stargazers over time](https://starchart.cc/yosebyte/nodepass.svg?variant=adaptive)](https://starchart.cc/yosebyte/nodepass)

