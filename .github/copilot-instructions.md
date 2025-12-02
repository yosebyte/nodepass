# NodePass Development Guide

## Architecture Overview

NodePass is a Go-based TCP/UDP tunneling solution with a **tri-modal architecture** (Server/Client/Master) built on separation of control and data channels.

### Core Components

- **`cmd/nodepass/`**: Entry point with URL-based configuration parsing
  - `main.go`: Simple entry that invokes `start()` with version injection
  - `core.go`: URL parser, logger initialization, TLS mode selection, core factory (`createCore()`)
- **`internal/`**: Three operational modes sharing `common.go` base (~1970 lines):
  - `server.go`: Accepts tunnel connections via `tunnelHandshake()`, binds target addresses, supports bidirectional data flow
  - `client.go`: Initiates tunnel connections, supports single-end forwarding (`singleStart()`) and dual-end handshake (`commonStart()`)
  - `master.go`: RESTful API server with instance management, SSE events, gob persistence (~2165 lines)
  - `common.go`: Shared functionality - DNS caching, buffer pools, slot management, connection routing
- **External packages** (NodePassProject org on GitHub): 
  - `pool`: TCP connection pooling with auto-scaling (min/max capacity)
  - `quic`: QUIC transport with 0-RTT support
  - `npws`: WebSocket transport adapter
  - `conn`: Utilities (`DataExchange`, `StatConn` for traffic accounting, `RateLimiter` for bandwidth control)
  - `logs`: Structured logger with levels (none/debug/info/warn/error/event)
  - `cert`: TLS certificate generation and management

### Data Flow Modes

1. **Server Receives Mode** (Reverse): Server binds target address locally → signals client → client connects back → data flows: External → Server → Client → Target
2. **Server Sends Mode** (Forward): Server connects to remote target → client signals server → server creates outgoing connection → data flows: Client → Server → Remote Target
3. **Client Single-End Forwarding**: Client binds tunnel address locally (e.g., `127.0.0.1:8080`) → direct forwarding to target without server coordination (no control channel)

Mode selection is **automatic** via `initTargetListener()` success/failure. Server tries binding target address; if successful = mode 1 (reverse), if fails = mode 2 (forward). Client tries binding tunnel address; if successful = single-end, if fails = dual-end. Force with `mode` query parameter (`0`=auto, `1`=reverse/single, `2`=forward/dual).

## URL-Based Configuration

All configuration through URL scheme: `<mode>://<auth>@<tunnel>/<target>?<params>`

**URL Structure Examples:**
```
server://password@0.0.0.0:10101/127.0.0.1:8080?tls=1&max=512
client://password@server.com:10101/localhost:8080?min=64&type=1
master://0.0.0.0:9090/api?log=debug&tls=2&crt=/path/cert.pem&key=/path/key.pem
```

**Critical query parameters:**
- `log`: Log level - `none`|`debug`|`info`(default)|`warn`|`error`|`event`
- `tls`: Encryption mode - `0` (plain TCP/UDP), `1` (self-signed cert in memory), `2` (custom cert with `crt`/`key` files)
  - Mode 0: No encryption, fastest but insecure
  - Mode 1: Auto-generated self-signed cert, no verification, protects against passive sniffing
  - Mode 2: Custom certificate with validation, requires both `crt` and `key` parameters pointing to PEM files
  - **Note**: QUIC transport (`type=1`) requires minimum `tls=1`
- `type`: Pool transport protocol - `0` (TCP pool, default), `1` (QUIC with 0-RTT), `2` (WebSocket)
- `mode`: Force run mode - `0` (auto-detect via binding), `1` (server=reverse/client=single-end), `2` (server=forward/client=dual-end)
- `dns`: DNS cache TTL duration (default `5m`, accepts Go duration syntax like `30s`, `10m`, `1h`)
- `min`: Client minimum pool capacity (default `64`)
- `max`: Server maximum pool capacity (default `1024`)
- `rate`: Bandwidth limit in **Mbps * 8** (e.g., `rate=100` = 100Mbps = 12.5MB/s; internal unit is bytes/sec, computed as rate*125000)
- `slot`: Max concurrent connections - TCP+UDP combined (default `65536`, `0`=unlimited)
- `proxy`: PROXY protocol version - `0` (disabled), `1` (v1 text format), `2` (v2 binary format)
- `read`: Connection read timeout (default `0` = infinite, accepts Go duration like `30s`, `5m`)
- `dial`: Local bind IP for outgoing connections (default `auto` = system routing, or specific IP like `192.168.1.100`)
  - Automatic fallback to system routing if specified IP fails (logged as "fallback to system auto")
