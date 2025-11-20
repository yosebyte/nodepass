# NodePass AI Coding Agent Instructions

## Project Overview

NodePass is an enterprise-grade TCP/UDP network tunneling solution with a three-tier S/C/M architecture supporting server, client, and master modes. Written in Go 1.25+, focused on performance, security, and zero-configuration deployment.

## Architecture Essentials

### Three-Tier S/C/M Architecture

1. **Server Mode** (`internal/server.go`): Accepts tunnel connections, manages connection pools, forwards traffic bidirectionally
2. **Client Mode** (`internal/client.go`): Connects to servers, supports single-end forwarding or dual-end handshake modes
3. **Master Mode** (`internal/master.go`): RESTful API for dynamic instance management with persistent state in `nodepass.gob`

### Critical Design Patterns

- **Separation of Control/Data Channels**: 
  - Control channel: Unencrypted TCP for signaling (`np://` scheme with fragments)
  - Data channel: Configurable TLS (modes 0/1/2) for actual traffic
  
- **Connection Pooling**: Pre-established connections via `github.com/NodePassProject/pool` library
  - Server controls `max` pool capacity, passes to client during handshake
  - Client manages `min` capacity for persistent connections
  - QUIC multiplexing available as alternative transport (`quic=1`)
  
- **Bidirectional Data Flow**: Automatic mode detection in `Common.runMode`
  - Mode 0: Auto-detect based on target address bindability
  - Mode 1: Reverse/single-end (server receives OR client listens locally)
  - Mode 2: Forward/dual-end (server sends OR client connects remotely)

### Key Components
- `/cmd/nodepass/main.go`: Entry point, version variable injection
- `/cmd/nodepass/core.go`: Mode dispatch, TLS setup, certificate hot-reload
- `/internal/common.go`: Shared primitives (buffer pools, slot management, DNS resolution, encoding)
- `/internal/{server,client,master}.go`: Mode-specific implementations inheriting `Common`

### External Dependencies (NodePassProject Ecosystem)

All critical networking primitives are in separate libraries:
- `github.com/NodePassProject/cert`: TLS certificate generation and management
- `github.com/NodePassProject/conn`: Enhanced connections (`StatConn` with traffic tracking)
- `github.com/NodePassProject/logs`: Multi-level logger (None/Debug/Info/Warn/Error/Event)
- `github.com/NodePassProject/name`: DNS resolver with caching and background refresh
- `github.com/NodePassProject/pool`: TCP connection pooling with auto-scaling
- `github.com/NodePassProject/quic`: QUIC multiplexing for 0-RTT connections

**Never modify these libraries directly** - they're external dependencies. Use their exported APIs only.

## Configuration System

### URL-Based Configuration

All modes use URL-style configuration: `scheme://[password@]host:port/target?param=value`

```bash
# Server: bind_addr/target_addr with pool capacity and TLS
server://password@0.0.0.0:10101/127.0.0.1:8080?max=1024&tls=1&log=debug

# Client: server_addr/local_addr with min capacity and mode
client://password@server:10101/127.0.0.1:9090?min=128&mode=0&rate=100

# Master: api_addr/prefix with TLS and custom certs
master://0.0.0.0:9090/api?log=info&tls=2&crt=/path/cert.pem&key=/path/key.pem
```

### Query Parameters

- `log`: none|debug|info|warn|error|event (default: info)
- `tls`: 0=plain, 1=self-signed, 2=custom cert (server/master only, client inherits from server)
- `min`/`max`: Connection pool capacity (client sets min, server sets max)
- `mode`: 0=auto, 1=reverse/single-end, 2=forward/dual-end
- `quic`: 0=TCP pool, 1=QUIC multiplexing (requires tls≥1)
- `dns`: Custom DNS servers (comma-separated, default: 1.1.1.1,8.8.8.8)
- `read`: Timeout duration (e.g., 1h, 30m, 15s, default: 0=no timeout)
- `rate`: Mbps bandwidth limit (0=unlimited)
- `slot`: Max concurrent connections (default: 65536)
- `proxy`: PROXY protocol v1 support (0=off, 1=on)
- `dial`: Local bind IP for outbound connections (default: auto)
- `notcp`/`noudp`: Disable TCP/UDP (0=enabled, 1=disabled)

### Environment Variables for Tuning

