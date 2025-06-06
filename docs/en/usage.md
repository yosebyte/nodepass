# Usage Instructions

NodePass creates tunnels with an unencrypted TCP control channel and configurable TLS #### Examples

```bash
# Client single-end forwarding mode - Local proxy listening on port 1080, forwarding to target server
nodepass client://127.0.0.1:1080/target.example.com:8080?log=debug

# Connect to NodePass server and automatically adopt its TLS security policy - Client sends mode
nodepass client://server.example.com:10101/127.0.0.1:8080

# Connect with debug logging - Client receives mode
nodepass client://server.example.com:10101/192.168.1.100:8080?log=debug
```n options for data exchange. This guide covers the three operating modes and explains how to use each effectively.

## Command Line Syntax

The general syntax for NodePass commands is:

```bash
nodepass <core>://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

Where:
- `<core>`: Specifies the operating mode (`server`, `client`, or `master`)
- `<tunnel_addr>`: The tunnel endpoint address for control channel communications 
- `<target_addr>`: The destination address for business data with bidirectional flow support (or API prefix in master mode)
- `<level>`: Log verbosity level (`debug`, `info`, `warn`, `error`, or `event`)
- `<mode>`: TLS security level for data channels (`0`, `1`, or `2`) - server/master modes only
- `<cert_file>`: Path to certificate file (when `tls=2`) - server/master modes only
- `<key_file>`: Path to private key file (when `tls=2`) - server/master modes only

## Operating Modes

NodePass offers three complementary operating modes to suit various deployment scenarios.

### Server Mode

Server mode establishes tunnel control channels and supports bidirectional data flow forwarding.

```bash
nodepass server://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

#### Parameters

- `tunnel_addr`: Address for the TCP tunnel endpoint (control channel) that clients will connect to (e.g., 10.1.0.1:10101)
- `target_addr`: The destination address for business data with bidirectional flow support (e.g., 10.1.0.1:8080)
- `log`: Log level (debug, info, warn, error, event)
- `tls`: TLS encryption mode for the target data channel (0, 1, 2)
  - `0`: No TLS encryption (plain TCP/UDP)
  - `1`: Self-signed certificate (automatically generated)
  - `2`: Custom certificate (requires `crt` and `key` parameters)
- `crt`: Path to certificate file (required when `tls=2`)
- `key`: Path to private key file (required when `tls=2`)

#### How Server Mode Works

In server mode, NodePass supports two data flow directions:

**Mode 1: Server Receives Traffic** (target_addr is local address)
1. Listens for TCP tunnel connections (control channel) on `tunnel_addr`
2. Listens for incoming TCP and UDP traffic on `target_addr` 
3. When a connection arrives at `target_addr`, it signals the connected client through the control channel
4. Creates a data channel for each connection with the specified TLS encryption level

**Mode 2: Server Sends Traffic** (target_addr is remote address)
1. Listens for TCP tunnel connections (control channel) on `tunnel_addr`
2. Waits for clients to listen locally and receive connections through the tunnel
3. Establishes connections to remote `target_addr` and forwards data

#### Examples

```bash
# No TLS encryption for data channel - Server receives mode
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=0"

# Self-signed certificate (auto-generated) - Server sends mode
nodepass "server://10.1.0.1:10101/192.168.1.100:8080?log=debug&tls=1"

# Custom domain certificate - Server receives mode
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### Client Mode

Client mode connects to a NodePass server and supports bidirectional data flow forwarding.

```bash
nodepass client://<tunnel_addr>/<target_addr>?log=<level>
```

#### Parameters

- `tunnel_addr`: Address of the NodePass server's tunnel endpoint to connect to (e.g., 10.1.0.1:10101)
- `target_addr`: The destination address for business data with bidirectional flow support (e.g., 127.0.0.1:8080)
- `log`: Log level (debug, info, warn, error, event)

#### How Client Mode Works

In client mode, NodePass supports three operating modes:

**Mode 1: Client Single-End Forwarding** (when tunnel address is local)
1. Listens for TCP and UDP connections on the local tunnel address
2. Uses connection pooling technology to pre-establish TCP connections to target address, eliminating connection latency
3. Directly forwards received traffic to the target address with high performance
4. No handshake with server required, enables point-to-point direct forwarding
5. Suitable for local proxy and simple forwarding scenarios

**Mode 2: Client Receives Traffic** (when server sends traffic)
1. Connects to the server's TCP tunnel endpoint (control channel)
2. Listens locally and waits for connections through the tunnel
3. Establishes connections to local `target_addr` and forwards data

**Mode 3: Client Sends Traffic** (when server receives traffic)
1. Connects to the server's TCP tunnel endpoint (control channel)
2. Listens for signals from the server through this control channel
3. When a signal is received, establishes a data connection with the TLS security level specified by the server
4. Creates a connection to `target_addr` and forwards traffic

#### Examples

```bash
# Connect to a NodePass server and automatically adopt its TLS security policy - Client sends mode
nodepass client://server.example.com:10101/127.0.0.1:8080