- `notcp`: Disable TCP forwarding - `0` (enabled), `1` (disabled)
- `noudp`: Disable UDP forwarding - `0` (enabled), `1` (disabled)

**Password field usage:** The `@` password portion in URLs (e.g., `mykey@server:10101`) becomes `tunnelKey` for authentication - it's NOT a system password, just a shared secret for tunnel validation. Server compares incoming `tunnelKey` via XOR+base64 encoding in handshake.

Examples in `docs/en/examples.md`, full configuration reference in `docs/en/configuration.md`.

## Development Workflow

### Building

```bash
# Development build
cd cmd/nodepass
go build -ldflags "-X main.version=dev"

# Release build (via goreleaser)
goreleaser build --snapshot --clean

# Docker build (multi-stage, scratch-based final image)
docker build --build-arg VERSION=dev -t nodepass:dev .
```

Build produces single static binary with no external dependencies. The `-ldflags "-X main.version=..."` injects version into `main.version` variable displayed in `exit()` banner.

### Testing Patterns

**No test suite exists** - all testing is manual via URL invocations. Common test scenarios:

```bash
# Server mode (binds :10101 for tunnel, forwards to local 8080)
nodepass "server://:10101/127.0.0.1:8080?log=debug&tls=1"

# Client mode (connects to server:10101, creates local listener on :8080)
nodepass "client://server:10101/127.0.0.1:8080?min=128&log=debug"

# Master API mode (launches API server on :10101 with /api prefix)
nodepass "master://:10101/api?log=debug&tls=1"

# Test QUIC transport with bandwidth limiting
nodepass "server://:10101/127.0.0.1:8080?type=1&tls=1&rate=100"

# Test multi-target load balancing (comma-separated targets)
nodepass "client://server:10101/target1.com:80,target2.com:80,target3.com:80?mode=2"
```

**Debugging tips:**
- Use `log=debug` to see connection lifecycle events, pool operations, handshake details
- Check `DataExchange` log messages for connection completion status and byte counts
- Monitor pool capacity with `Active()` and `Capacity()` calls logged periodically
- TLS handshake failures appear as "access denied" warnings - verify `tunnelKey` matches
- DNS resolution issues trigger fallback to cached addresses with warning logs

### Environment Tuning

Performance constants in `common.go` (lines 93-105) are environment-configurable via `NP_*` prefix:

```bash
# Increase semaphore limit for high concurrency (default 65536)
export NP_SEMAPHORE_LIMIT=131072

# Larger TCP buffer for high-bandwidth links (default 16384)
export NP_TCP_DATA_BUF_SIZE=32768

# Extend handshake timeout for slow networks (default 5s)
export NP_HANDSHAKE_TIMEOUT=10s

# Pool connection acquisition timeout (default 5s)
export NP_POOL_GET_TIMEOUT=10s

# Pool scaling intervals (defaults: min=100ms, max=1s)
export NP_MIN_POOL_INTERVAL=50ms
export NP_MAX_POOL_INTERVAL=2s

# Health check report frequency (default 5s)
export NP_REPORT_INTERVAL=10s

# Service restart cooldown (default 3s)
export NP_SERVICE_COOLDOWN=5s

# Graceful shutdown timeout (default 5s)
export NP_SHUTDOWN_TIMEOUT=10s

# TLS certificate reload interval for mode 2 (default 1h)
export NP_RELOAD_INTERVAL=30m
```

All duration values accept Go duration syntax (`s`, `m`, `h`). Changes require restart to take effect.

## Code Conventions

### Logging

Use structured logging with `logger` from `logs.Logger`. Six levels: none/debug/info/warn/error/event. Format strings with `%v` placeholders:

```go
logger.Debug("TLS cert reloaded: %v", crtFile)
logger.Info("Server started: server://%v@%v/%v", key, tunnel, target)
logger.Warn("tunnelHandshake: access denied: %v", remoteAddr)
logger.Error("Certificate load failed: %v", err)
logger.Event("Traffic stats: TCP RX=%d TX=%d", tcpRX, tcpTX)
```

**Never use `fmt.Printf`** except in `exit()` help banner. All user-facing output goes through logger.

