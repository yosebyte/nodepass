# Configuration Options

NodePass uses a minimalist approach to configuration, with all settings specified via command-line parameters and environment variables. This guide explains all available configuration options and provides recommendations for various deployment scenarios.

## Log Levels

NodePass provides six log verbosity levels that control the amount of information displayed:

- `none`: Disable logging - no log information displayed
- `debug`: Verbose debugging information - shows all operations and connections
- `info`: General operational information (default) - shows startup, shutdown, and key events
- `warn`: Warning conditions - only shows potential issues that don't affect core functionality
- `error`: Error conditions - shows only problems that affect functionality
- `event`: Event recording - shows important operational events and traffic statistics

You can set the log level in the command URL:

```bash
nodepass server://0.0.0.0:10101/0.0.0.0:8080?log=debug
```

## TLS Encryption Modes

For server and master modes, NodePass offers three TLS security levels for data channels:

- **Mode 0**: No TLS encryption (plain TCP/UDP)
  - Fastest performance, no overhead
  - No security for data channel (only use in trusted networks)
  
- **Mode 1**: Self-signed certificate (automatically generated)
  - Good security with minimal setup
  - Certificate is automatically generated and not verified
  - Protects against passive eavesdropping
  
- **Mode 2**: Custom certificate (requires `crt` and `key` parameters)
  - Highest security with certificate validation
  - Requires providing certificate and key files
  - Suitable for production environments

Example with TLS Mode 1 (self-signed):
```bash
nodepass server://0.0.0.0:10101/0.0.0.0:8080?tls=1
```

Example with TLS Mode 2 (custom certificate):
```bash
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

## Run Mode Control

NodePass supports configurable run modes via the `mode` query parameter to control the behavior of both client and server instances. This provides flexibility in deployment scenarios where automatic mode detection may not be suitable.

### Client Mode Control

For client instances, the `mode` parameter controls the connection strategy:

- **Mode 0** (Default): Automatic mode detection
  - Attempts to bind to tunnel address locally first
  - If successful, operates in single-end forwarding mode
  - If binding fails, operates in dual-end handshake mode
  
- **Mode 1**: Force single-end forwarding mode
  - Binds to tunnel address locally and forwards traffic directly to target
  - Uses direct connection establishment for high performance
  - No handshake with server required
  
- **Mode 2**: Force dual-end handshake mode
  - Always connects to remote server for tunnel establishment
  - Requires handshake with server before data transfer
  - Supports bidirectional data flow coordination

Example:
```bash
# Force client to operate in single-end forwarding mode
nodepass "client://127.0.0.1:1080/target.example.com:8080?mode=1"

# Force client to operate in dual-end handshake mode
nodepass "client://server.example.com:10101/127.0.0.1:8080?mode=2"
```

### Server Mode Control

For server instances, the `mode` parameter controls the data flow direction:

- **Mode 0** (Default): Automatic flow direction detection
  - Attempts to bind to target address locally first
  - If successful, operates in reverse mode (server receives traffic)
  - If binding fails, operates in forward mode (server sends traffic)
  
- **Mode 1**: Force reverse mode
  - Server binds to target address locally and receives traffic
  - Incoming connections are forwarded to connected clients
  - Data flow: External → Server → Client → Target
  
- **Mode 2**: Force forward mode  
  - Server connects to remote target address
  - Client connections are forwarded to remote target
  - Data flow: Client → Server → External Target

Example:
```bash
# Force server to operate in reverse mode
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?mode=1"

# Force server to operate in forward mode
nodepass "server://0.0.0.0:10101/remote.example.com:8080?mode=2"
```

## DNS Resolution Configuration

NodePass uses the system's built-in DNS resolver with intelligent caching for improved performance and reliability. The DNS cache reduces query overhead and prevents resolution delays.

- `dns`: DNS cache TTL duration (default: 5m)
  - Specifies how long resolved hostnames are cached before re-querying
  - Accepts time duration format: `1h`, `30m`, `15s`, `500ms`, etc.
  - Longer TTL reduces DNS query overhead but may cache stale records
  - Shorter TTL ensures fresher DNS data but increases query frequency
  - Set to `0` to disable caching (always query DNS on every connection)
  - Applies to both client and server modes for resolving all hostnames

**DNS Cache Features:**
- **System Integration**: Uses operating system's native DNS resolver for maximum compatibility
- **Intelligent Caching**: Resolved hostnames are cached with configurable TTL to reduce query overhead
- **Automatic Expiration**: Cached entries are automatically removed after TTL expires
- **IP Address Bypass**: Direct IP addresses skip DNS resolution for maximum efficiency
- **Protocol-Aware**: Automatically handles both IPv4 and IPv6 addresses
- **Thread-Safe**: Concurrent DNS lookups are safely cached and shared across connections

Example:
```bash
# Use default 5-minute cache TTL
nodepass "server://0.0.0.0:10101/example.com:8080"