# Connect with debug logging - Client receives mode
nodepass client://server.example.com:10101/192.168.1.100:8080?log=debug
```

### Master Mode (API)

Master mode runs a RESTful API server for centralized management of NodePass instances.

```bash
nodepass master://<api_addr>[<prefix>]?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

#### Parameters

- `api_addr`: Address where the API service will listen (e.g., 0.0.0.0:9090)
- `prefix`: Optional API prefix path (e.g., /management). Default is `/api`
- `log`: Log level (debug, info, warn, error, event)
- `tls`: TLS encryption mode for the API service (0, 1, 2)
  - `0`: No TLS encryption (HTTP)
  - `1`: Self-signed certificate (HTTPS with auto-generated cert)
  - `2`: Custom certificate (HTTPS with provided cert)
- `crt`: Path to certificate file (required when `tls=2`)
- `key`: Path to private key file (required when `tls=2`)

#### How Master Mode Works

In master mode, NodePass:
1. Runs a RESTful API server that allows dynamic management of NodePass instances
2. Provides endpoints for creating, starting, stopping, and monitoring client and server instances
3. Includes Swagger UI for easy API exploration at `{prefix}/v1/docs`
4. Automatically inherits TLS and logging settings for instances created through the API

#### API Endpoints

All endpoints are relative to the configured prefix (default: `/api`):

- `GET {prefix}/v1/instances` - List all instances
- `POST {prefix}/v1/instances` - Create a new instance with JSON body: `{"url": "server://0.0.0.0:10101/0.0.0.0:8080"}`
- `GET {prefix}/v1/instances/{id}` - Get instance details
- `PUT {prefix}/v1/instances/{id}` - Update instance with JSON body: `{"action": "start|stop|restart"}`
- `DELETE {prefix}/v1/instances/{id}` - Delete instance
- `GET {prefix}/v1/openapi.json` - OpenAPI specification
- `GET {prefix}/v1/docs` - Swagger UI documentation

#### Examples

```bash
# Start master with HTTP using default API prefix (/api)
nodepass "master://0.0.0.0:9090?log=info"

# Start master with custom API prefix (/management)
nodepass "master://0.0.0.0:9090/management?log=info"

# Start master with HTTPS (self-signed certificate)
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"

# Start master with HTTPS (custom certificate)
nodepass "master://0.0.0.0:9090?log=info&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

## Managing NodePass Instances

### Creating and Managing via API

You can use standard HTTP requests to manage NodePass instances through the master API:

```bash
# Create and manage instances via API (using default prefix)
curl -X POST http://localhost:9090/api/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'

# Using custom prefix
curl -X POST http://localhost:9090/admin/v1/instances \
  -H "Content-Type: application/json" \
  -d '{"url":"server://0.0.0.0:10101/0.0.0.0:8080?tls=1"}'

# List all running instances
curl http://localhost:9090/api/v1/instances

# Control an instance (replace {id} with actual instance ID)
curl -X PUT http://localhost:9090/api/v1/instances/{id} \
  -H "Content-Type: application/json" \
  -d '{"action":"restart"}'
```

## Bidirectional Data Flow Explanation

NodePass supports flexible bidirectional data flow configuration:

### Client Single-End Forwarding Mode
- **Client**: Listens on local tunnel address, uses connection pooling technology to directly forward to target address
- **Connection Pool Optimization**: Pre-establishes TCP connections, eliminates connection latency, provides high-performance forwarding
- **No Server Required**: Operates independently without server handshake
- **Use Case**: Local proxy, simple port forwarding, testing environments, high-performance forwarding

### Server Receives Mode (dataFlow: "-")
- **Server**: Listens for incoming connections on target_addr, forwards through tunnel to client
- **Client**: Connects to local target_addr to provide services
- **Use Case**: Expose internal services to external access

### Server Sends Mode (dataFlow: "+")  
- **Server**: Connects to remote target_addr to fetch data, sends through tunnel to client
- **Client**: Listens locally to receive connections from server
- **Use Case**: Access remote services through tunnel proxy

The system automatically selects the appropriate operation mode based on tunnel and target addresses:
- If the client's tunnel address is a local address, enables single-end forwarding mode
- If target address is a local address, uses Server Receives Mode
- If target address is a remote address, uses Server Sends Mode

## Next Steps

- Learn about [configuration options](/docs/en/configuration.md) to fine-tune NodePass
- Explore [examples](/docs/en/examples.md) of common deployment scenarios
- Understand [how NodePass works](/docs/en/how-it-works.md) under the hood
- Check the [troubleshooting guide](/docs/en/troubleshooting.md) if you encounter issues