### Error Handling

Wrap errors with context using `fmt.Errorf` with `%w` verb for error chain preservation:

```go
return fmt.Errorf("start: initTunnelListener failed: %w", err)
return fmt.Errorf("tunnelHandshake: decode failed: %w", err)
```

Functions return `error` as last return value. Restart logic uses `err != nil && err != io.EOF` pattern - `io.EOF` signals graceful shutdown, other errors trigger restart after `serviceCooldown`.

### Connection Pool Interface

All transport types (`pool.ServerPool`, `quic.ServerPool`, `npws.ServerPool`) implement unified `TransportPool` interface (defined in `common.go` line 92):

```go
type TransportPool interface {
    // IncomingGet retrieves connection from server pool by ID with timeout
    IncomingGet(timeout time.Duration) (string, net.Conn, error)
    
    // OutgoingGet retrieves connection from client pool for given ID with timeout
    OutgoingGet(id string, timeout time.Duration) (net.Conn, error)
    
    // Flush signals pool to drop all connections and reset state
    Flush()
    
    // Close terminates pool and all managed connections
    Close()
    
    // Ready reports if pool has reached minimum capacity
    Ready() bool
    
    // Active returns current active connection count
    Active() int
    
    // Capacity returns maximum pool capacity
    Capacity() int
    
    // Interval returns current auto-scaling interval
    Interval() time.Duration
    
    // AddError increments error counter for health monitoring
    AddError()
    
    // ErrorCount returns cumulative error count
    ErrorCount() int
    
    // ResetError clears error counter
    ResetError()
}
```

Connection IDs are generated via FNV hash: `hash := fnv.New64a(); hash.Write([]byte); id := hex.EncodeToString(hash.Sum(nil))`. Server generates IDs for incoming connections, client receives IDs via control channel.

### Buffer Pool Management

**Critical**: Always return buffers to prevent memory leaks. Pools are initialized in constructor with `sync.Pool`:

```go
tcpBufferPool: &sync.Pool{
    New: func() any {
        buf := make([]byte, tcpDataBufSize)
        return &buf
    },
}
```

Usage pattern:
```go
buffer := c.getTCPBuffer()        // Acquire from pool
defer c.putTCPBuffer(buffer)      // ALWAYS return via defer
// Use buffer for I/O operations...
```

UDP buffers follow identical pattern with `getUDPBuffer()`/`putUDPBuffer()`. Buffer sizes configurable via `NP_TCP_DATA_BUF_SIZE` (default 16384) and `NP_UDP_DATA_BUF_SIZE` (default 16384).

### Slot Management

Connection slots limit concurrent connections via atomic counters. Check before accepting connections:

```go
if !c.tryAcquireSlot(isUDP) {
    logger.Warn("Slot limit reached: %d", c.slotLimit)
    conn.Close()
    return
}
defer c.releaseSlot(isUDP)
```

Slots are combined TCP+UDP count. `slotLimit=0` disables limit. Slot tracking uses `atomic.AddInt32()` for thread-safe counters.

### Context Management

Each mode initializes context in `start()` method:

```go
func (c *Common) initContext() {
    c.ctx, c.cancel = context.WithCancel(context.Background())
}
```

Graceful shutdown via `shutdown(ctx, stopFunc)` helper:
1. Calls `stopFunc()` to close listeners/pools
2. Waits for `ctx.Done()` or `shutdownTimeout` (default 5s)
3. Logs completion/timeout status

Restart loop pattern in `Run()` methods:
```go
for ctx.Err() == nil {
    if err := c.start(); err != nil && err != io.EOF {
        c.logger.Error("Client error: %v", err)
        c.stop()
        select {
        case <-ctx.Done():
            return
        case <-time.After(serviceCooldown):  // 3s default
        }
        logInfo("Client restart")
    }
}
```

Use `contextCheckInterval` (50ms) in tight loops: `select { case <-ctx.Done(): return; case <-time.After(contextCheckInterval): }`

### Traffic Accounting

All connections wrapped in `conn.StatConn` for automatic byte counting and rate limiting:

```go
targetConn = &conn.StatConn{
    Conn: targetConn,
    RX:   &c.tcpRX,     // Points to Common's atomic uint64 counter
    TX:   &c.tcpTX,     // Points to Common's atomic uint64 counter
    Rate: c.rateLimiter, // Optional rate limiter (nil if rate=0)
}
```

