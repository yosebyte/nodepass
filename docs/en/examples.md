# Usage Examples

This page provides practical examples of NodePass in various deployment scenarios. These examples cover common use cases and can be adapted to suit your specific requirements.

## Basic Server Setup with TLS Options

### Example 1: No TLS Encryption

When speed is more important than security (e.g., in trusted networks):

```bash
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=0"
```

This starts a NodePass server that:
- Listens for tunnel connections on all interfaces, port 10101
- Forwards traffic to localhost:8080
- Uses debug logging for detailed information
- Uses no encryption for data channels (fastest performance)

### Example 2: Self-Signed Certificate

For balanced security and ease of setup (recommended for most cases):

```bash
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1"
```

This configuration:
- Automatically generates a self-signed certificate
- Provides encryption without requiring certificate management
- Protects data traffic from passive eavesdropping
- Works well for internal or testing environments

### Example 3: Custom Domain Certificate

For production environments requiring verified certificates:

```bash
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

This setup:
- Uses your provided TLS certificate and private key
- Offers the highest security level with certificate validation
- Is ideal for production environments and public-facing services
- Allows clients to verify the server's identity

## Connecting to a NodePass Server

### Example 4: Basic Client Connection

Connect to a NodePass server with default settings:

```bash
nodepass client://server.example.com:10101/127.0.0.1:8080
```

This client:
- Connects to the NodePass server at server.example.com:10101
- Forwards received traffic to localhost:8080
- Automatically adopts the server's TLS security policy
- Uses the default info log level

### Example 5: Client with Debug Logging

For troubleshooting connection issues:

```bash
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

This enables verbose output to help identify:
- Connection establishment issues
- Signal processing
- Data transfer details
- Error conditions

### Example 6: Run Mode Control

Control the operational behavior with explicit mode settings:

```bash
# Force server to operate in reverse mode (server receives traffic)
nodepass "server://0.0.0.0:10101/0.0.0.0:8080?mode=1&tls=1"

# Force client to operate in single-end forwarding mode (high performance local proxy)
nodepass "client://127.0.0.1:1080/remote.example.com:8080?mode=1"

# Force client to operate in dual-end handshake mode (requires server coordination)
nodepass "client://server.example.com:10101/127.0.0.1:8080?mode=2&log=debug"
```

These configurations:
- **Server mode=1**: Forces reverse mode where server binds to target address locally
- **Client mode=1**: Forces single-end forwarding with direct connection establishment for high performance
- **Client mode=2**: Forces dual-end handshake mode for scenarios requiring server coordination
- Use mode control when automatic detection doesn't match your deployment requirements

## Database Access Through Firewall

### Example 7: Database Tunneling

Enable secure access to a database server behind a firewall:

```bash
# Server side (outside secured network) with TLS encryption
nodepass server://:10101/127.0.0.1:5432?tls=1

# Client side (inside the firewall)
nodepass client://server.example.com:10101/127.0.0.1:5432
```

This configuration:
- Creates an encrypted tunnel to a PostgreSQL database (port 5432)
- Allows secure access to the database without exposing it directly to the internet
- Encrypts all database traffic with a self-signed certificate
- Maps the remote database to appear as a local service on the client side

## Secure Microservice Communication

### Example 8: Service-to-Service Communication

Enable secure communication between microservices:

```bash
# Service A (consuming API) with custom certificate
nodepass "server://0.0.0.0:10101/127.0.0.1:8081?log=warn&tls=2&crt=/path/to/service-a.crt&key=/path/to/service-a.key"

# Service B (providing API)
nodepass client://service-a:10101/127.0.0.1:8082
```

This setup:
- Creates a secure channel between two microservices
- Uses a custom certificate for service identity verification
- Limits logging to warnings and errors only
- Maps service A's API to appear as a local service on service B

## Bandwidth Rate Limiting

### Example 9: File Transfer Server with Rate Limit

Control bandwidth usage for file transfer services:

```bash
# Server side: Limit bandwidth to 100 Mbps for file transfers
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=info&tls=1&rate=100"

# Client side: Connect with 50 Mbps rate limit
nodepass "client://fileserver.example.com:10101/127.0.0.1:3000?log=info&rate=50"
```