Runtime behavior tunable without recompilation (see `internal/common.go`):
```go
NP_TCP_DATA_BUF_SIZE=16384      // TCP buffer size
NP_UDP_DATA_BUF_SIZE=16384      // UDP buffer size  
NP_HANDSHAKE_TIMEOUT=5s         // Handshake timeout
NP_POOL_GET_TIMEOUT=5s          // Pool connection acquisition timeout
NP_REPORT_INTERVAL=5s           // Health check reporting interval
NP_RELOAD_INTERVAL=1h           // TLS cert hot-reload interval (mode 2)
NP_SEMAPHORE_LIMIT=65536        // Signal channel buffer size
NP_DNS_CACHING_TTL=5m           // DNS cache TTL
```

## Development Workflow

### Building

```bash
# Development build
go build -o nodepass ./cmd/nodepass

# Release build with version injection (mimics .goreleaser.yml)
go build -trimpath -ldflags="-s -w -X main.version=1.0.0" -o nodepass ./cmd/nodepass
```

### Testing Manually

No automated test suite exists currently. Test via real-world scenarios:

```bash
# Terminal 1: Server with debug logging and self-signed TLS
./nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1&max=256"

# Terminal 2: Client connecting to server
./nodepass "client://localhost:10101/127.0.0.1:9090?log=debug&min=64"

# Terminal 3: Master API mode
./nodepass "master://0.0.0.0:9090/api?log=debug&tls=0"
```

**Test checklist** (from CONTRIBUTING.md):
1. Test each mode (server, client, master) with `log=debug`
2. Verify TCP and UDP forwarding separately
3. Test all TLS modes (0, 1, 2) with certificate validation
4. Test QUIC mode (`quic=1`) with TLS≥1
5. Verify graceful shutdown with SIGTERM/SIGINT
6. Stress test with high concurrency and connection pool scaling

### Docker Build

```bash
docker build --build-arg VERSION=dev -t nodepass:dev .
```

### Release Process

Uses GoReleaser on tag push (`v*.*.*`). See `.goreleaser.yml` for build matrix (Linux, Windows, macOS, FreeBSD across multiple architectures).

## Code Patterns and Conventions

### Error Handling

Always wrap errors with context using `fmt.Errorf("function: action failed: %w", err)`. See pattern in `start()`, `createCore()`, `NewServer()`, etc.

### Logging

Use the injected `logger` instance with appropriate levels:
```go
logger.Debug("Detailed info: %v", detail)  // Verbose debugging
logger.Info("Operation: %v", status)       // Normal operations
logger.Warn("Non-critical issue: %v", err) // Recoverable problems
logger.Error("Critical error: %v", err)    // Functionality affected
logger.Event("Traffic stats: %v", stats)   // Important events
```

### Goroutine Management

All long-running goroutines must:
1. Check `ctx.Err()` regularly for cancellation
2. Use proper cleanup with `defer` statements
3. Handle panics in critical sections
4. Release resources (slots, buffers, connections) on exit

### Buffer Management

Use sync.Pool for TCP/UDP buffers to reduce GC pressure:
```go
buf := c.getTCPBuffer()  // Gets []byte from tcpBufferPool
defer c.putTCPBuffer(buf)
```

### Slot Management

Connection slots prevent resource exhaustion:
```go
if !c.tryAcquireSlot(isUDP) {
    return fmt.Errorf("slot limit reached")
}
defer c.releaseSlot(isUDP)
```

### Configuration via Environment Variables

Runtime behavior tunable without recompilation:
```go
var tcpDataBufSize = getEnvAsInt("NP_TCP_DATA_BUF_SIZE", 16384)
```
See `common.go` for full list: `NP_SEMAPHORE_LIMIT`, `NP_HANDSHAKE_TIMEOUT`, `NP_POOL_GET_TIMEOUT`, etc.

### TLS Certificate Hot-Reload

Mode 2 (custom certs) reloads certificates hourly without restart using `GetCertificate` callback in `core.go`.

### Comments Style

Maintain bilingual (Chinese/English) comments for public APIs and exported functions:
```go
// NewServer 创建新的服务端实例
// NewServer creates a new server instance
func NewServer(parsedURL *url.URL, ...) (*Server, error) { ... }
```

## External Dependencies

All from `github.com/NodePassProject/*` ecosystem:
- **cert**: TLS certificate generation and management
- **conn**: Enhanced network connections with statistics tracking (`StatConn`)
- **logs**: Multi-level logger (None/Debug/Info/Warn/Error/Event)
- **name**: DNS resolver with caching and background refresh
- **pool**: TCP connection pooling with auto-scaling
- **quic**: QUIC multiplexing for 0-RTT connections

