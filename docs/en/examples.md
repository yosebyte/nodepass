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

## Database Access Through Firewall

### Example 6: Database Tunneling

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

### Example 7: Service-to-Service Communication

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

## IoT Device Management

### Example 8: IoT Gateway

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

### Example 9: Development Environment Access

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

## Container Deployment

### Example 10: Containerized NodePass

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

### Example 11: Centralized Management

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

### Example 12: Custom API Prefix

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

## Next Steps

Now that you've seen various usage examples, you might want to:

- Learn about [configuration options](/docs/en/configuration.md) for fine-tuning
- Understand [how NodePass works](/docs/en/how-it-works.md) under the hood
- Check the [troubleshooting guide](/docs/en/troubleshooting.md) for common issues