# Set 1-hour cache TTL for stable domains
nodepass "server://0.0.0.0:10101/internal.example.com:8080?dns=1h"

# Set 30-second cache TTL for dynamic DNS
nodepass "client://server.example.com:10101/database.local:3306?dns=30s"

# Disable DNS caching entirely (query on every connection)
nodepass "server://0.0.0.0:10101/service.example.com:8080?dns=0"

# Combined with other parameters
nodepass "server://0.0.0.0:10101/backend.example.com:8080?dns=10m&log=info&tls=1&mode=2"
```

**DNS Configuration Use Cases:**
- **Corporate Networks**: Use longer TTL (e.g., 1h) for stable internal hostnames
- **Dynamic DNS**: Use shorter TTL (e.g., 30s) for frequently changing DNS records
- **High Availability**: Longer TTL reduces DNS server load and improves reliability
- **Load Balancing**: Shorter TTL enables faster failover for load-balanced services
- **Performance**: Longer TTL reduces connection latency by minimizing DNS queries

**DNS Caching Behavior:**
- Cache TTL is configurable via the `dns` query parameter (default: 5 minutes)
- Expired entries are removed and fresh lookups performed on next access
- Cache is per-instance and not shared between NodePass processes
- IP addresses are never cached (direct use, no DNS lookup needed)
- System DNS resolver is used for all hostname lookups

**Important Notes:**
- Both IPv4 and IPv6 addresses are supported
- DNS resolution timeout is controlled by the operating system
- When using target address groups, each address is resolved independently
- DNS resolution applies to both tunnel addresses and target addresses
- Tunnel address DNS resolution occurs once at startup
- Target address DNS resolution uses caching for repeated connections

## Outbound Connection Source IP Control

NodePass supports specifying the local IP address used for outbound connections to target addresses. This feature is useful for systems with multiple network interfaces where traffic routing needs to be controlled explicitly.

- `dial`: Source IP address for outbound connections (default: auto)
  - Value `auto` or omitted: System automatically selects the source IP based on routing table
  - Valid IP address: Forces all outbound connections to use the specified local IP address
  - Applies to both TCP and UDP connections to target addresses
  - Applies to both client and server modes
  - Automatic fallback to system-selected IP if the specified address fails
  - Invalid IP addresses trigger an error log and fallback to auto mode

The `dial` parameter provides precise control over which network interface NodePass uses for outbound connections, enabling advanced network configurations such as:

Example:
```bash
# Server with specific source IP for outbound connections
nodepass "server://0.0.0.0:10101/remote.example.com:8080?dial=10.1.0.100"

# Client with specific source IP for target connections
nodepass "client://server.example.com:10101/127.0.0.1:8080?dial=192.168.1.50"