Counters updated atomically on every Read/Write. Master mode reads counters to compute traffic deltas. `DataExchange()` from `conn` package handles bidirectional copy with automatic accounting:

```go
conn.DataExchange(connA, connB, readTimeout, buffer1, buffer2)
```

Rate limiting initialized via `initRateLimiter()` if `rateLimit > 0` (rate in bytes/sec = query param * 125000).

## Master Mode Specifics

### Instance Management

Instances stored in `sync.Map` (concurrent-safe), persisted to `gob/nodepass.gob` using `gob` encoding. State file layout:
- API key (auto-generated 32-byte hex on first start)
- Instance map serialization with all fields except those tagged `gob:"-"`

Key `Instance` struct fields:
```go
type Instance struct {
    ID       string        // 8-char hex identifier
    Alias    string        // User-friendly name
    Type     string        // "server" or "client"
    Status   string        // "running", "stopped", "error"
    URL      string        // Original user-provided URL
    Config   string        // Computed URL with all defaults filled
    Restart  bool          // Auto-restart policy
    Meta     Meta          // Metadata with peer info and tags
    cmd      *exec.Cmd     // Running subprocess (not serialized)
    stopped  chan struct{} // Shutdown coordination (not serialized)
    // Traffic baseline tracking (not serialized)
    TCPRXBase/TCPTXBase/UDPRXBase/UDPTXBase uint64
}
```

Instance lifecycle:
1. **Create**: `POST /instances` with URL → generates ID → spawns subprocess → stores in `sync.Map` → persists to gob
2. **Monitor**: Periodic goroutine reads `/proc/<pid>/status` for traffic stats, computes deltas from baseline
3. **Update**: `PATCH /instances/{id}` with actions: `start`, `stop`, `restart`, `reset-traffic`, `toggle-restart`
4. **Delete**: `DELETE /instances/{id}` → stops subprocess → removes from map → re-persists gob

Subprocess management uses `exec.CommandContext()` with instance-specific context. Logs captured via custom `InstanceLogWriter` that parses structured logs and emits SSE events.

### SSE Events

Real-time updates via `/events` endpoint (Server-Sent Events). Event types and payloads:

- `initial`: Full instance list on connection (sent once per subscriber)
- `create`: New instance created (includes full Instance object)
- `update`: Instance state changed (includes full Instance object with updated fields)
- `delete`: Instance removed (includes ID only)
- `shutdown`: Master shutting down (no payload)
- `log`: Instance log line (includes `instance.id` and `logs` fields)

Subscribers stored in `sync.Map` with unique IDs. Event broadcasting via `notifyChannel` (buffered channel). Connection management pattern:

```go
subscriber := &Subscriber{id: generateID(), channel: make(chan *InstanceEvent, 100)}
m.subscribers.Store(subscriber.id, subscriber)
defer m.subscribers.Delete(subscriber.id)

for {
    select {
    case event := <-subscriber.channel:
        fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, jsonData)
        flusher.Flush()
    case <-r.Context().Done():
        return
    }
}
```

### API Authentication

Auto-generated API key on first start. Special instance ID `********` (8 asterisks) reserved for key operations:
- `GET /instances/********`: Retrieve current API key
- `PATCH /instances/********` with `{"action": "restart"}`: Regenerate API key

Protected endpoints check `X-API-Key` header. Public endpoints: `/openapi.json`, `/docs` (Swagger UI).

Key validation pattern:
```go
if apiKey := r.Header.Get("X-API-Key"); apiKey != m.apiKey {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

### TCPing Functionality

Built-in connectivity testing via `GET /tcping?target=host:port`. Concurrent limit enforced via buffered semaphore (`tcpingSem chan struct{}` with capacity 10). Returns JSON:

```json
{
  "target": "example.com:443",
  "connected": true,
  "latency": 42,
  "error": null
}
```

Timeout handling: 1s for semaphore acquisition, 5s for TCP dial. Latency measured in milliseconds.

## Integration Points

### External Package Boundaries

- **`github.com/NodePassProject/pool`**: TCP connection pooling with dynamic scaling
- **`github.com/NodePassProject/quic`**: QUIC-based transport (0-RTT support)
- **`github.com/NodePassProject/npws`**: WebSocket transport wrapper
- **`github.com/NodePassProject/conn`**: Connection helpers (`DataExchange`, `StatConn`, `RateLimiter`)
- **`github.com/NodePassProject/cert`**: TLS certificate generation/management

When modifying transport behavior, coordinate with corresponding package version in `go.mod`.

### DNS Caching

Custom DNS resolution via `dnsCacheEntry` stored in `sync.Map` with TTL. Functions: `getTunnelTCPAddr()`, `getTargetTCPAddr()`.

### Handshake Protocol

**Server-side handshake** (`server.go` lines 194-320):
1. Accepts connections on `tunnelListener`
2. Reads XOR+base64 encoded `tunnelKey` from client (5s timeout)
3. Validates key match - closes connection on mismatch with "access denied" warning
4. Extracts client IP from `RemoteAddr()`
5. Constructs tunnel config URL: `np://<poolType>@<maxPoolCapacity>/<dataFlow>#<tlsCode>`
6. Sends encoded config to client
7. Assigns connection to `controlConn` for signal channel

