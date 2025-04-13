# Configuration Options

NodePass uses a minimalist approach to configuration, with all settings specified via command-line parameters and environment variables. This guide explains all available configuration options and provides recommendations for various deployment scenarios.

## Log Levels

NodePass provides five log verbosity levels that control the amount of information displayed:

- `debug`: Verbose debugging information - shows all operations and connections
- `info`: General operational information (default) - shows startup, shutdown, and key events
- `warn`: Warning conditions - only shows potential issues that don't affect core functionality
- `error`: Error conditions - shows only problems that affect functionality
- `fatal`: Critical conditions - shows only severe errors that cause termination

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
nodepass server://0.0.0.0:10101/0.0.0.0:8080?tls=2&crt=/path/to/cert.pem&key=/path/to/key.pem
```

## Environment Variables

NodePass behavior can be fine-tuned using environment variables. Below is the complete list of available variables with their descriptions, default values, and recommended settings for different scenarios.

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SEMAPHORE_LIMIT` | Maximum number of concurrent connections | 1024 | `export SEMAPHORE_LIMIT=2048` |
| `MIN_POOL_CAPACITY` | Minimum connection pool size | 16 | `export MIN_POOL_CAPACITY=32` |
| `MAX_POOL_CAPACITY` | Maximum connection pool size | 1024 | `export MAX_POOL_CAPACITY=4096` |
| `MIN_POOL_INTERVAL` | Minimum interval between connection creations | 1s | `export MIN_POOL_INTERVAL=500ms` |
| `MAX_POOL_INTERVAL` | Maximum interval between connection creations | 5s | `export MAX_POOL_INTERVAL=3s` |
| `UDP_DATA_BUF_SIZE` | Buffer size for UDP packets | 8192 | `export UDP_DATA_BUF_SIZE=16384` |
| `UDP_READ_TIMEOUT` | Timeout for UDP read operations | 5s | `export UDP_READ_TIMEOUT=10s` |
| `REPORT_INTERVAL` | Interval for health check reports | 5s | `export REPORT_INTERVAL=10s` |
| `RELOAD_INTERVAL` | Interval for certificate reload | 1h | `export RELOAD_INTERVAL=30m` |
| `SERVICE_COOLDOWN` | Cooldown period before restart attempts | 5s | `export SERVICE_COOLDOWN=3s` |
| `SHUTDOWN_TIMEOUT` | Timeout for graceful shutdown | 5s | `export SHUTDOWN_TIMEOUT=10s` |

### Connection Pool Tuning

The connection pool parameters are among the most important settings for performance tuning:

#### Pool Capacity Settings

- `MIN_POOL_CAPACITY`: Ensures a minimum number of available connections
  - Too low: Increased latency during traffic spikes as new connections must be established
  - Too high: Wasted resources maintaining idle connections
  - Recommended starting point: 25-50% of your average concurrent connections

- `MAX_POOL_CAPACITY`: Prevents excessive resource consumption while handling peak loads
  - Too low: Connection failures during traffic spikes
  - Too high: Potential resource exhaustion affecting system stability
  - Recommended starting point: 150-200% of your peak concurrent connections

#### Pool Interval Settings

- `MIN_POOL_INTERVAL`: Controls the minimum time between connection creation attempts
  - Too low: May overwhelm network with connection attempts
  - Recommended range: 500ms-2s depending on network latency

- `MAX_POOL_INTERVAL`: Controls the maximum time between connection creation attempts
  - Too high: May result in pool depletion during traffic spikes
  - Recommended range: 3s-10s depending on expected traffic patterns

#### Connection Management

- `SEMAPHORE_LIMIT`: Controls the maximum number of concurrent tunnel operations
  - Too low: Rejected connections during traffic spikes
  - Too high: Potential memory pressure from too many concurrent goroutines
  - Recommended range: 1000-5000 for most applications, higher for high-throughput scenarios

### UDP Settings

For applications relying heavily on UDP traffic:

- `UDP_DATA_BUF_SIZE`: Buffer size for UDP packets
  - Increase for applications sending large UDP packets
  - Default (8192) works well for most cases
  - Consider increasing to 16384 or higher for media streaming or game servers

- `UDP_READ_TIMEOUT`: Timeout for UDP read operations
  - Increase for high-latency networks or applications with slow response times
  - Decrease for low-latency applications requiring quick failover

### Service Management Settings

- `REPORT_INTERVAL`: Controls how frequently health status is reported
  - Lower values provide more frequent updates but increase log volume
  - Higher values reduce log output but provide less immediate visibility

- `RELOAD_INTERVAL`: Controls how frequently TLS certificates are checked for changes
  - Lower values detect certificate changes faster but increase file system operations
  - Higher values reduce overhead but delay detection of certificate updates

- `SERVICE_COOLDOWN`: Time to wait before attempting service restarts
  - Lower values attempt recovery faster but might cause thrashing in case of persistent issues
  - Higher values provide more stability but slower recovery from transient issues

- `SHUTDOWN_TIMEOUT`: Maximum time to wait for connections to close during shutdown
  - Lower values ensure quicker shutdown but may interrupt active connections
  - Higher values allow more time for connections to complete but delay shutdown

## Configuration Profiles

Here are some recommended environment variable configurations for common scenarios:

### High-Throughput Configuration

For applications requiring maximum throughput (e.g., media streaming, file transfers):

```bash
export MIN_POOL_CAPACITY=64
export MAX_POOL_CAPACITY=4096
export MIN_POOL_INTERVAL=500ms
export MAX_POOL_INTERVAL=3s
export SEMAPHORE_LIMIT=8192
export UDP_DATA_BUF_SIZE=32768
export REPORT_INTERVAL=10s
```

### Low-Latency Configuration

For applications requiring minimal latency (e.g., gaming, financial trading):

```bash
export MIN_POOL_CAPACITY=128
export MAX_POOL_CAPACITY=2048
export MIN_POOL_INTERVAL=100ms
export MAX_POOL_INTERVAL=1s
export SEMAPHORE_LIMIT=4096
export UDP_READ_TIMEOUT=2s
export REPORT_INTERVAL=1s
```

### Resource-Constrained Configuration

For deployment on systems with limited resources (e.g., IoT devices, small VPS):

```bash
export MIN_POOL_CAPACITY=8
export MAX_POOL_CAPACITY=256
export MIN_POOL_INTERVAL=2s
export MAX_POOL_INTERVAL=10s
export SEMAPHORE_LIMIT=512
export REPORT_INTERVAL=30s
export SHUTDOWN_TIMEOUT=3s
```

## Next Steps

- See [usage instructions](/docs/en/usage.md) for basic operational commands
- Explore [examples](/docs/en/examples.md) to understand deployment patterns
- Learn about [how NodePass works](/docs/en/how-it-works.md) to optimize your configuration
- Check the [troubleshooting guide](/docs/en/troubleshooting.md) if you encounter issues