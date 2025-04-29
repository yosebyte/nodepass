# NodePass API Reference

## Overview

NodePass offers a RESTful API in Master Mode that enables programmatic control and integration with frontend applications. This section provides comprehensive documentation of the API endpoints, integration patterns, and best practices.

## Master Mode API

When running NodePass in Master Mode (`master://`), it exposes a REST API that allows frontend applications to:

1. Create and manage NodePass server and client instances
2. Monitor connection status and statistics
3. Control running instances (start, stop, restart)
4. Configure behavior through parameters

### Base URL

```
master://<api_addr>/<prefix>?<log>&<tls>
```

Where:
- `<api_addr>` is the address specified in the master mode URL (e.g., `0.0.0.0:9090`)
- `<prefix>` is the optional API prefix (if not specified, a randomly generated ID will be used as the prefix)

**Note:** If no custom prefix is specified, the system will automatically generate a random prefix for enhanced security. The generated prefix will be displayed in the startup logs.

### Starting Master Mode

To start NodePass in Master Mode with default settings:

```bash
nodepass "master://0.0.0.0:9090?log=info"
```

With custom API prefix and TLS enabled:

```bash
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"
```

### Available Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/instances` | GET | List all NodePass instances |
| `/v1/instances` | POST | Create a new NodePass instance |
| `/v1/instances/{id}` | GET | Get details about a specific instance |
| `/v1/instances/{id}` | PATCH | Update or control a specific instance |
| `/v1/instances/{id}` | DELETE | Remove a specific instance |
| `/v1/events` | GET | Subscribe to instance events using SSE |
| `/v1/openapi.json` | GET | OpenAPI specification |
| `/v1/docs` | GET | Swagger UI documentation |

### API Authentication

The Master API currently does not implement authentication. When deploying in production environments, it's recommended to:
- Use a reverse proxy with authentication
- Restrict access using network policies
- Enable TLS encryption (`tls=1` or `tls=2`)

## Frontend Integration Guidelines

When integrating NodePass with frontend applications, consider the following important points:

### Instance Persistence

NodePass Master Mode now supports instance persistence using the gob serialization format. Instances and their states are saved to a `nodepass.gob` file in the same directory as the executable, and automatically restored when the master restarts.

Key persistence features:
- Instance configurations are automatically saved to disk
- Instance state (running/stopped) is preserved
- Traffic statistics are retained between restarts
- No need for manual re-registration after restart

**Note:** While instance configurations are now persisted, frontend applications should still maintain their own record of instance configurations as a backup strategy.

### Instance ID Persistence

With NodePass now using gob format for persistent storage of instance state, instance IDs **no longer change** after a master restart. This means:

1. Frontend applications can safely use instance IDs as unique identifiers
2. Instance configurations, states, and statistics are automatically restored after restart
3. No need to implement logic for handling instance ID changes

This greatly simplifies frontend integration by eliminating the previous complexity of handling instance recreation and ID mapping.

### Instance Lifecycle Management

For proper lifecycle management:

1. **Creation**: Store instance configurations and URLs
   ```javascript
   async function createNodePassInstance(config) {
     const response = await fetch(`${API_URL}/v1/instances`, {
       method: 'POST',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({
         url: `server://0.0.0.0:${config.port}/${config.target}?tls=${config.tls}`
       })
     });
     
     const data = await response.json();
     
     // Store in frontend persistence
     saveInstanceConfig({
       id: data.data.id,
       originalConfig: config,
       url: data.data.url
     });
     
     return data;
   }
   ```

2. **Status Monitoring**: Monitor instance state changes
   
   NodePass provides two methods for monitoring instance status:
   
   A. **Using SSE (Recommended)**: Receive real-time events via persistent connection
   ```javascript
   function connectToEventSource() {
     const eventSource = new EventSource(`${API_URL}/v1/events`);
     
     eventSource.addEventListener('instance', (event) => {
       const data = JSON.parse(event.data);
       // Process different event types: initial, create, update, delete
       // ...see the "Real-time Event Monitoring with SSE" section for implementation details
     });
     
     // Error handling and reconnection logic
     // ...see previous example
     
     return eventSource;
   }
   ```
   
   B. **Traditional Polling (Alternative)**: Use in environments where SSE is not supported
   ```javascript
   function startInstanceMonitoring(instanceId, interval = 5000) {
     return setInterval(async () => {
       try {
         const response = await fetch(`${API_URL}/v1/instances/${instanceId}`);
         const data = await response.json();
         
         if (data.success) {
           updateInstanceStatus(instanceId, data.data.status);
           updateInstanceMetrics(instanceId, {
             connections: data.data.connections,
             pool_size: data.data.pool_size,
             uptime: data.data.uptime
           });
         }
       } catch (error) {
         markInstanceUnreachable(instanceId);
       }
     }, interval);
   }
   ```

   **Recommendation:** Prefer the SSE approach as it provides more efficient real-time monitoring and reduces server load. Only use the polling approach for client environments with specific compatibility needs or where SSE is not supported.

3. **Control Operations**: Start, stop, restart instances
   ```javascript
   async function controlInstance(instanceId, action) {
     // action can be: start, stop, restart
     const response = await fetch(`${API_URL}/v1/instances/${instanceId}`, {
       method: 'PATCH',  // Note: API has been updated to use PATCH instead of PUT
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({ action })
     });
     
     const data = await response.json();
     return data.success;
   }
   ```

### Real-time Event Monitoring with SSE

NodePass now supports Server-Sent Events (SSE) for real-time monitoring of instance state changes. This allows frontend applications to receive instant notifications about instance creation, updates, and deletions without polling.

#### Using the SSE Endpoint

The SSE endpoint is available at:
```
GET /v1/events
```

This endpoint establishes a persistent connection that delivers events in real-time using the SSE protocol format.

#### Event Types

The following event types are supported:

1. `initial` - Sent when a connection is established, containing the current state of all instances
2. `create` - Sent when a new instance is created
3. `update` - Sent when an instance is updated (status change, start/stop operations)
4. `delete` - Sent when an instance is deleted
5. `shutdown` - Sent when the master service is about to shut down, notifying frontend applications to close their connections

#### JavaScript Client Implementation

Here's an example of how to consume the SSE endpoint in a JavaScript frontend:

```javascript
function connectToEventSource() {
  const eventSource = new EventSource(`${API_URL}/v1/events`);
  
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

// 初始化SSE连接
const eventSource = connectToEventSource();

// 在应用程序关闭时清理连接
function cleanup() {
  if (eventSource) {
    eventSource.close();
  }
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

1. **Enable Debug Mode**: Traffic statistics are only available when debug mode is enabled. 

   ```bash
   # Master with debug mode enabled
   nodepass master://0.0.0.0:10101?log=debug
   ```

   Without enabling debug mode, traffic statistics will not be collected or returned by the API.

2. **Basic Traffic Metrics**: NodePass periodically provides cumulative TCP and UDP traffic values in both inbound and outbound directions. The frontend application needs to store and process these values to derive meaningful statistics.
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

3. **Data Persistence**: Since the API only provides cumulative values, the frontend must implement proper storage and calculation logic
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

## API Endpoint Documentation

For detailed API documentation including request and response examples, please use the built-in Swagger UI documentation available at the `/v1/docs` endpoint. This interactive documentation provides comprehensive information about:

- Available endpoints
- Required parameters
- Response formats
- Example requests and responses
- Schema definitions

### Accessing Swagger UI

To access the Swagger UI documentation:

```
http(s)://<api_addr>[<prefix>]/v1/docs
```

For example:
```
http://localhost:9090/api/v1/docs
```

![np-api](https://cdn.yobc.de/assets/np-api.png)

The Swagger UI provides a convenient way to explore and test the API directly in your browser. You can execute API calls against your running NodePass Master instance and see the actual responses.

## Best Practices

### Scalable Management

For managing many NodePass instances:

1. **Bulk Operations**: Implement batch operations for managing multiple instances
   ```javascript
   async function bulkControlInstances(instanceIds, action) {
     const promises = instanceIds.map(id => controlInstance(id, action));
     return Promise.all(promises);
   }
   ```

2. **Connection Pooling**: Use a connection pool for API requests
   ```javascript
   const http = require('http');
   const agent = new http.Agent({ keepAlive: true, maxSockets: 50 });
   
   async function optimizedFetch(url, options = {}) {
     return fetch(url, { ...options, agent });
   }
   ```

3. **Caching**: Cache instance details to reduce API calls
   ```javascript
   const instanceCache = new Map();
   const CACHE_TTL = 60000; // 1 minute
   
   async function getCachedInstance(id) {
     const now = Date.now();
     const cached = instanceCache.get(id);
     
     if (cached && now - cached.timestamp < CACHE_TTL) {
       return cached.data;
     }
     
     const response = await fetch(`${API_URL}/v1/instances/${id}`);
     const data = await response.json();
     
     instanceCache.set(id, {
       data: data.data,
       timestamp: now
     });
     
     return data.data;
   }
   ```

### Monitoring and Health Checks

Implement comprehensive monitoring:

1. **API Health Check**: Verify the Master API is responsive
   ```javascript
   async function isApiHealthy() {
     try {
       const response = await fetch(`${API_URL}/v1/instances`, {
         method: 'GET',
         timeout: 5000 // 5 second timeout
       });
       
       return response.status === 200;
     } catch (error) {
       return false;
     }
   }
   ```

2. **Instance Health Check**: Monitor individual instance health
   ```javascript
   async function checkInstanceHealth(id) {
     try {
       const response = await fetch(`${API_URL}/v1/instances/${id}`);
       const data = await response.json();
       
       if (!data.success) return false;
       
       return data.data.status === 'running';
     } catch (error) {
       return false;
     }
   }
   ```

## Summary

The NodePass Master Mode API provides a powerful interface for programmatic management of NodePass instances. When integrating with frontend applications, be particularly mindful of:

1. **Instance persistence** - Store configurations and handle restarts
2. **Instance ID persistence** - Use instance IDs as stable identifiers
3. **Proper error handling** - Gracefully recover from API errors
4. **Traffic statistics** - Collect and visualize connection metrics (requires debug mode)

These guidelines will help you build robust integrations between your frontend applications and NodePass.

For information about the internal mechanisms of NodePass, see the [How It Works](/docs/en/how-it-works.md) section, which includes details about:
- Connection Pooling
- Signal Communication Protocol
- Data Transmission
