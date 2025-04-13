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
http(s)://<api_addr>[<prefix>]/v1/
```

Where:
- `<api_addr>` is the address specified in the master mode URL (e.g., `0.0.0.0:9090`)
- `<prefix>` is the optional API prefix (defaults to `/api`)

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
| `/v1/instances/{id}` | PUT | Update or control a specific instance |
| `/v1/instances/{id}` | DELETE | Remove a specific instance |
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

**Important:** NodePass Master Mode **does not persist instance configurations** between restarts. When the Master Mode process restarts, all instance information is lost.

Frontend applications should:
1. Store instance configurations in their own persistent storage
2. Re-register all instances when detecting a NodePass Master restart
3. Compare returned instance IDs with stored IDs to detect and handle restarts

Example re-registration logic:
```javascript
function checkAndRestoreInstances() {
  try {
    // Simple health check to detect if master is running
    const response = await fetch(`${API_URL}/v1/instances`);
    
    if (response.status === 200) {
      const data = await response.json();
      
      // If no instances but we have stored configs, master likely restarted
      if (data.data.instances.length === 0 && storedInstances.length > 0) {
        console.log("Detected master restart, re-registering instances...");
        
        for (const instance of storedInstances) {
          const newInstance = await createInstance(instance.url);
          // Update stored ID with newly assigned ID
          updateStoredInstanceId(instance.id, newInstance.data.id);
        }
      }
    }
  } catch (error) {
    console.error("NodePass master unreachable:", error);
  }
}
```

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

2. **Status Monitoring**: Poll status periodically
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

3. **Control Operations**: Start, stop, restart instances
   ```javascript
   async function controlInstance(instanceId, action) {
     // action can be: start, stop, restart
     const response = await fetch(`${API_URL}/v1/instances/${instanceId}`, {
       method: 'PUT',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({ action })
     });
     
     const data = await response.json();
     return data.success;
   }
   ```

### Traffic Statistics

The Master API provides basic traffic statistics, but these are limited in scope:

1. **Basic Traffic Metrics**: NodePass only periodically provides cumulative TCP and UDP traffic values in both inbound and outbound directions. The frontend application needs to store and process these values to derive meaningful statistics.
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

### Instance ID Changes

Instance IDs will change after Master Mode restarts. To handle this:

1. **Track by URL**: Use the instance URL as the stable identifier
   ```javascript
   function findInstanceByUrl(url) {
     return storedInstances.find(instance => instance.url === url);
   }
   ```

2. **ID Mapping**: Maintain a mapping between your application's stable IDs and NodePass instance IDs
   ```javascript
   const instanceMapping = {};
   
   function updateInstanceMapping(appInstanceId, nodepassInstanceId) {
     instanceMapping[appInstanceId] = nodepassInstanceId;
   }
   
   function getNodePassId(appInstanceId) {
     return instanceMapping[appInstanceId];
   }
   ```

3. **Recovery Procedure**: Implement a recovery procedure for when IDs change
   ```javascript
   async function recoverInstances() {
     // Get all current instances from NodePass
     const response = await fetch(`${API_URL}/v1/instances`);
     const data = await response.json();
     
     // Match instances by URL
     for (const storedInstance of storedInstances) {
       const matchingInstance = data.data.instances.find(
         instance => instance.url === storedInstance.url
       );
       
       if (matchingInstance) {
         // Update the ID mapping
         updateInstanceMapping(storedInstance.appId, matchingInstance.id);
       } else {
         // Instance doesn't exist, recreate it
         const newInstance = await createInstance(storedInstance.url);
         updateInstanceMapping(storedInstance.appId, newInstance.data.id);
       }
     }
   }
   ```

## Response Format

All API responses follow a standard format:

**Success Response:**
```json
{
  "success": true,
  "data": {
    // Response data varies by endpoint
  },
  "error": null
}
```

**Error Response:**
```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message"
  }
}
```

## API Endpoint Examples

### List All Instances

**Request:**
```
GET /api/v1/instances
```

**Response:**
```json
{
  "success": true,
  "data": {
    "instances": [
      {
        "id": "ins_abc123",
        "url": "server://0.0.0.0:10101/0.0.0.0:8080?tls=1",
        "status": "running",
        "created_at": "2025-04-10T15:30:45Z",
        "connections": 12,
        "pool_size": 32
      },
      {
        "id": "ins_def456",
        "url": "client://server.example.com:10101/127.0.0.1:8080",
        "status": "stopped",
        "created_at": "2025-04-11T09:15:22Z",
        "connections": 0,
        "pool_size": 0
      }
    ]
  },
  "error": null
}
```

### Create a New Instance

**Request:**
```
POST /api/v1/instances
Content-Type: application/json

{
  "url": "server://0.0.0.0:10101/0.0.0.0:8080?tls=1"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "ins_ghi789",
    "url": "server://0.0.0.0:10101/0.0.0.0:8080?tls=1",
    "status": "running",
    "created_at": "2025-04-13T10:25:15Z"
  },
  "error": null
}
```

### Control an Instance

**Request:**
```
PUT /api/v1/instances/ins_ghi789
Content-Type: application/json

{
  "action": "restart"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "ins_ghi789",
    "status": "running",
    "restarted_at": "2025-04-13T10:30:45Z"
  },
  "error": null
}
```

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
2. **Instance ID changes** - Implement stable identification strategies
3. **Proper error handling** - Gracefully recover from API errors
4. **Traffic statistics** - Collect and visualize connection metrics

These guidelines will help you build robust integrations between your frontend applications and NodePass.

For information about the internal mechanisms of NodePass, see the [How It Works](/docs/en/how-it-works.md) section, which includes details about:
- Connection Pooling
- Signal Communication Protocol
- Data Transmission