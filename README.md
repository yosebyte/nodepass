# nodepass

<div align="center">

**A lightweight and flexible TCP tunneling tool written in Go**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Container](https://img.shields.io/badge/Container-ghcr.io-2496ED?style=flat-square&logo=github)](https://github.com/yosebyte/nodepass/pkgs/container/nodepass)

</div>

`nodepass` allows you to forward TCP traffic from one address to another, making it useful for accessing services behind firewalls, load balancing, and creating secure tunnels.

## ✨ Features

- **Secure Tunneling** - Forward TCP traffic between networks with TLS encryption for control plane data
- **Flexible Architecture** - Run as either a server listening for connections or a client establishing tunnels
- **Efficient Connection Management** - Connection pooling with on-demand creation and one-time use
- **Health Monitoring** - Built-in health checks to ensure tunnel availability
- **Highly Configurable** - Customize operation via environment variables
- **Comprehensive Logging** - Multiple log levels for effective monitoring and debugging
- **Lightweight & Fast** - Minimal memory footprint with high-performance data transfer
- **Automatic Recovery** - Self-healing from connection failures with configurable cooldown periods

## 🚀 Installation

### Prerequisites
- Go 1.24 or newer

### From Source
```bash
git clone https://github.com/yosebyte/nodepass.git
cd nodepass
go build ./cmd/nodepass
```

### Using Go Install
```bash
go install github.com/yosebyte/nodepass@latest
```

### Using Container Image
```bash
# Pull the latest image
docker pull ghcr.io/yosebyte/nodepass:latest

# Or specific version
docker pull ghcr.io/yosebyte/nodepass:v1.0.0
```

### Binary Releases
Download pre-built binaries from the [releases page](https://github.com/yosebyte/nodepass/releases).

## 📋 Usage

```
nodepass <core_mode>://<tunnel_addr>/<target_addr>?<log=level>
```

### Core Modes

- `server` - Listens for incoming tunnel connections and forwards traffic
- `client` - Connects to a server to establish the tunnel

### Parameters

- `tunnel_addr` - The tunnel establishment address
  - Server: Address to listen on
  - Client: Address of the server to connect to
- `target_addr` - Destination address where traffic will be forwarded
- `log` - (Optional) Sets logging level: `debug`, `info`, `warn`, `error`, `fatal` (default: `info`)

### Examples

#### Server Mode

```bash
# Basic server mode
nodepass server://10.1.0.1:10101/10.1.0.1:8080?log=debug

# Using container image
docker run -p 10101:10101 -p 8080:8080 ghcr.io/yosebyte/nodepass \
  server://0.0.0.0:10101/0.0.0.0:8080?log=debug

# With environment variables
docker run -p 10101:10101 -p 8080:8080 \
  -e MAX_POOL_CAPACITY=2048 \
  -e REPORT_INTERVAL=2s \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/0.0.0.0:8080
```

This starts a server that:
- Listens for incoming connections on the specified interface and port
- Forwards traffic to the target service
- Uses the specified logging level and configuration

#### Client Mode

```bash
# Basic client mode
nodepass client://10.1.0.1:10101/127.0.0.1:8080?log=warn

# Using container image
docker run -p 8080:8080 ghcr.io/yosebyte/nodepass \
  client://server-host:10101/0.0.0.0:8080

# With environment variables
docker run -p 8080:8080 \
  -e MIN_POOL_CAPACITY=16 \
  -e CLIENT_COOLDOWN=10s \
  ghcr.io/yosebyte/nodepass client://server-host:10101/0.0.0.0:8080?log=debug
```

This starts a client that:
- Connects to the server at the specified address
- Forwards traffic to the specified local address
- Uses the configured settings

### Common Use Cases

#### Web Service Access
```bash
# Server side (where the web service is running)
docker run -p 10101:10101 ghcr.io/yosebyte/nodepass \
  server://0.0.0.0:10101/internal-web:3000

# Client side
docker run -p 3000:3000 ghcr.io/yosebyte/nodepass \
  client://server-host:10101/0.0.0.0:3000
```
Then access the web service at http://localhost:3000

#### Database Access
```bash
# Server side (with database access)
docker run -p 10101:10101 ghcr.io/yosebyte/nodepass \
  server://0.0.0.0:10101/db-server:5432

# Client side
docker run -p 5432:5432 ghcr.io/yosebyte/nodepass \
  client://server-host:10101/0.0.0.0:5432
```
Connect to the database using localhost:5432

#### High-Traffic API Gateway
```bash
# Server side with increased capacity
docker run -p 10101:10101 \
  -e MAX_POOL_CAPACITY=4096 \
  -e SEMAPHORE_LIMIT=2048 \
  -e REPORT_INTERVAL=1s \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/api-gateway:8080

# Client side
docker run -p 8080:8080 \
  -e MAX_POOL_CAPACITY=4096 \
  ghcr.io/yosebyte/nodepass client://server-host:10101/0.0.0.0:8080
```

### Docker Compose Examples

#### Basic Setup
```yaml
# docker-compose.yml
version: '3'

services:
  nodepass-server:
    image: ghcr.io/yosebyte/nodepass
    ports:
      - "10101:10101"
    environment:
      - MAX_POOL_CAPACITY=2048
      - REPORT_INTERVAL=5s
    command: server://0.0.0.0:10101/backend:8080
    networks:
      - internal

  nodepass-client:
    image: ghcr.io/yosebyte/nodepass
    ports:
      - "8080:8080" 
    environment:
      - MIN_POOL_CAPACITY=16
    command: client://nodepass-server:10101/0.0.0.0:8080
    depends_on:
      - nodepass-server
    networks:
      - external

  backend:
    image: nginx
    networks:
      - internal

networks:
  internal:
  external:
```

#### Multi-Service Setup
```yaml
# docker-compose.yml
version: '3'

services:
  tunnel-server:
    image: ghcr.io/yosebyte/nodepass
    ports:
      - "10101:10101"
    environment:
      - SEMAPHORE_LIMIT=4096
      - MAX_POOL_CAPACITY=4096
    command: server://0.0.0.0:10101/web-app:3000?log=debug
  
  web-app:
    image: node:alpine
    working_dir: /app
    command: npm start
    volumes:
      - ./app:/app
```

## 🔄 Architecture and Data Flow

### Connection Establishment

1. **Server Initialization**
   - Listens on `tunnel_addr` with TLS encryption for control plane
   - Opens a listener on a random port (`remoteAddr`) for data transfer
   - Pre-creates a connection pool for efficient handling

2. **Client Initialization**
   - Connects to server's `tunnel_addr` via TLS
   - Receives the random `remoteAddr` from server via secure tunnel

3. **Target Communication**
   - Both sides maintain awareness of `target_addr` destination

### Data Transfer Process

1. **Server accepts connection** on the `target_addr`
2. **Server signals client** through the TLS tunnel
3. **Both sides retrieve connections** from their respective pools
4. **Data is bidirectionally forwarded** between endpoints
5. **Connections are closed** after transfer completes

### Connection Pool Management

- **Single-use connections** - Each connection is used once then closed
- **Dynamic scaling** - Pools adjust capacity based on traffic demand
- **Efficient resource usage** - On-demand creation and prompt cleanup

## ⚙️ Configuration

Customize behavior using these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SEMAPHORE_LIMIT` | Max concurrent connections | 1024 |
| `SIGNAL_QUEUE_LIMIT` | Max signals in queue | 1024 |
| `SIGNAL_BUFFER` | Signal buffer size | 1024 |
| `MIN_POOL_CAPACITY` | Min connections in pool | 8 |
| `MAX_POOL_CAPACITY` | Max connections in pool | 1024 |
| `REPORT_INTERVAL` | Health check interval | 5s |
| `SERVER_COOLDOWN` | Server error recovery time | 5s |
| `CLIENT_COOLDOWN` | Client error recovery time | 5s |
| `SHUTDOWN_TIMEOUT` | Graceful shutdown period | 5s |

### Environment Variable Examples

```bash
# Using environment variables with binary
export MAX_POOL_CAPACITY=4096
export SERVER_COOLDOWN=10s
nodepass server://0.0.0.0:10101/app:8080

# Using environment variables with Docker
docker run -p 10101:10101 \
  -e REPORT_INTERVAL=2s \
  -e MAX_POOL_CAPACITY=2048 \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/service:8080
```

## 📦 Dependencies

- [yosebyte/x](https://github.com/yosebyte/x) - Go utility collection for logging, IO operations, TLS configuration, and connection pooling

## 📄 License

[MIT](LICENSE)