# Combined with other parameters
nodepass "server://0.0.0.0:10101/remote.example.com:8080?log=info&tls=1&dial=10.1.0.100&mode=2"
```

**Source IP Control Use Cases:**
- **Multi-Homed Systems**: Control which network interface is used for outbound traffic
- **Policy Routing**: Ensure traffic uses specific routes based on source IP
- **Network Segmentation**: Direct traffic through specific VLANs or network zones
- **Firewall Rules**: Match specific firewall rules that filter by source IP
- **Load Distribution**: Distribute outbound traffic across multiple network links
- **Testing**: Simulate traffic from specific network locations or interfaces

**Automatic Fallback Behavior:**
- If the specified IP address cannot be bound (e.g., doesn't exist on the system), NodePass logs an error and automatically falls back to system-selected IP
- The fallback only occurs once per instance - after fallback, all subsequent connections use auto mode
- Fallback is logged at ERROR level for visibility: "dialWithRotation: fallback to system auto due to dialer failure"

**Important Notes:**
- The specified IP must exist on the local system and be properly configured
- Source IP applies only to outbound connections to target addresses, not tunnel connections
- IPv4 and IPv6 addresses are both supported (address family must match target address)
- Binding failures trigger automatic fallback to prevent connection failures
- This parameter does not affect incoming tunnel connections or server listen addresses

## Connection Pool Types

NodePass supports three connection pool types for tunnel connection management in dual-end handshake mode. Each type provides different transport protocols and performance characteristics.

- `type`: Connection pool type (default: 0)
  - Value 0: Use TCP-based connection pool (traditional pool library)
  - Value 1: Use QUIC-based connection pool (UDP multiplexing with streams)
  - Value 2: Use WebSocket/WSS-based connection pool (HTTP upgrade connections)
  - Only applies to dual-end handshake mode (mode=2)
  - Automatically enables TLS if not already configured (minimum tls=1)
  - Server configuration is automatically delivered to client during handshake

### TCP Pool (type=0)

Traditional TCP-based connection pool providing maximum compatibility and reliability.

**Advantages:**
- Maximum network compatibility, TCP supported by virtually all networks
- Strict TCP semantics and ordering guarantees
- Stable performance in low-loss networks
- Widely accepted in enterprise environments
- Low latency in stable networks

**Use Cases:**
- Networks that block or throttle UDP traffic
- Applications requiring strict TCP semantics
- Default choice in enterprise environments
- Maximum compatibility requirements

### QUIC Pool (type=1)

Modern connection pool based on QUIC protocol, providing UDP-based multiplexed streams.

**Advantages:**
- **Multiplexing**: Multiple streams over a single UDP connection
- **Reduced Latency**: Faster connection establishment with 0-RTT support
- **Better Loss Recovery**: Stream-level flow control and congestion management
- **NAT Traversal**: UDP-based protocol works better through NATs and firewalls
- **Built-in Encryption**: Mandatory TLS 1.3 encryption for all QUIC connections

**Use Cases:**
- Mobile networks or frequently changing network conditions
- High-latency connections (satellite, long-distance)
- NAT-heavy environments
- Real-time applications benefiting from stream independence
- Scenarios where 0-RTT reconnection provides value

**Requirements:**
- TLS mode must be enabled (tls=1 or tls=2)
- Only available in dual-end handshake mode (mode=2)
- UDP port accessibility required

### WebSocket Pool (type=2)

Connection pool based on WebSocket protocol, establishing connections via HTTP upgrade.

**Advantages:**
- **Proxy Traversal**: Can traverse HTTP proxies and CDNs
- **Firewall Friendly**: Uses standard HTTP/HTTPS ports, easily passes through firewalls
- **Web Infrastructure Compatible**: Integrates with existing web infrastructure
- **Bidirectional Communication**: Supports full-duplex communication
- **Wide Support**: Supported by all modern browsers and platforms

**Use Cases:**
- Need to traverse HTTP proxies or CDNs
- Corporate environments allowing only HTTP/HTTPS traffic
- Firewalls blocking raw TCP connections
- Need compatibility with existing web infrastructure
- Web proxy or VPN alternative solutions

**Requirements:**
- TLS mode must be enabled for WSS (tls=1 or tls=2)
- Only available in dual-end handshake mode (mode=2)
- TCP port with WebSocket upgrade support required

### Configuration Examples

```bash
# TCP pool (default)
nodepass "server://0.0.0.0:10101/remote.example.com:8080?type=0&mode=2&tls=1"

# QUIC pool (automatically enables TLS)
nodepass "server://0.0.0.0:10101/remote.example.com:8080?type=1&mode=2"

# WebSocket pool (with custom TLS certificate)
nodepass "server://0.0.0.0:10101/remote.example.com:8080?type=2&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"

# Client automatically adopts server's pool type configuration
nodepass "client://server.example.com:10101/127.0.0.1:8080?mode=2"
```

**Important Notes:**
- Only server needs to configure `type` parameter - client receives configuration automatically
- If type=1 or type=2 but TLS not configured, system automatically sets tls=1
- All pool types only available in dual-end handshake mode (mode=2 or mode=0 with remote addresses)
- Not applicable to single-end forwarding mode (mode=1)

**Pool Type Use Cases:**
- **TCP Pool**: Standard enterprise environments, maximum compatibility, stable networks
- **QUIC Pool**: High-latency networks, mobile networks, real-time applications, complex NAT environments
- **WebSocket Pool**: HTTP proxy traversal, enterprise firewall restrictions, web infrastructure integration

## Connection Pool Capacity Parameters

Connection pool capacity parameters only apply to dual-end handshake mode and are configured through different approaches:

- `min`: Minimum connection pool capacity (default: 64) - Set by client via URL query parameters
- `max`: Maximum connection pool capacity (default: 1024) - Determined by server and delivered to client during handshake

**Important Notes**:
- The `max` parameter set by client will be overridden by the value delivered from server during handshake
- The `min` parameter is fully controlled by client and will not be modified by server
- In client single-end forwarding mode, connection pools are not used and these parameters are ignored
- Applies to all connection pool types (type=0 for TCP, type=1 for QUIC, type=2 for WebSocket)

Example:
```bash
# Client sets minimum pool to 32, maximum pool will be determined by server
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=32"