This configuration:
- Limits server bandwidth to 100 Mbps to prevent network congestion
- Client further limits download speed to 50 Mbps for fair sharing
- Allows file transfers while preserving bandwidth for other services
- Uses TLS encryption for secure file transfer

### Example 10: IoT Sensor Data Collection with Conservative Limits

For IoT devices with limited bandwidth or metered connections:

```bash
# Server: Accept IoT data with 5 Mbps limit
nodepass "server://0.0.0.0:10101/127.0.0.1:1883?log=warn&rate=5"

# IoT device client: Send sensor data with 2 Mbps limit  
nodepass "client://iot-gateway.example.com:10101/127.0.0.1:1883?log=error&rate=2"
```

This setup:
- Limits server to 5 Mbps for collecting sensor data from multiple IoT devices
- Individual IoT clients limited to 2 Mbps to prevent single device consuming all bandwidth
- Minimal logging (warn/error) to reduce resource usage on IoT devices
- Efficient for MQTT or other IoT protocols

### Example 11: Development Environment Rate Control

Testing applications under bandwidth constraints:

```bash
# Simulate slow network conditions for testing
nodepass "client://api.example.com:443/127.0.0.1:8080?log=debug&rate=1"

# High-speed development server with monitoring
nodepass "server://0.0.0.0:10101/127.0.0.1:3000?log=debug&rate=500"
```

This configuration:
- Client simulation of 1 Mbps connection for testing slow network scenarios
- Development server with 500 Mbps limit and detailed logging for debugging
- Helps identify performance issues under different bandwidth constraints

## IoT Device Management

### Example 12: IoT Gateway

Create a central access point for IoT devices:

```bash
# Central management server
nodepass "server://0.0.0.0:10101/127.0.0.1:8888?log=info&tls=1"

# IoT device
nodepass client://mgmt.example.com:10101/127.0.0.1:80
```

This configuration:
- Enables secure connections from distributed IoT devices to a central server
- Uses self-signed certificates for adequate security
- Allows embedded devices to expose their local web interfaces securely
- Centralizes device management through a single endpoint

## Multi-environment Development

### Example 13: Development Environment Access

Access different development environments through tunnels:

```bash
# Production API access tunnel
nodepass client://tunnel.example.com:10101/127.0.0.1:3443

# Development environment
nodepass server://tunnel.example.com:10101/127.0.0.1:3000

# Testing environment
nodepass "server://tunnel.example.com:10101/127.0.0.1:3001?log=warn&tls=1"
```

This setup:
- Creates secure access to multiple environments (production, development, testing)
- Uses different levels of logging based on environment sensitivity
- Enables developers to access environments without direct network exposure
- Maps remote services to different local ports for easy identification

## PROXY Protocol Integration

### Example 14: Load Balancer Integration with PROXY Protocol

Enable PROXY protocol support for integration with load balancers and reverse proxies:

```bash
# Server side: Enable PROXY protocol v1 for HAProxy/Nginx integration
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=info&tls=1&proxy=1"

# Client side: Enable PROXY protocol to preserve client connection information
nodepass "client://tunnel.example.com:10101/127.0.0.1:3000?log=info&proxy=1"
```

This configuration:
- Sends PROXY protocol v1 headers before data transfer begins
- Preserves original client IP and port information through the tunnel
- Enables backend services to see real client connection details
- Compatible with HAProxy, Nginx, and other PROXY protocol aware services
- Useful for maintaining accurate access logs and IP-based access controls

### Example 15: Reverse Proxy Support for Web Applications

Enable web applications behind NodePass to receive original client information:

```bash
# NodePass server with PROXY protocol for web application
nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=warn&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem&proxy=1"

# Backend web server (e.g., Nginx) configuration to handle PROXY protocol
# In nginx.conf:
# server {
#     listen 8080 proxy_protocol;
#     real_ip_header proxy_protocol;
#     set_real_ip_from 127.0.0.1;
#     ...
# }
```

This setup:
- Web applications receive original client IP addresses instead of NodePass tunnel IP
- Enables proper access logging, analytics, and security controls
- Supports compliance requirements for connection auditing
- Works with web servers that support PROXY protocol (Nginx, HAProxy, etc.)

### Example 16: Database Access with Client IP Preservation

Maintain client IP information for database access logging and security:

