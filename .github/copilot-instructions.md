# NodePass AI Coding Agent Instructions

## Project Overview

NodePass is an enterprise-grade TCP/UDP network tunneling solution with a three-tier architecture supporting server, client, and master modes. The core is written in Go with a focus on performance, security, and minimal configuration.

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
  
- **Bidirectional Data Flow**: Automatic mode detection in `Common.runMode`
  - Mode 0: Auto-detect based on target address bindability
  - Mode 1: Reverse/single-end (server receives OR client listens locally)
  - Mode 2: Forward/dual-end (server sends OR client connects remotely)

### External Dependencies (NodePassProject Ecosystem)

All critical networking primitives are in separate libraries:
- `github.com/NodePassProject/cert`: TLS certificate generation and management
- `github.com/NodePassProject/conn`: Custom connection types (`StatConn`, `TimeoutReader`, `DataExchange`)
- `github.com/NodePassProject/logs`: Structured logging with levels (None/Debug/Info/Warn/Error/Event)
- `github.com/NodePassProject/pool`: Connection pool management for both server and client

**Never modify these libraries directly** - they're external dependencies. Use their exported APIs only.

## Configuration System

### URL-Based Configuration

All modes use URL-style configuration: `scheme://[password@]host:port/target?param=value`

**Server**: `server://bind_addr:port/target_addr:port?max=1024&tls=1&log=debug`
**Client**: `client://server_addr:port/local_addr:port?min=128&mode=0&rate=100`
**Master**: `master://api_addr:port/prefix?log=info&tls=2&crt=path&key=path`

### Query Parameters

- `log`: none|debug|info|warn|error|event (default: info)
- `tls`: 0=plain, 1=self-signed, 2=custom cert (server/master only)
- `min`/`max`: Connection pool capacity (client sets min, server sets max)
- `mode`: 0=auto, 1=reverse/single-end, 2=forward/dual-end
- `read`: Timeout duration (e.g., 1h, 30m, 15s)
- `rate`: Mbps bandwidth limit (0=unlimited)
- `slot`: Max concurrent connections (default: 65536)
- `proxy`: PROXY protocol v1 support (0=off, 1=on)

### Environment Variables for Tuning

See `internal/common.go` for all `NP_*` environment variables:
- `NP_TCP_DATA_BUF_SIZE`: TCP buffer size (default: 16384)
- `NP_UDP_DATA_BUF_SIZE`: UDP buffer size (default: 2048)
- `NP_HANDSHAKE_TIMEOUT`: Handshake timeout (default: 5s)
- `NP_POOL_GET_TIMEOUT`: Pool connection timeout (default: 5s)
- `NP_REPORT_INTERVAL`: Health check interval (default: 5s)
- `NP_RELOAD_INTERVAL`: TLS cert reload interval (default: 1h)

## Development Workflow

### Building

```bash
# Development build
go build -o nodepass ./cmd/nodepass

# Release build (mimics .goreleaser.yml)
go build -trimpath -ldflags="-s -w -X main.version=dev" -o nodepass ./cmd/nodepass
```

### Testing Manually

No automated test suite exists currently. Test via real-world scenarios:

```bash
# Terminal 1: Server with debug logging
./nodepass "server://0.0.0.0:10101/127.0.0.1:8080?log=debug&tls=1&max=256"

# Terminal 2: Client
./nodepass "client://localhost:10101/127.0.0.1:9090?log=debug&min=64"

# Terminal 3: Master mode for API testing
./nodepass "master://0.0.0.0:9090/api?log=debug&tls=0"
```

Test all TLS modes (0, 1, 2) and protocol types (TCP, UDP). Verify graceful shutdown with SIGTERM/SIGINT.

### Release Process

Uses GoReleaser on tag push (`v*.*.*`). See `.goreleaser.yml` for build matrix (Linux, Windows, macOS, FreeBSD across multiple architectures).

## Code Patterns & Conventions

### Error Handling

Always wrap errors with context using `fmt.Errorf("function: operation failed: %w", err)`

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

### Buffer Pooling

Always use `Common.getTCPBuffer()` / `Common.putTCPBuffer()` or UDP equivalents to minimize allocations:
```go
buf := c.getTCPBuffer()
defer c.putTCPBuffer(buf)
// ... use buf
```

### Connection Slot Management

Before creating connections:
```go
if !c.tryAcquireSlot(isUDP) {
    return fmt.Errorf("slot limit reached")
}
defer c.releaseSlot(isUDP)
```

### Comments Style

Maintain bilingual (Chinese/English) comments for public APIs and exported functions:
```go
// NewServer 创建新的服务端实例
// NewServer creates a new server instance
func NewServer(parsedURL *url.URL, ...) (*Server, error) { ... }
```

## Master Mode Specifics

### API Structure

RESTful endpoints at `/{prefix}/*` (default `/api/*`):
- Instance CRUD: POST/GET/PATCH/PUT/DELETE on `/instances` and `/instances/{id}`
- Real-time events: SSE stream at `/events` (types: initial, create, update, delete, shutdown, log)
- OpenAPI docs: `/openapi.json` and `/docs` (Swagger UI)

### State Persistence

All instances stored in `nodepass.gob` using Go's `encoding/gob`:
- Auto-saved on instance changes via `saveMasterState()`
- Restored on startup via `restoreMasterState()`
- Mutex-protected writes with `stateMu`

### API Authentication

API Key in `X-API-Key` header. Special instance ID `********` for key regeneration via PATCH action `restart`.

## Common Pitfalls

1. **Don't modify NodePassProject libraries**: These are external dependencies, not internal packages
2. **Always decode before using tunnel URLs**: Use `Common.decode()` for base64+XOR encoded data
3. **TLS mode is server-controlled**: Clients receive TLS mode during handshake, don't override
4. **Pool capacity coordination**: Server sets `max`, client sets `min` - they must align correctly
5. **UDP session cleanup**: Sessions in `targetUDPSession` require explicit cleanup with timeouts
6. **Certificate hot-reload**: Only applies to `tls=2` mode with periodic checks every `ReloadInterval`
7. **Graceful shutdown**: Use context cancellation propagation, don't abruptly close connections

## Key Files Reference

- `cmd/nodepass/main.go`: Entry point, version variable injection
- `cmd/nodepass/core.go`: Mode dispatch, TLS setup, CLI help formatting
- `internal/common.go`: Shared primitives (buffer pools, slot management, encoding, config init)
- `internal/server.go`: Server lifecycle, tunnel handshake, forward/reverse modes
- `internal/client.go`: Client lifecycle, single-end/dual-end modes, tunnel connection
- `internal/master.go`: HTTP API, SSE events, instance subprocess management, state persistence
- `docs/en/how-it-works.md`: Detailed architecture documentation
- `docs/en/configuration.md`: Complete parameter reference
- `docs/en/api.md`: Master mode API specification

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