# Client with QUIC pool and custom pool capacity
nodepass "client://server.example.com:10101/127.0.0.1:8080?type=1&min=128"
```

## Data Read Timeout

The `read` parameter controls data read timeout behavior:

- `read`: Data read timeout (default: 0, meaning no timeout)
  - Value 0 or omitted: No data read timeout
  - Positive integer with time unit: Sets read timeout
    - Value format: integer followed by unit (`s` for seconds, `m` for minutes, `h` for hours)
    - Examples: `30s` (30 seconds), `5m` (5 minutes), `1h` (1 hour)
    - If no data is received within the timeout period, the connection is closed
  - Applies to both client and server modes

Example:
```bash
# Set data read timeout to 5 minutes
nodepass "client://server.example.com:10101/127.0.0.1:8080?read=5m"

# Set data read timeout to 30 seconds for fast-response applications
nodepass "client://server.example.com:10101/127.0.0.1:8080?read=30s"

# Set data read timeout to 1 hour for long-running transfers
nodepass "client://server.example.com:10101/127.0.0.1:8080?read=1h"

# Default behavior: no timeout (omit read parameter or set to 0)
nodepass "client://server.example.com:10101/127.0.0.1:8080"
```

**Data Read Timeout Use Cases:**
- **Connection Management**: Prevent idle connections from consuming resources indefinitely
- **Resource Control**: Set appropriate timeouts based on expected data transfer patterns
- **Network Reliability**: Handle network interruptions gracefully with automatic cleanup

## Rate Limiting
NodePass supports bandwidth rate limiting for traffic control through the `rate` parameter. This feature helps prevent network congestion and ensures fair resource allocation across multiple connections.

- `rate`: Maximum bandwidth limit in Mbps (Megabits per second)
  - Value 0 or omitted: No rate limiting (unlimited bandwidth)
  - Positive integer: Rate limit in Mbps (e.g., 10 means 10 Mbps)
  - Applied to both upload and download traffic
  - Uses token bucket algorithm for smooth traffic shaping

Example:
```bash
# Limit bandwidth to 50 Mbps
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?rate=50"

# Client with 100 Mbps rate limit
nodepass "client://server.example.com:10101/127.0.0.1:8080?rate=100"

# Combined with other parameters
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?log=error&tls=1&rate=50"
```

**Rate Limiting Use Cases:**
- **Bandwidth Control**: Prevent NodePass from consuming all available bandwidth
- **Fair Sharing**: Ensure multiple applications can share network resources
- **Cost Management**: Control data usage in metered network environments
- **QoS Compliance**: Meet service level agreements for bandwidth usage
- **Testing**: Simulate low-bandwidth environments for application testing

## Connection Slot Limit

NodePass provides connection slot limiting to control the maximum number of concurrent connections and prevent resource exhaustion. This feature helps maintain system stability and predictable performance under high load conditions.

- `slot`: Maximum total connection limit (default: 65536)
  - Value 0 or omitted: No connection limit (unlimited concurrent connections)
  - Positive integer: Maximum number of concurrent TCP and UDP connections combined
  - Applies to both client and server modes
  - Connections exceeding the limit will be rejected until slots become available
  - Separate tracking for TCP and UDP connections with combined limit enforcement

The slot limit provides a circuit breaker mechanism that prevents NodePass from accepting more connections than the system can handle effectively. When the limit is reached, new connection attempts are rejected immediately rather than queuing, which helps maintain low latency for existing connections.

Example:
```bash
# Limit total concurrent connections to 1000
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?slot=1000"

# Client with 500 connection limit
nodepass "client://server.example.com:10101/127.0.0.1:8080?slot=500"

# Combined with other parameters
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?log=info&tls=1&slot=2000&rate=100"
```

**Connection Slot Limit Use Cases:**
- **Resource Protection**: Prevent system overload by limiting concurrent connections
- **Memory Management**: Control memory usage in high-traffic scenarios
- **Performance Consistency**: Maintain predictable latency by avoiding resource contention
- **Capacity Planning**: Set known limits for infrastructure capacity planning
- **DoS Protection**: Provide basic protection against connection flood attacks
- **Service Stability**: Ensure critical services remain responsive under load

**Best Practices for Slot Configuration:**
- **Small Systems**: Set conservative limits (100-1000) for resource-constrained environments
- **High-Performance Systems**: Configure higher limits (10000-50000) based on available memory and CPU
- **Load Testing**: Determine optimal limits through performance testing under expected load
- **Monitoring**: Track connection usage to identify if limits need adjustment
- **Headroom**: Leave 20-30% headroom below theoretical system limits for stability

## PROXY Protocol Support

NodePass supports PROXY protocol v1 for preserving client connection information when forwarding traffic through load balancers, reverse proxies, or other intermediary services.

- `proxy`: PROXY protocol support (default: 0)
  - Value 0: Disabled - no PROXY protocol header is sent
  - Value 1: Enabled - sends PROXY protocol v1 header before data transfer
  - Works with both TCP4 and TCP6 connections
  - Compatible with HAProxy, Nginx, and other PROXY protocol aware services

The PROXY protocol header includes original client IP, server IP, and port information, allowing downstream services to identify the real client connection details even when traffic passes through NodePass tunnels.

Example:
```bash
# Enable PROXY protocol v1 for server mode
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?proxy=1"

