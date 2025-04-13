# Usage Instructions

NodePass creates tunnels with an unencrypted TCP control channel and configurable TLS encryption options for data exchange. This guide covers the three operating modes and explains how to use each effectively.

## Command Line Syntax

The general syntax for NodePass commands is:

```bash
nodepass <core>://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

Where:
- `<core>`: Specifies the operating mode (`server`, `client`, or `master`)
- `<tunnel_addr>`: The tunnel endpoint address for control channel communications 
- `<target_addr>`: The destination address for forwarded traffic (or API prefix in master mode)
- `<level>`: Log verbosity level (`debug`, `info`, `warn`, `error`, or `fatal`)
- `<mode>`: TLS security level for data channels (`0`, `1`, or `2`) - server/master modes only
- `<cert_file>`: Path to certificate file (when `tls=2`) - server/master modes only
- `<key_file>`: Path to private key file (when `tls=2`) - server/master modes only

## Operating Modes

NodePass offers three complementary operating modes to suit various deployment scenarios.

### Server Mode

Server mode listens for client connections and forwards traffic from a target address through the tunnel.

```bash
nodepass server://<tunnel_addr>/<target_addr>?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

#### Parameters

- `tunnel_addr`: Address for the TCP tunnel endpoint (control channel) that clients will connect to (e.g., 10.1.0.1:10101)
- `target_addr`: Address where the server listens for incoming connections (TCP and UDP) that will be tunneled to clients (e.g., 10.1.0.1:8080)
- `log`: Log level (debug, info, warn, error, fatal)
- `tls`: TLS encryption mode for the target data channel (0, 1, 2)
  - `0`: No TLS encryption (plain TCP/UDP)
  - `1`: Self-signed certificate (automatically generated)
  - `2`: Custom certificate (requires `crt` and `key` parameters)
- `crt`: Path to certificate file (required when `tls=2`)
- `key`: Path to private key file (required when `tls=2`)

#### How Server Mode Works

In server mode, NodePass:
1. Listens for TCP tunnel connections (control channel) on `tunnel_addr`
2. Listens for incoming TCP and UDP traffic on `target_addr` 
3. When a connection arrives at `target_addr`, it signals the connected client through the unencrypted TCP tunnel
4. Creates a data channel for each connection with the specified TLS encryption level

#### Examples

```bash
# No TLS encryption for data channel
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=0"

# Self-signed certificate (auto-generated)
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=1"

# Custom domain certificate
nodepass "server://10.1.0.1:10101/10.1.0.1:8080?log=debug&tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem"
```

### Client Mode

Client mode connects to a NodePass server and forwards traffic to a local target address.

```bash
nodepass client://<tunnel_addr>/<target_addr>?log=<level>
```

#### Parameters

- `tunnel_addr`: Address of the NodePass server's tunnel endpoint to connect to (e.g., 10.1.0.1:10101)
- `target_addr`: Local address where traffic will be forwarded to (e.g., 127.0.0.1:8080)
- `log`: Log level (debug, info, warn, error, fatal)

#### How Client Mode Works

In client mode, NodePass:
1. Connects to the server's unencrypted TCP tunnel endpoint (control channel) at `tunnel_addr`
2. Listens for signals from the server through this control channel
3. When a signal is received, establishes a data connection with the TLS security level specified by the server
4. Creates a local connection to `target_addr` and forwards traffic

#### Examples

```bash
# Connect to a NodePass server and automatically adopt its TLS security policy
nodepass client://server.example.com:10101/127.0.0.1:8080

# Connect with debug logging
nodepass client://server.example.com:10101/127.0.0.1:8080?log=debug
```

### Master Mode (API)

Master mode runs a RESTful API server for centralized management of NodePass instances.

```bash
nodepass master://<api_addr>[<prefix>]?log=<level>&tls=<mode>&crt=<cert_file>&key=<key_file>
```

#### Parameters

- `api_addr`: Address where the API service will listen (e.g., 0.0.0.0:9090)
- `prefix`: Optional API prefix path (e.g., /management). Default is `/api`
- `log`: Log level (debug, info, warn, error, fatal)
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

## Next Steps

- Learn about [configuration options](/docs/en/configuration.md) to fine-tune NodePass
- Explore [examples](/docs/en/examples.md) of common deployment scenarios
- Understand [how NodePass works](/docs/en/how-it-works.md) under the hood
- Check the [troubleshooting guide](/docs/en/troubleshooting.md) if you encounter issues