**Client-side handshake** (`client.go` lines 218-273):
1. Dials server tunnel address with 5s timeout
2. Sends XOR+base64 encoded `tunnelKey`
3. Reads tunnel config URL from server
4. Decodes and parses URL to extract:
   - `poolType`: Transport protocol to use
   - `maxPoolCapacity`: Server-allocated pool size
   - `dataFlow`: Direction indicator (`+` or `-`)
   - `tlsCode`: TLS mode for data connections
5. Updates local config with server values
6. Wraps connection in `TimeoutReader` (3x reportInterval = 15s default)

Handshake encoding uses `xor()` followed by `base64.StdEncoding` - see `common.go` lines 250-270.

### Load Balancing & Failover

Multi-target support via comma-separated addresses in URL path. `dialWithRotation()` (`common.go` lines 385-450) implements:
- Round-robin distribution using atomic counter
- Automatic failover on connection errors
- Single-target fast path optimization
- Dynamic DNS resolution per attempt

Example: `client://server:10101/target1:80,target2:80,target3:80` rotates across three backends.

## Common Pitfalls

1. **TLS Mode vs Pool Type**: `tls` parameter applies to data channel, `type` parameter selects transport (QUIC requires `tls=1` minimum)
2. **URL Password Field**: Used as `tunnelKey` for authentication - not actual password
3. **Buffer Pool Management**: Always return buffers via `putTCPBuffer()`/`putUDPBuffer()` to prevent leaks
4. **Signal Channel Buffering**: `signalChan` has `semaphoreLimit` capacity - blocks if full
5. **Instance Config vs URL**: Master stores both user-provided URL and computed config string with all defaults

## Key File References

- **`internal/common.go`** (1970 lines): Core shared functionality
  - Lines 29-85: `Common` struct definition with all shared fields
  - Lines 93-122: Environment-configurable performance constants
  - Lines 140-165: Buffer pool management (`getTCPBuffer`, `putTCPBuffer`, `getUDPBuffer`, `putUDPBuffer`)
  - Lines 168-200: Slot management (`tryAcquireSlot`, `releaseSlot`)
  - Lines 250-270: Handshake encoding/decoding (`xor`, `encode`, `decode`)
  - Lines 385-450: Load balancing with failover (`dialWithRotation`)
  - Lines 722-726: Rate limiter initialization
  - Lines 1229, 1568: `DataExchange` calls for bidirectional traffic

- **`internal/server.go`** (320 lines): Server mode implementation
  - Lines 32-62: Server constructor with pool initialization
  - Lines 65-106: Run loop with restart logic
  - Lines 109-183: Start sequence and mode detection
  - Lines 194-320: Tunnel handshake with concurrent connection acceptance

- **`internal/client.go`** (273 lines): Client mode implementation
  - Lines 33-61: Client constructor
  - Lines 111-132: Mode detection logic (single-end vs dual-end)
  - Lines 135-210: Pool initialization per transport type
  - Lines 218-273: Tunnel handshake with config reception

- **`internal/master.go`** (2165 lines): Master API server
  - Lines 67-90: Master struct definition
  - Lines 91-124: Instance struct with traffic tracking
  - Lines 138-145: InstanceEvent for SSE
  - Lines 330+: RESTful handlers and instance management

- **`cmd/nodepass/core.go`** (165 lines): Entry point and configuration
  - Lines 17-35: URL parsing and core creation
  - Lines 38-59: Logger initialization
  - Lines 62-75: Core factory (`createCore`)
  - Lines 78-143: TLS configuration with three modes