# Enable PROXY protocol v1 for client mode  
nodepass "client://server.example.com:10101/127.0.0.1:8080?proxy=1"

# Combined with other parameters
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?log=info&tls=1&proxy=1&rate=100"
```

**PROXY Protocol Use Cases:**
- **Load Balancer Integration**: Preserve client IP information when forwarding through load balancers
- **Reverse Proxy Support**: Enable backend services to see original client connections
- **Logging and Analytics**: Maintain accurate client connection logs for security and analysis
- **Access Control**: Allow downstream services to apply IP-based access controls
- **Compliance**: Meet regulatory requirements for connection logging and auditing

**Important Notes:**
- The target service must support PROXY protocol v1 to properly handle the header
- PROXY headers are only sent for TCP connections, not UDP
- The header format follows the HAProxy PROXY protocol v1 specification
- If the target service doesn't support PROXY protocol, connections may fail or behave unexpectedly

## TCP Support Control

NodePass supports TCP traffic tunneling by default. The `notcp` parameter allows you to disable TCP support when only UDP traffic needs to be handled, which can reduce resource usage and simplify configuration.

- `notcp`: TCP support control (default: 0)
  - Value 0: TCP support enabled - both TCP and UDP traffic will be tunneled
  - Value 1: TCP support disabled - only UDP traffic will be tunneled, TCP connections are ignored
  - Applies to both client and server modes
  - When disabled, TCP-related resources (buffers, connections, sessions) are not allocated

Example:
```bash
# Enable TCP support (default behavior)
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?notcp=0"

# Disable TCP support for UDP-only scenarios
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?notcp=1"

# Client with TCP disabled
nodepass "client://server.example.com:10101/127.0.0.1:8080?notcp=1"

# Combined with other parameters
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?log=info&tls=1&notcp=1"
```

**TCP Support Control Use Cases:**
- **UDP-Only Services**: Disable TCP when tunneling only UDP-based applications
- **Resource Optimization**: Reduce memory and CPU usage by avoiding TCP processing overhead
- **Security**: Prevent TCP-based attacks or unwanted traffic in restricted environments
- **Simplified Configuration**: Easier setup when TCP tunneling is not required
- **Network Isolation**: Isolate TCP and UDP traffic handling for better control

**Important Notes:**
- When TCP is disabled, any TCP connections sent to the tunnel will be silently dropped
- Existing TCP sessions will be terminated when switching to notcp=1
- TCP buffer pools and session management are disabled when notcp=1

## UDP Support Control

NodePass supports UDP traffic tunneling in addition to TCP. The `noudp` parameter allows you to disable UDP support when only TCP traffic needs to be handled, which can reduce resource usage and simplify configuration.

- `noudp`: UDP support control (default: 0)
  - Value 0: UDP support enabled - both TCP and UDP traffic will be tunneled
  - Value 1: UDP support disabled - only TCP traffic will be tunneled, UDP packets are ignored
  - Applies to both client and server modes
  - When disabled, UDP-related resources (buffers, connections, sessions) are not allocated

Example:
```bash
# Enable UDP support (default behavior)
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?noudp=0"

# Disable UDP support for TCP-only scenarios
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?noudp=1"

# Client with UDP disabled
nodepass "client://server.example.com:10101/127.0.0.1:8080?noudp=1"

# Combined with other parameters
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?log=info&tls=1&noudp=1"
```

**UDP Support Control Use Cases:**
- **TCP-Only Services**: Disable UDP when tunneling only TCP-based applications
- **Resource Optimization**: Reduce memory and CPU usage by avoiding UDP processing overhead
- **Security**: Prevent UDP-based attacks or unwanted traffic in restricted environments
- **Simplified Configuration**: Easier setup when UDP tunneling is not required
- **Network Isolation**: Isolate TCP and UDP traffic handling for better control

**Important Notes:**
- When UDP is disabled, any UDP packets sent to the tunnel will be silently dropped
- Existing UDP sessions will be terminated when switching to noudp=1
- UDP buffer pools and session management are disabled when noudp=1

## Target Address Groups and Load Balancing

NodePass supports configuring multiple target addresses to achieve high availability and load balancing. Target address groups are only applicable to the egress side (the final destination of traffic) and should not be used on the ingress side.

### Target Address Group Configuration

Target address groups are configured by separating multiple addresses with commas. NodePass automatically performs round-robin and failover across these addresses:

```bash
# Server with multiple backend targets (forward mode, mode=2)
nodepass "server://0.0.0.0:10101/backend1.example.com:8080,backend2.example.com:8080,backend3.example.com:8080?mode=2&tls=1"