```bash
# Database proxy server with PROXY protocol
nodepass "server://0.0.0.0:10101/127.0.0.1:5432?log=error&proxy=1"

# Application client connecting through tunnel
nodepass "client://dbproxy.example.com:10101/127.0.0.1:5432?proxy=1"
```

Benefits:
- Database logs show original application server IPs instead of tunnel IPs
- Enables IP-based database access controls to work properly
- Maintains audit trails for security and compliance
- Compatible with databases that support PROXY protocol (PostgreSQL with appropriate configuration)

**Important Notes for PROXY Protocol:**
- Target services must support PROXY protocol v1 to handle the headers correctly
- PROXY headers are only sent for TCP connections, not UDP traffic
- The header includes: protocol (TCP4/TCP6), source IP, destination IP, source port, destination port
- If target service doesn't support PROXY protocol, connections may fail or behave unexpectedly
- Test thoroughly in non-production environments before deploying with PROXY protocol enabled

## Container Deployment

### Example 17: Containerized NodePass

Deploy NodePass in a Docker environment:

```bash
# Create a network for the containers
docker network create nodepass-net

# Deploy NodePass server with self-signed certificate
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

This configuration:
- Creates a containerized tunnel between services
- Uses Docker networking to connect containers
- Exposes only necessary ports to the host
- Provides secure access to an internal web service

## Master API Management

### Example 18: Centralized Management

Set up a central controller for multiple NodePass instances:

```bash
# Start the master API service with self-signed certificate
nodepass "master://0.0.0.0:9090?log=info&tls=1"
```

You can then manage instances via API calls:

```bash
# Create a server instance
curl -X POST http://localhost:9090/api/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'

# Create a client instance
curl -X POST http://localhost:9090/api/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"client://localhost:10101/127.0.0.1:8081"}'

# List all running instances
curl http://localhost:9090/api/v1/instances

# Control an instance (replace {id} with actual instance ID)
curl -X PUT http://localhost:9090/api/v1/instances/{id} \
  -H "Content-Type: application/json" \
  -d '{"action":"restart"}'
```

This setup:
- Provides a central management interface for all NodePass instances
- Allows dynamic creation and control of tunnels
- Offers a RESTful API for automation and integration
- Includes a built-in Swagger UI at http://localhost:9090/api/v1/docs

### Example 19: Custom API Prefix

Use a custom API prefix for the master mode:

```bash
# Start with custom API prefix
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"

# Create an instance using the custom prefix
curl -X POST http://localhost:9090/admin/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'
```

This allows:
- Integration with existing API gateways
- Custom URL paths for security or organizational purposes
- Swagger UI access at http://localhost:9090/admin/v1/docs

### Example 20: Real-time Connection and Traffic Monitoring

Monitor instance connection counts and traffic statistics through the master API:

```bash
# Get detailed instance information including connection count statistics
curl -H "X-API-Key: your-api-key" http://localhost:9090/api/v1/instances/{id}

# Example response (including TCPS and UDPS fields)
{
  "id": "a1b2c3d4",
  "alias": "web-proxy",
  "type": "server",
  "status": "running", 
  "url": "server://0.0.0.0:10101/127.0.0.1:8080",
  "restart": true,
  "pool": 64,
  "ping": 25,
  "tcps": 12,
  "udps": 5,
  "tcprx": 1048576,
  "tcptx": 2097152,
  "udprx": 512000,
  "udptx": 256000
}

# Use SSE to monitor real-time status changes for all instances
curl -H "X-API-Key: your-api-key" \
  -H "Accept: text/event-stream" \
  http://localhost:9090/api/v1/events
```

This monitoring setup provides:
- **Real-time connection tracking**: TCPS and UDPS fields show current active connection counts
- **Performance analysis**: Evaluate system load through connection and traffic data
- **Capacity planning**: Resource planning based on historical connection data
- **Troubleshooting**: Abnormal connection count changes may indicate network issues

## Next Steps

Now that you've seen various usage examples, you might want to:

- Learn about [configuration options](/docs/en/configuration.md) for fine-tuning
- Understand [how NodePass works](/docs/en/how-it-works.md) under the hood
- Check the [troubleshooting guide](/docs/en/troubleshooting.md) for common issues