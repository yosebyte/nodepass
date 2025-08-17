# NodePass API Reference

## Overview

NodePass offers a RESTful API in Master Mode that enables programmatic control and integration with frontend applications. This section provides comprehensive d     // Configure auto-start policy for new instance based on type
     if (data.success) {

# NodePass API Reference

## Overview

NodePass provides a RESTful API in Master Mode for programmatic control and frontend integration. This document covers all endpoints, data structures, and best practices.

## Master Mode API

When running in `master://` mode, NodePass supports:

1. Creating and managing server/client instances
2. Real-time monitoring of status, traffic, and health checks
3. Instance control (start, stop, restart, reset traffic)
4. Configurable auto-start policy
5. Flexible parameter configuration

### Base URL

```
master://<api_addr>/<prefix>?<log>&<tls>
```

- `<api_addr>`: Listen address (e.g. `0.0.0.0:9090`)
- `<prefix>`: API path prefix (default `/api`)

### Starting Master Mode

```bash
nodepass "master://0.0.0.0:9090?log=info"
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"
```

### Main Endpoints

| Endpoint           | Method | Description           |
|--------------------|--------|----------------------|
| `/instances`       | GET    | List all instances   |
| `/instances`       | POST   | Create new instance  |
| `/instances/{id}`  | GET    | Get instance details |
| `/instances/{id}`  | PATCH  | Update/control       |
| `/instances/{id}`  | PUT    | Update instance URL  |
| `/instances/{id}`  | DELETE | Delete instance      |
| `/events`          | GET    | SSE event stream     |
| `/info`            | GET    | Master info          |
| `/tcping`          | GET    | TCP connectivity test |
| `/openapi.json`    | GET    | OpenAPI spec         |
| `/docs`            | GET    | Swagger UI           |

### API Authentication

API Key authentication is enabled by default. The key is auto-generated and stored in `nodepass.gob`.

- Protected: `/instances`, `/instances/{id}`, `/events`, `/info`, `/tcping`
- Public: `/openapi.json`, `/docs`
- Use header: `X-API-Key: <key>`
- Regenerate: PATCH `/instances/********` with `{ "action": "restart" }`

### Instance Data Structure

```json
{
  "id": "a1b2c3d4",
  "alias": "alias",
  "type": "client|server",
  "status": "running|stopped|error",
  "url": "...",
  "restart": true,
  "ping": 0,
  "pool": 0,
  "tcps": 0,
  "udps": 0,
  "tcprx": 0,
  "tcptx": 0,
  "udprx": 0,
  "udptx": 0
}
```

- `ping`/`pool`: health check data
- `tcps`/`udps`: current active connection count
- `tcprx`/`tcptx`/`udprx`/`udptx`: cumulative traffic stats
- `restart`: auto-start policy

### Instance URL Format

- Server: `server://<bind_addr>:<bind_port>/<target_host>:<target_port>?<params>`
- Client: `client://<server_host>:<server_port>/<local_host>:<local_port>?<params>`
- Supported params: `log`, `tls`, `crt`, `key`, `min`, `max`, `mode`, `read`, `rate`

### URL Query Parameters

- `log`: Log level (`none`, `debug`, `info`, `warn`, `error`, `event`)
- `tls`: TLS encryption mode (`0`, `1`, `2`) - server/master modes only
- `crt`/`key`: Certificate/key file paths (when `tls=2`)
- `min`/`max`: Connection pool capacity (`min` set by client, `max` set by server and delivered to client during handshake)
- `mode`: Run mode control (`0`, `1`, `2`) - controls operational behavior
  - For servers: `0`=auto, `1`=reverse mode, `2`=forward mode
  - For clients: `0`=auto, `1`=single-end forwarding, `2`=dual-end handshake
- `read`: Data read timeout duration (e.g. 1h, 30m, 15s)
- `rate`: Bandwidth rate limit in Mbps (0=unlimited)

### Real-time Events (SSE)

- Event types: `initial`, `create`, `update`, `delete`, `shutdown`, `log`
- Only normal logs are sent in `log` events; traffic/health logs are filtered
- Connect to `/events` for real-time instance and log updates

### Additional Notes

- All instance, traffic, health, alias, and auto-start data are persisted and restored after restart
- Full OpenAPI spec: `/openapi.json`, Swagger UI: `/docs`
### Real-time Event Monitoring with SSE

NodePass now supports Server-Sent Events (SSE) for real-time monitoring of instance state changes. This allows frontend applications to receive instant notifications about instance creation, updates, and deletions without polling.

#### Using the SSE Endpoint

The SSE endpoint is available at:
```
GET /events
```

This endpoint establishes a persistent connection that delivers events in real-time using the SSE protocol format.

#### Event Types

The following event types are supported:

1. `initial` - Sent when a connection is established, containing the current state of all instances
2. `create` - Sent when a new instance is created
3. `update` - Sent when an instance is updated (status change, start/stop operations)
4. `delete` - Sent when an instance is deleted
5. `shutdown` - Sent when the master service is about to shut down, notifying frontend applications to close their connections
6. `log` - Sent when an instance produces new log content, including the log text

#### Handling Instance Logs

The new `log` event type allows for real-time reception and display of instance log output. This is useful for monitoring and debugging:

```javascript
// Handle log events
function appendLogToInstanceUI(instanceId, logText) {
  // Find or create log container
  let logContainer = document.getElementById(`logs-${instanceId}`);
  if (!logContainer) {
    logContainer = document.createElement('div');
    logContainer.id = `logs-${instanceId}`;
    document.getElementById('instance-container').appendChild(logContainer);
  }
  
  // Create new log entry
  const logEntry = document.createElement('div');
  logEntry.className = 'log-entry';
  
  // Can parse ANSI color codes or format logs here
  logEntry.textContent = logText;
  
  // Add to container
  logContainer.appendChild(logEntry);
  
  // Scroll to latest log
  logContainer.scrollTop = logContainer.scrollHeight;
}
```

When implementing log handling, consider the following best practices:

1. **Buffer Management**: Limit the number of log entries to prevent memory issues
2. **ANSI Color Parsing**: Parse ANSI color codes in logs for better readability
3. **Filtering Options**: Provide options to filter logs by severity or content
4. **Search Functionality**: Allow users to search within instance logs
5. **Log Persistence**: Optionally save logs to local storage for review after page refresh

#### JavaScript Client Implementation

Here's an example of how to consume the SSE endpoint in a JavaScript frontend:

```javascript
function connectToEventSource() {
  const eventSource = new EventSource(`${API_URL}/events`, {
    // If authentication is needed, native EventSource doesn't support custom headers
    // Need to use fetch API to implement a custom SSE client
  });
  
  // If using API Key, use custom implementation instead of native EventSource
  // Example using native EventSource (for non-protected endpoints)
  eventSource.addEventListener('instance', (event) => {
    const data = JSON.parse(event.data);
    
    switch (data.type) {
      case 'initial':
        console.log('Initial instance state:', data.instance);
        updateInstanceUI(data.instance);
        break;
      case 'create':
        console.log('Instance created:', data.instance);
        addInstanceToUI(data.instance);
        break;
      case 'update':
        console.log('Instance updated:', data.instance);
        updateInstanceUI(data.instance);
        break;
      case 'delete':
        console.log('Instance deleted:', data.instance);
        removeInstanceFromUI(data.instance.id);
        break;
      case 'log':
        console.log(`Instance ${data.instance.id} log:`, data.logs);
        appendLogToInstanceUI(data.instance.id, data.logs);
        break;
      case 'shutdown':
        console.log('Master service is shutting down');
        // Close the event source and show notification
        eventSource.close();
        showShutdownNotification();
        break;
    }
  });
  
  eventSource.addEventListener('error', (error) => {
    console.error('SSE connection error:', error);
    // Attempt to reconnect after a delay
    setTimeout(() => {
      eventSource.close();
      connectToEventSource();
    }, 5000);
  });
  
  return eventSource;
}

// Example of creating SSE connection with API Key
function connectToEventSourceWithApiKey(apiKey) {
  // Native EventSource doesn't support custom headers, need to use fetch API
  fetch(`${API_URL}/events`, {
    method: 'GET',
    headers: {
      'X-API-Key': apiKey,
      'Cache-Control': 'no-cache'
    }
  }).then(response => {
    if (!response.ok) {
      throw new Error(`HTTP error: ${response.status}`);
    }
    
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    
    function processStream() {
      reader.read().then(({ value, done }) => {
        if (done) {
          console.log('Connection closed');
          // Try to reconnect
          setTimeout(() => connectToEventSourceWithApiKey(apiKey), 5000);
          return;
        }
        
        buffer += decoder.decode(value, { stream: true });
        
        const lines = buffer.split('\n\n');
        buffer = lines.pop() || '';
        
        for (const line of lines) {
          if (line.trim() === '') continue;
          
          const eventMatch = line.match(/^event: (.+)$/m);
          const dataMatch = line.match(/^data: (.+)$/m);
          
          if (eventMatch && dataMatch) {
            const data = JSON.parse(dataMatch[1]);
            // Process events - see switch code above
          }
        }
        
        processStream();
      }).catch(error => {
        console.error('Read error:', error);
        // Try to reconnect
        setTimeout(() => connectToEventSourceWithApiKey(apiKey), 5000);
      });
    }
    
    processStream();
  }).catch(error => {
    console.error('Connection error:', error);
    // Try to reconnect
    setTimeout(() => connectToEventSourceWithApiKey(apiKey), 5000);
  });
}
```

#### Benefits of SSE over Polling

Using SSE for instance monitoring offers several advantages over traditional polling:

1. **Reduced Latency**: Changes are delivered in real-time
2. **Reduced Server Load**: Eliminates unnecessary polling requests
3. **Bandwidth Efficiency**: Only sends data when changes occur
4. **Native Browser Support**: Built-in browser support without additional libraries
5. **Automatic Reconnection**: Browsers automatically reconnect if the connection is lost

#### Best Practices for SSE Implementation

When implementing SSE in your frontend:

1. **Handle Reconnection**: While browsers attempt to reconnect automatically, implement custom logic for persistent connections
2. **Process Events Efficiently**: Keep event processing fast to avoid UI blocking
3. **Implement Fallback**: For environments where SSE is not supported, implement a polling fallback
4. **Handle Errors**: Properly handle connection errors and disconnects

### Traffic Statistics

The Master API provides traffic statistics, but there are important requirements to note:

1. **Basic Traffic Metrics**: NodePass periodically provides cumulative TCP and UDP traffic values in both inbound and outbound directions. The frontend application needs to store and process these values to derive meaningful statistics.
   ```javascript
   function processTrafficStats(instanceId, currentStats) {
     // Store the current timestamp
     const timestamp = Date.now();
     
     // If we have previous stats for this instance, calculate the difference
     if (previousStats[instanceId]) {
       const timeDiff = timestamp - previousStats[instanceId].timestamp;
       const tcpInDiff = currentStats.tcp_in - previousStats[instanceId].tcp_in;
       const tcpOutDiff = currentStats.tcp_out - previousStats[instanceId].tcp_out;
       const udpInDiff = currentStats.udp_in - previousStats[instanceId].udp_in;
       const udpOutDiff = currentStats.udp_out - previousStats[instanceId].udp_out;
       
       // Store historical data for graphs
       storeTrafficHistory(instanceId, {
         timestamp,
         tcp_in_rate: tcpInDiff / timeDiff * 1000, // bytes per second
         tcp_out_rate: tcpOutDiff / timeDiff * 1000,
         udp_in_rate: udpInDiff / timeDiff * 1000,
         udp_out_rate: udpOutDiff / timeDiff * 1000
       });
     }
     
     // Update the previous stats for next calculation
     previousStats[instanceId] = {
       timestamp,
       tcp_in: currentStats.tcp_in,
       tcp_out: currentStats.tcp_out,
       udp_in: currentStats.udp_in,
       udp_out: currentStats.udp_out
     };
   }
   ```

2. **Data Persistence**: Since the API only provides cumulative values, the frontend must implement proper storage and calculation logic
   ```javascript
   // Example of frontend storage structure for traffic history
   const trafficHistory = {};
   
   function storeTrafficHistory(instanceId, metrics) {
     if (!trafficHistory[instanceId]) {
       trafficHistory[instanceId] = {
         timestamps: [],
         tcp_in_rates: [],
         tcp_out_rates: [],
         udp_in_rates: [],
         udp_out_rates: []
       };
     }
     
     trafficHistory[instanceId].timestamps.push(metrics.timestamp);
     trafficHistory[instanceId].tcp_in_rates.push(metrics.tcp_in_rate);
     trafficHistory[instanceId].tcp_out_rates.push(metrics.tcp_out_rate);
     trafficHistory[instanceId].udp_in_rates.push(metrics.udp_in_rate);
     trafficHistory[instanceId].udp_out_rates.push(metrics.udp_out_rate);
     
     // Keep history size manageable
     const MAX_HISTORY = 1000;
     if (trafficHistory[instanceId].timestamps.length > MAX_HISTORY) {
       trafficHistory[instanceId].timestamps.shift();
       trafficHistory[instanceId].tcp_in_rates.shift();
       trafficHistory[instanceId].tcp_out_rates.shift();
       trafficHistory[instanceId].udp_in_rates.shift();
       trafficHistory[instanceId].udp_out_rates.shift();
     }
   }
   ```

## Instance Data Structure

Instance objects in API responses contain the following fields:

```json
{
  "id": "a1b2c3d4",           // Unique instance identifier
  "alias": "web-server-01",   // Instance alias (optional, for friendly display names)
  "type": "server",           // Instance type: server or client
  "status": "running",        // Instance status: running, stopped, or error
  "url": "server://...",      // Instance configuration URL
  "restart": true,            // Auto-start policy
  "tcprx": 1024,             // TCP bytes received
  "tcptx": 2048,             // TCP bytes transmitted
  "udprx": 512,              // UDP bytes received
  "udptx": 256               // UDP bytes transmitted
}
```

**Notes:** 
- `alias` field is optional and will be an empty string if not set
- Traffic statistics fields (tcprx, tcptx, udprx, udptx) are only valid when debug mode is enabled
- `restart` field controls the instance's auto-start behavior

## System Information Endpoint

The `/info` endpoint provides system information about the NodePass Master service. This endpoint is useful for monitoring, troubleshooting, and verifying system status.

### Request

```
GET /info
```

API Key Authentication Required: Yes

### Response

The response contains the following system information fields:

```json
{
  "os": "linux",          // Operating system type
  "arch": "amd64",        // System architecture
  "ver": "1.2.0",         // NodePass version
  "name": "example.com",  // Tunnel hostname
  "uptime": 11525,         // API uptime in seconds
  "log": "info",          // Log level
  "tls": "1",             // TLS status
  "crt": "/path/to/cert", // Certificate path
  "key": "/path/to/key"   // Key path
}
```

### Usage Example

```javascript
// Get system information
async function getSystemInfo() {
  const response = await fetch(`${API_URL}/info`, {
    method: 'GET',
    headers: {
      'X-API-Key': apiKey
    }
  });
  
  return await response.json();
}

// Display service uptime
function displayServiceUptime() {
  getSystemInfo().then(info => {
    console.log(`Service uptime: ${info.uptime} seconds`);
    // You can also format it for better readability
    const hours = Math.floor(info.uptime / 3600);
    const minutes = Math.floor((info.uptime % 3600) / 60);
    const seconds = info.uptime % 60;
    console.log(`Service uptime: ${hours}h ${minutes}m ${seconds}s`);
  });
}
```

### Monitoring Best Practices

- **Regular Polling**: Periodically poll this endpoint to ensure service is running
- **Version Verification**: Check version number after deploying updates
- **Uptime Monitoring**: Monitor uptime to detect unexpected restarts
- **Log Level Verification**: Confirm that the current log level matches expectations

## API Endpoint Documentation

For detailed API documentation including request and response examples, please use the built-in Swagger UI documentation available at the `/docs` endpoint. This interactive documentation provides comprehensive information about:

- Available endpoints
- Required parameters
- Response formats
- Example requests and responses
- Schema definitions

### Accessing Swagger UI

To access the Swagger UI documentation:

```
http(s)://<api_addr>[<prefix>]/docs
```

For example:
```
http://localhost:9090/api/docs
```

The Swagger UI provides a convenient way to explore and test the API directly in your browser. You can execute API calls against your running NodePass Master instance and see the actual responses.

## Complete API Reference

### Instance Management Endpoints Details

#### GET /instances
- **Description**: Get list of all instances
- **Authentication**: API Key required
- **Response**: Array of instance objects
- **Example**:
```javascript
const instances = await fetch(`${API_URL}/instances`, {
  headers: { 'X-API-Key': apiKey }
});
```

#### POST /instances
- **Description**: Create new instance
- **Authentication**: API Key required
- **Request Body**: `{ "url": "client:// or server:// format URL" }`
- **Response**: Newly created instance object
- **Example**:
```javascript
const newInstance = await fetch(`${API_URL}/instances`, {
  method: 'POST',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ url: 'server://0.0.0.0:8080/localhost:3000' })
});
```

#### GET /instances/{id}
- **Description**: Get specific instance details
- **Authentication**: API Key required
- **Response**: Instance object
- **Example**:
```javascript
const instance = await fetch(`${API_URL}/instances/abc123`, {
  headers: { 'X-API-Key': apiKey }
});
```

#### PATCH /instances/{id}
- **Description**: Update instance state, alias, or perform control actions
- **Authentication**: API Key required
- **Request Body**: `{ "alias": "new alias", "action": "start|stop|restart|reset", "restart": true|false }`
- **Note**: Only specified fields are updated without interrupting running instances. `action: "reset"` will clear the traffic statistics (tcprx, tcptx, udprx, udptx) for the instance.
- **Example**:
```javascript
// Update alias and auto-start policy
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    alias: 'Web Server',
    restart: true 
  })
});

// Control instance operations
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ action: 'restart' })
});

// Clear traffic statistics
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ action: 'reset' })
});
```

#### PUT /instances/{id}
- **Description**: Fully update the instance URL configuration
- **Authentication**: API Key required
- **Request Body**: `{ "url": "new client:// or server:// style URL" }`
- **Note**: The instance will be restarted.
- **Restriction**: API Key instance (ID `********`) does not support this operation
- **Example**:
```javascript
// Update instance URL
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PUT',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    url: 'server://0.0.0.0:9090/localhost:8080?tls=1' 
  })
});
```

#### DELETE /instances/{id}
- **Description**: Delete instance
- **Authentication**: API Key required
- **Response**: 204 No Content
- **Restrictions**: API Key instance (ID `********`) cannot be deleted
- **Example**:
```javascript
await fetch(`${API_URL}/instances/abc123`, {
  method: 'DELETE',
  headers: { 'X-API-Key': apiKey }
});
```

### Other Endpoints

#### GET /events
- **Description**: Establish SSE connection to receive real-time events
- **Authentication**: API Key required
- **Response**: Server-Sent Events stream
- **Event Types**: `initial`, `create`, `update`, `delete`, `shutdown`, `log`

#### GET /info
- **Description**: Get master service information
- **Authentication**: API Key required
- **Response**: Contains system info, version, uptime, etc.

#### GET /tcping
- **Description**: TCP connectivity test to check target reachability and latency
- **Authentication**: API Key required
- **Parameters**:
  - `target` (required): Target address in format `host:port`
- **Response**:
  ```json
  {
    "target": "example.com:80",
    "connected": true,
    "latency": 45,
    "error": null
  }
  ```
- **Example**: `GET /api/tcping?target=fast.com:443`

#### GET /openapi.json
- **Description**: Get OpenAPI 3.1.1 specification
- **Authentication**: No authentication required
- **Response**: JSON formatted API specification

#### GET /docs
- **Description**: Swagger UI documentation interface
- **Authentication**: No authentication required
- **Response**: HTML formatted interactive documentation

### Instance URL Format Specification

Instance URLs must follow these formats:

#### Server Mode
```
server://<bind_address>:<bind_port>/<target_host>:<target_port>?<parameters>
```

Examples:
- `server://0.0.0.0:8080/localhost:3000` - Listen on port 8080, forward to local port 3000
- `server://0.0.0.0:9090/localhost:8080?tls=1&mode=1` - Server with TLS enabled, forced reverse mode

#### Client Mode
```
client://<server_host>:<server_port>/<local_host>:<local_port>?<parameters>
```

Examples:
- `client://example.com:8080/localhost:3000` - Connect to remote server, listen locally on port 3000
- `client://remote.example.com:443/localhost:22?mode=2&min=32` - Connect to remote server, forced dual-end mode

#### Supported Parameters

| Parameter | Description | Values | Default | Applicable To |
|-----------|-------------|---------|---------|---------------|
| `log` | Log level | `none`, `debug`, `info`, `warn`, `error`, `event` | `info` | Both |
| `tls` | TLS encryption level | `0`(none), `1`(self-signed), `2`(certificate) | `0` | Server only |
| `crt` | Certificate path | File path | None | Server only |
| `key` | Private key path | File path | None | Server only |
| `mode` | Run mode control | `0`(auto), `1`(force mode 1), `2`(force mode 2) | `0` | Both |
| `min` | Min pool capacity | Integer > 0 | `64` | Client dual-end handshake only |
| `max` | Max pool capacity | Integer > 0 | `1024` | Dual-end handshake mode |
| `read` | Read timeout | Duration (e.g., `10m`, `30s`, `1h`) | `10m` | Both |
| `rate` | Bandwidth rate limit | Integer (Mbps), 0=unlimited | `0` | Both |