# Client with multiple local services (single-end forwarding mode, mode=1)
nodepass "client://127.0.0.1:1080/app1.local:8080,app2.local:8080?mode=1"
```

### Rotation Strategy

NodePass employs a Round-Robin algorithm that combines failover and load balancing features:

- **Load Balancing**: After each successful connection establishment, automatically switches to the next target address for even traffic distribution
- **Failover**: When a connection to an address fails, immediately tries the next address to ensure service availability
- **Automatic Recovery**: Failed addresses are retried in subsequent rotation cycles and automatically resume receiving traffic after recovery

### Use Cases

Target address groups are suitable for the following scenarios:

- **High Availability Deployment**: Multiple backend servers for automatic failover
- **Load Balancing**: Even traffic distribution across multiple backend instances
- **Canary Releases**: Gradually shifting traffic to new service versions
- **Geographic Distribution**: Selecting optimal paths based on network topology

### Important Notes

- **Egress Only**: Target address groups can only be configured at the final traffic destination
  - ✓ Server forward mode (mode=2): `server://0.0.0.0:10101/target1:80,target2:80`
  - ✓ Client single-end forwarding mode (mode=1): `client://127.0.0.1:1080/target1:80,target2:80`
  - ✗ Tunnel addresses do not support: Do not use multi-address configuration for tunnel addresses
  
- **Address Format**: All addresses must use the same port or explicitly specify the port for each address
- **Protocol Consistency**: All addresses in the group must support the same protocol (TCP/UDP)
- **Thread Safety**: Rotation index uses atomic operations, supporting high-concurrency scenarios

Example configurations:

```bash
# Correct example: Server with 3 backend web servers
nodepass "server://0.0.0.0:10101/web1.internal:8080,web2.internal:8080,web3.internal:8080?mode=2&log=info"

# Correct example: Client with 2 local database instances
nodepass "client://127.0.0.1:3306/db-primary.local:3306,db-secondary.local:3306?mode=1&log=warn"

# Incorrect example: Do not use multi-address for tunnel addresses (will cause parsing errors)
# nodepass "server://host1:10101,host2:10101/target:8080"  # ✗ Wrong usage
```

## URL Query Parameter Scope and Applicability

NodePass allows flexible configuration via URL query parameters. The following table shows which parameters are applicable in server, client, and master modes:

| Parameter | Description | Default | Accepted Values | server | client | master |
|-----------|-------------|---------|-----------------|:------:|:------:|:------:|
| `log` | Log level | `info` | `none`/`debug`/`info`/`warn`/`error`/`event` | O | O | O |
| `tls` | TLS encryption mode | `0` | `0`/`1`/`2` | O | X | O |
| `crt` | Custom certificate path | N/A | File path | O | X | O |
| `key` | Custom key path | N/A | File path | O | X | O |
| `dns` | DNS cache TTL | `5m` | `30s`/`5m`/`1h` etc. | O | O | X |
| `min` | Minimum pool capacity | `64` | Positive integer | X | O | X |
| `max` | Maximum pool capacity | `1024` | Positive integer | O | X | X |
| `mode` | Run mode control | `0` | `0`/`1`/`2` | O | O | X |
| `type` | Connection pool type | `0` | `0`/`1`/`2` | O | X | X |
| `dial` | Source IP for outbound | `auto` | `auto`/IP address | O | O | X |
| `read` | Data read timeout | `0` | `0`/`30s`/`5m` etc. | O | O | X |
| `rate` | Bandwidth rate limit | `0` | `0` or integer (Mbps) | O | O | X |
| `slot` | Maximum connection limit | `65536` | `0` or integer | O | O | X |
| `proxy` | PROXY protocol support | `0` | `0`/`1` | O | O | X |
| `notcp` | TCP support control | `0` | `0`/`1` | O | O | X |
| `noudp` | UDP support control | `0` | `0`/`1` | O | O | X |

- O: Parameter is valid and recommended for configuration
- X: Parameter is not applicable and should be ignored

**Best Practices:**
- For server/master modes, configure security-related parameters (`tls`, `crt`, `key`) to enhance data channel security.
- For client/server dual-end handshake modes, adjust connection pool capacity (`min`, `max`) based on traffic and resource constraints for optimal performance.
- Use run mode control (`mode`) when automatic detection doesn't match your deployment requirements or for consistent behavior across environments.
- Configure rate limiting (`rate`) to control bandwidth usage and prevent network congestion in shared environments.
- Configure QUIC transport (`quic`) on the server only - clients automatically receive the configuration during handshake.
- Set `notcp=1` when only UDP traffic needs to be tunneled to reduce resource usage and simplify configuration.
- Set `noudp=1` when only TCP traffic needs to be tunneled to reduce resource usage and simplify configuration.
- Log level (`log`) can be set in all modes for easier operations and troubleshooting.