## Master Mode Specifics

### API Patterns

- Authentication via `X-API-Key` header (auto-generated, stored in `nodepass.gob`)
- SSE events at `/events` endpoint for real-time updates
- State persistence with `encoding/gob` for instance recovery
- OpenAPI spec at `/openapi.json`, Swagger UI at `/docs`

RESTful endpoints at `/{prefix}/*` (default `/api/*`):
- Instance CRUD: POST/GET/PATCH/PUT/DELETE on `/instances` and `/instances/{id}`
- Real-time events: SSE stream at `/events` (types: initial, create, update, delete, shutdown, log)
- Service info: GET/POST on `/info` for master details and alias updates
- TCPing utility: GET on `/tcping` for connection testing

### Instance Management

Each instance runs as a separate `exec.Cmd` process. Master tracks via `instances sync.Map` with status fields: `running`, `stopped`, `error`. Auto-restart enabled via `Restart` boolean field.

### State Persistence

All instances stored in `nodepass.gob` using Go's `encoding/gob`:
- Auto-saved on instance changes via `saveMasterState()`
- Restored on startup via `restoreMasterState()`
- Mutex-protected writes with `stateMu`

### API Authentication

API Key in `X-API-Key` header. Special instance ID `********` for key regeneration via PATCH action `restart`.

## Testing and Validation

No automated test suite currently. Manual testing workflow (from CONTRIBUTING.md):
1. Test each mode (server, client, master) with `log=debug`
2. Verify TCP and UDP forwarding separately
3. Test TLS modes 0, 1, 2 with certificate validation
4. Stress test with high concurrency and connection pool scaling

## Common Pitfalls

- **URL parsing**: Always include scheme (`server://`, `client://`, `master://`) or startup fails
- **TLS mismatch**: Client inherits TLS mode from server during handshake—don't configure client TLS manually
- **Pool capacity**: Server sets `max`, client sets `min`—mismatch causes connection issues
- **Local address detection**: Single-end mode triggers automatically for localhost/127.0.0.1 tunnel addresses
- **QUIC requirement**: QUIC mode (`quic=1`) forces TLS mode 1 minimum—cannot use mode 0
- **Don't modify NodePassProject libraries**: These are external dependencies, not internal packages
- **Always decode before using tunnel URLs**: Use `Common.decode()` for base64+XOR encoded data
- **UDP session cleanup**: Sessions in `targetUDPSession` require explicit cleanup with timeouts
- **Certificate hot-reload**: Only applies to `tls=2` mode with periodic checks every `ReloadInterval`
- **Graceful shutdown**: Use context cancellation propagation, don't abruptly close connections

## Key Files Reference

- `cmd/nodepass/main.go`: Entry point, version variable injection
- `cmd/nodepass/core.go`: Mode dispatch, TLS setup, CLI help formatting
- `internal/common.go`: Shared primitives (buffer pools, slot management, encoding, config init)
- `internal/server.go`: Server lifecycle, tunnel handshake, forward/reverse modes
- `internal/client.go`: Client lifecycle, single-end/dual-end modes, tunnel connection
- `internal/master.go`: HTTP API, SSE events, instance subprocess management, state persistence

## Documentation Requirements

When adding features:
1. Update relevant `docs/en/*.md` and `docs/zh/*.md` files
2. Add examples to `docs/en/examples.md`
3. Document new query parameters in `docs/en/configuration.md`
4. Update API endpoints in `docs/en/api.md` if touching master mode
5. Keep README.md feature list current

## Additional Notes

- Project uses Go 1.25+ features, maintain compatibility
- Single binary with no external runtime dependencies (except TLS cert files for mode 2)
- Focus on zero-configuration deployment - defaults should work for most use cases
- Performance-critical paths: buffer allocation, connection pooling, data transfer loops
- Security considerations: TLS mode selection, API key protection, input validation on master API

## Documentation References

- `/docs/en/how-it-works.md`: Deep dive into control/data channel separation and data flow modes
- `/docs/en/configuration.md`: Complete parameter reference with examples
- `/docs/en/api.md`: Master mode API specification with authentication and SSE events
- `CONTRIBUTING.md`: Development setup, architecture overview, contribution guidelines