## Environment Variables

NodePass behavior can be fine-tuned using environment variables. Below is the complete list of available variables with their descriptions, default values, and recommended settings for different scenarios.

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `NP_SEMAPHORE_LIMIT` | Signal channel buffer size | 65536 | `export NP_SEMAPHORE_LIMIT=2048` |
| `NP_TCP_DATA_BUF_SIZE` | Buffer size for TCP data transfer | 16384 | `export NP_TCP_DATA_BUF_SIZE=65536` |
| `NP_UDP_DATA_BUF_SIZE` | Buffer size for UDP packets | 16384 | `export NP_UDP_DATA_BUF_SIZE=16384` |
| `NP_HANDSHAKE_TIMEOUT` | Timeout for handshake operations | 5s | `export NP_HANDSHAKE_TIMEOUT=30s` |
| `NP_UDP_READ_TIMEOUT` | Timeout for UDP read operations | 30s | `export NP_UDP_READ_TIMEOUT=60s` |
| `NP_TCP_DIAL_TIMEOUT` | Timeout for establishing TCP connections | 5s | `export NP_TCP_DIAL_TIMEOUT=60s` |
| `NP_UDP_DIAL_TIMEOUT` | Timeout for establishing UDP connections | 5s | `export NP_UDP_DIAL_TIMEOUT=30s` |
| `NP_POOL_GET_TIMEOUT` | Timeout for getting connections from pool | 5s | `export NP_POOL_GET_TIMEOUT=60s` |
| `NP_MIN_POOL_INTERVAL` | Minimum interval between connection creations | 100ms | `export NP_MIN_POOL_INTERVAL=200ms` |
| `NP_MAX_POOL_INTERVAL` | Maximum interval between connection creations | 1s | `export NP_MAX_POOL_INTERVAL=3s` |
| `NP_REPORT_INTERVAL` | Interval for health check reports | 5s | `export NP_REPORT_INTERVAL=10s` |
| `NP_SERVICE_COOLDOWN` | Cooldown period before restart attempts | 3s | `export NP_SERVICE_COOLDOWN=5s` |
| `NP_SHUTDOWN_TIMEOUT` | Timeout for graceful shutdown | 5s | `export NP_SHUTDOWN_TIMEOUT=10s` |
| `NP_RELOAD_INTERVAL` | Interval for cert reload/state backup | 1h | `export NP_RELOAD_INTERVAL=30m` |

### Connection Pool Tuning

The connection pool parameters are important settings for performance tuning in dual-end handshake mode and do not apply to client single-end forwarding mode:

#### Pool Capacity Settings

- `min` (URL parameter): Ensures a minimum number of available connections
  - Too low: Increased latency during traffic spikes as new connections must be established
  - Too high: Wasted resources maintaining idle connections
  - Recommended starting point: 25-50% of your average concurrent connections

- `max` (URL parameter): Prevents excessive resource consumption while handling peak loads
  - Too low: Connection failures during traffic spikes
  - Too high: Potential resource exhaustion affecting system stability
  - Recommended starting point: 150-200% of your peak concurrent connections

#### Pool Interval Settings

- `NP_MIN_POOL_INTERVAL`: Controls the minimum time between connection creation attempts
  - Too low: May overwhelm network with connection attempts
  - Recommended range: 100ms-500ms depending on network latency and expected load

- `NP_MAX_POOL_INTERVAL`: Controls the maximum time between connection creation attempts
  - Too high: May result in pool depletion during traffic spikes
  - Recommended range: 1s-5s depending on expected traffic patterns

#### Connection Management

- `NP_SEMAPHORE_LIMIT`: Controls signal channel buffer size
  - Too small: May cause signal loss
  - Too large: Increased memory usage
  - Recommended range: 1000-5000

### UDP Settings

For applications relying heavily on UDP traffic:

- `NP_UDP_DATA_BUF_SIZE`: Buffer size for UDP packets
  - Increase for applications sending large UDP packets
  - Default (16384) works well for most cases
  - Consider increasing to 16384 or higher for media streaming or game servers

- `NP_UDP_READ_TIMEOUT`: Timeout for UDP read operations
  - Default (30s) is suitable for most UDP application scenarios
  - Controls the maximum wait time for UDP connections when no data is being transferred
  - For real-time applications (e.g., gaming, VoIP), consider reducing this value to quickly detect disconnections
  - For applications allowing intermittent transmission, increase this value to avoid false timeout detection

- `NP_UDP_DIAL_TIMEOUT`: Timeout for establishing UDP connections
  - Default (5s) provides good balance for most applications
  - Increase for high-latency networks or applications with slow response times
  - Decrease for low-latency applications requiring quick failover

### TCP Settings

For optimizing TCP connections:

- `NP_TCP_DATA_BUF_SIZE`: Buffer size for TCP data transfer
  - Default (32768) provides good balance for most applications
  - Increase for high-throughput applications requiring larger buffers
  - Consider increasing to 65536 or higher for bulk data transfers and streaming

- `NP_TCP_DIAL_TIMEOUT`: Timeout for establishing TCP connections
  - Default (5s) is suitable for most network conditions
  - Increase for unstable network conditions
  - Decrease for applications that need quick connection success/failure determination

### Pool Management Settings

- `NP_POOL_GET_TIMEOUT`: Maximum time to wait when getting connections from pool
  - Default (5s) provides sufficient time for connection establishment
  - Increase for high-latency environments or when using large pool sizes
  - Decrease for applications requiring fast failure detection
  - In client single-end forwarding mode, connection pools are not used and this parameter is ignored

### Service Management Settings

- `NP_REPORT_INTERVAL`: Controls how frequently health status is reported
  - Lower values provide more frequent updates but increase log volume
  - Higher values reduce log output but provide less immediate visibility

- `NP_RELOAD_INTERVAL`: Controls how frequently TLS certificates are checked for changes and state backups are performed
  - Lower values provide faster certificate change detection and more frequent backups but increase file system operations
  - Higher values reduce overhead but delay certificate updates and backup frequency

- `NP_SERVICE_COOLDOWN`: Time to wait before attempting service restarts
  - Lower values attempt recovery faster but might cause thrashing in case of persistent issues
  - Higher values provide more stability but slower recovery from transient issues

- `NP_SHUTDOWN_TIMEOUT`: Maximum time to wait for connections to close during shutdown
  - Lower values ensure quicker shutdown but may interrupt active connections
  - Higher values allow more time for connections to complete but delay shutdown

## Configuration Profiles

Here are some recommended environment variable configurations for common scenarios:

### High-Throughput Configuration

For applications requiring maximum throughput (e.g., media streaming, file transfers):

URL parameters:
```bash
# High-throughput server with 1 Gbps rate limit and high connection capacity
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?max=8192&rate=1000&slot=10000"

# High-throughput client with 500 Mbps rate limit and high connection capacity
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=128&rate=500&slot=5000"
```

Environment variables:
```bash
export NP_MIN_POOL_INTERVAL=50ms
export NP_MAX_POOL_INTERVAL=500ms
export NP_SEMAPHORE_LIMIT=8192
export NP_TCP_DATA_BUF_SIZE=65536
export NP_UDP_DATA_BUF_SIZE=32768
export NP_POOL_GET_TIMEOUT=60s
export NP_REPORT_INTERVAL=10s
```

### Low-Latency Configuration

For applications requiring minimal latency (e.g., gaming, financial trading):

URL parameters:
```bash
# Low-latency server with moderate rate limit and moderate connection limit
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?max=4096&rate=200&slot=3000"

# Low-latency client with moderate rate limit and moderate connection limit
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=256&rate=200&slot=2000"
```

Environment variables:
```bash
export NP_MIN_POOL_INTERVAL=50ms
export NP_MAX_POOL_INTERVAL=500ms
export NP_SEMAPHORE_LIMIT=4096
export NP_TCP_DIAL_TIMEOUT=5s
export NP_UDP_DIAL_TIMEOUT=5s
export NP_POOL_GET_TIMEOUT=15s
export NP_REPORT_INTERVAL=1s
```

### Resource-Constrained Configuration

For deployment on systems with limited resources (e.g., IoT devices, small VPS):

URL parameters:
```bash
# Resource-constrained server with conservative rate limit and low connection limit
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?max=512&rate=50&slot=500"

# Resource-constrained client with conservative rate limit and low connection limit
nodepass "client://server.example.com:10101/127.0.0.1:8080?min=16&rate=50&slot=200"
```

Environment variables:
```bash
export NP_MIN_POOL_INTERVAL=200ms
export NP_MAX_POOL_INTERVAL=2s
export NP_SEMAPHORE_LIMIT=512
export NP_TCP_DIAL_TIMEOUT=20s
export NP_UDP_DIAL_TIMEOUT=20s
export NP_POOL_GET_TIMEOUT=45s
export NP_REPORT_INTERVAL=30s
export NP_SHUTDOWN_TIMEOUT=3s
```

## Next Steps

- See [usage instructions](/docs/en/usage.md) for basic operational commands
- Explore [examples](/docs/en/examples.md) to understand deployment patterns
- Learn about [how NodePass works](/docs/en/how-it-works.md) to optimize your configuration
- Check the [troubleshooting guide](/docs/en/troubleshooting.md) if you encounter issues