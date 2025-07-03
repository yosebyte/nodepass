# NodePass API Reference

## Overview

NodePass offers a RESTful API in Master Mode that enables programmatic control and integration with frontend applications. This section provides comprehensive d     // Configure auto-start policy for new instance based on type
     if (data.success) {
       const shouldAutoStart = config.type === 'server' || config.critical === true;
       await setAutoStartPolicy(data.data.id, shouldAutoStart);
     }ntation of the API endpoints, integration patterns, and best practices.

## Master Mode API

When running NodePass in Master Mode (`master://`), it exposes a REST API that allows frontend applications to:

1. Create and manage NodePass server and client instances
2. Monitor connection status and statistics
3. Control running instances (start, stop, restart)
4. Configure auto-start policies for automatic instance management
5. Configure behavior through parameters

### Base URL

```
master://<api_addr>/<prefix>?<log>&<tls>
```

Where:
- `<api_addr>` is the address specified in the master mode URL (e.g., `0.0.0.0:9090`)
- `<prefix>` is the optional API prefix (if not specified, `/api` will be used as the prefix)

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
| `/instances` | GET | List all NodePass instances |
| `/instances` | POST | Create a new NodePass instance |
| `/instances/{id}` | GET | Get details about a specific instance |
| `/instances/{id}` | PATCH | Update instance state or control operations |
| `/instances/{id}` | PUT | Update instance URL configuration |
| `/instances/{id}` | DELETE | Remove a specific instance |
| `/events` | GET | Subscribe to instance events using SSE |
| `/info` | GET | Get master service information |
| `/openapi.json` | GET | OpenAPI specification |
| `/docs` | GET | Swagger UI documentation |

### API Authentication

The Master API now supports API Key authentication to prevent unauthorized access. The system automatically generates an API Key on first startup.

#### API Key Features

1. **Automatic Generation**: Created automatically when master mode is first started
2. **Persistent Storage**: The API Key is saved along with other instance configurations in the `nodepass.gob` file
3. **Retention After Restart**: The API Key remains the same after restarting the master
4. **Selective Protection**: Only critical API endpoints are protected, public documentation remains accessible

#### Protected Endpoints

The following endpoints require API Key authentication:
- `/instances` (all methods)
- `/instances/{id}` (all methods: GET, PATCH, PUT, DELETE)
- `/events`
- `/info`

The following endpoints are publicly accessible (no API Key required):
- `/openapi.json`
- `/docs`

#### How to Use the API Key

Include the API Key in your API requests:

```javascript
// Using an API Key for instance management requests
async function getInstances() {
  const response = await fetch(`${API_URL}/instances`, {
    method: 'GET',
    headers: {
      'X-API-Key': 'your-api-key-here'
    }
  });
  
  return await response.json();
}
```

#### How to Get and Regenerate API Key

The API Key can be found in the system startup logs, and can be regenerated using:

```javascript
// Regenerate the API Key (requires knowing the current API Key)
async function regenerateApiKey() {
  const response = await fetch(`${API_URL}/instances/${apiKeyID}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': 'current-api-key'
    },
    body: JSON.stringify({ action: 'restart' })
  });
  
  const result = await response.json();
  return result.url; // The new API Key
}
```

**Note**: The API Key ID is fixed as `********` (eight asterisks). In the internal implementation, this is a special instance ID used to store and manage the API Key.

## Frontend Integration Guidelines

When integrating NodePass with frontend applications, consider the following important points:

### Instance Persistence

NodePass Master Mode now supports instance persistence using the gob serialization format. Instances and their states are saved to a `nodepass.gob` file in the same directory as the executable, and automatically restored when the master restarts.

Key persistence features:
- Instance configurations are automatically saved to disk
- Instance state (running/stopped) is preserved
- Auto-start policies are preserved across master restarts
- Traffic statistics are retained between restarts
- Instances with auto-start policy enabled will automatically start when master restarts
- No need for manual re-registration after restart

**Note:** While instance configurations are now persisted, frontend applications should still maintain their own record of instance configurations as a backup strategy.

### Instance ID Persistence

With NodePass now using gob format for persistent storage of instance state, instance IDs **no longer change** after a master restart. This means:

1. Frontend applications can safely use instance IDs as unique identifiers
2. Instance configurations, states, and statistics are automatically restored after restart
3. No need to implement logic for handling instance ID changes

This greatly simplifies frontend integration by eliminating the previous complexity of handling instance recreation and ID mapping.

### Auto-start Policy Management

NodePass now supports configurable auto-start policies for instances, allowing for automatic instance management and improved reliability. The auto-start policy feature enables:

1. **Automatic Instance Recovery**: Instances with auto-start policy enabled will automatically start when the master service restarts
2. **Selective Auto-start**: Configure which instances should auto-start based on their importance or role
3. **Persistent Policy Storage**: Auto-start policies are saved and restored across master restarts
4. **Fine-grained Control**: Each instance can have its own auto-start policy setting

#### How Auto-start Policy Works

- **Policy Assignment**: Each instance has a `restart` boolean field that determines its auto-start behavior
- **Master Startup**: When the master starts, it automatically launches all instances with `restart: true`
- **Policy Persistence**: Auto-start policies are saved in the same `nodepass.gob` file as other instance data
- **Runtime Management**: Auto-start policies can be modified while instances are running

#### Best Practices for Auto-start Policy

1. **Enable for Server Instances**: Server instances typically should have auto-start policy enabled for high availability
2. **Selective Client Auto-start**: Enable auto-start policy for critical client connections only
3. **Testing Scenarios**: Disable auto-start policy for temporary or testing instances
4. **Load Balancing**: Use auto-start policies to maintain minimum instance counts for load distribution

```javascript
// Example: Configure auto-start policies based on instance role
async function configureAutoStartPolicies(instances) {
  for (const instance of instances) {
    // Enable auto-start for servers and critical clients
    const shouldAutoStart = instance.type === 'server' || 
                            instance.tags?.includes('critical');
    
    await setAutoStartPolicy(instance.id, shouldAutoStart);
  }
}
```

### Instance Lifecycle Management

For proper lifecycle management:

1. **Creation**: Store instance configurations and URLs
   ```javascript
   async function createNodePassInstance(config) {
     const response = await fetch(`${API_URL}/instances`, {
       method: 'POST',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({
         url: `server://0.0.0.0:${config.port}/${config.target}?tls=${config.tls}`
       })
     });
     
     const data = await response.json();
     
     // Configure restart policy for new instance based on type
     if (data.success) {
       const shouldAutoRestart = config.type === 'server' || config.critical === true;
       await setRestartPolicy(data.data.id, shouldAutoRestart);
     }
     
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
   
   B. **Traditional Polling (Alternative)**: Use in environments where SSE is not supported
   ```javascript
   function startInstanceMonitoring(instanceId, interval = 5000) {
     return setInterval(async () => {
       try {
         const response = await fetch(`${API_URL}/instances/${instanceId}`);
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

3. **Instance Alias Management**: Set readable names for instances
   ```javascript
   // Batch set instance aliases
   async function setInstanceAliases(instances) {
     for (const instance of instances) {
       // Generate meaningful aliases based on instance type and purpose
       const alias = `${instance.type}-${instance.region || 'default'}-${instance.port || 'auto'}`;
       await updateInstanceAlias(instance.id, alias);
     }
   }
   
   // Find instance by alias
   async function findInstanceByAlias(targetAlias) {
     const response = await fetch(`${API_URL}/instances`, {
       headers: { 'X-API-Key': apiKey }
     });
     const data = await response.json();
     
     if (data.success) {
       return data.data.find(instance => instance.alias === targetAlias);
     }
     return null;
   }
   ```

4. **Control Operations**: Start, stop, restart instances
   ```javascript
   async function controlInstance(instanceId, action) {
     // action can be: start, stop, restart
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',  // Note: API has been updated to use PATCH instead of PUT
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({ action })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // Update instance alias
   async function updateInstanceAlias(instanceId, alias) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // If API Key is enabled 
       },
       body: JSON.stringify({ alias })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // Update instance URL configuration
   async function updateInstanceURL(instanceId, newURL) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PUT',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // If API Key is enabled 
       },
       body: JSON.stringify({ url: newURL })
     });
     
     const data = await response.json();
     return data.success;
   }
   ```

5. **Auto-start Policy Management**: Configure automatic startup behavior
   ```javascript
   async function setAutoStartPolicy(instanceId, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({ restart: enableAutoStart })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // Combined operation: control instance and update auto-start policy
   async function controlInstanceWithAutoStart(instanceId, action, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({ 
         action: action,
         restart: enableAutoStart 
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // Combined operation: update alias, control instance and auto-start policy
   async function updateInstanceComplete(instanceId, alias, action, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({ 
         alias: alias,
         action: action,
         restart: enableAutoStart 
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   ```

#### Complete Auto-start Policy Usage Example

Here's a comprehensive example showing how to implement auto-start policy management in a real-world scenario:

```javascript
// Scenario: Setting up a load-balanced server cluster with auto-start policies
async function setupServerCluster(serverConfigs) {
  const clusterInstances = [];
  
  for (const config of serverConfigs) {
    try {
      // Create server instance
      const instance = await createNodePassInstance({
        type: 'server',
        port: config.port,
        target: config.target,
        critical: config.isPrimary, // Primary servers are critical
        tls: config.enableTLS
      });
      
      if (instance.success) {
        // Set meaningful instance alias
        const alias = `${config.role}-server-${config.port}`;
        await updateInstanceAlias(instance.data.id, alias);
        
        // Configure auto-start policy based on server role
        const autoStartPolicy = config.isPrimary || config.role === 'essential';
        await setAutoStartPolicy(instance.data.id, autoStartPolicy);
        
        // Start the instance
        await controlInstance(instance.data.id, 'start');
        
        clusterInstances.push({
          id: instance.data.id,
          alias: alias,
          role: config.role,
          autoStartEnabled: autoStartPolicy
        });
        
        console.log(`Server ${alias} created with auto-start policy: ${autoStartPolicy}`);
      }
    } catch (error) {
      console.error(`Failed to create server ${config.role}:`, error);
    }
  }
  
  return clusterInstances;
}

// Monitor cluster health and adjust auto-start policies dynamically
async function monitorClusterHealth(clusterInstances) {
  const healthyInstances = [];
  
  for (const cluster of clusterInstances) {
    const instance = await fetch(`${API_URL}/instances/${cluster.id}`);
    const data = await instance.json();
    
    if (data.success && data.data.status === 'running') {
      healthyInstances.push(cluster);
    } else {
      // If a critical instance is down, enable auto-start for backup instances
      if (cluster.role === 'primary') {
        await enableBackupInstanceAutoStart(clusterInstances);
      }
    }
  }
  
  return healthyInstances;
}

async function enableBackupInstanceAutoStart(clusterInstances) {
  const backupInstances = clusterInstances.filter(c => c.role === 'backup');
  for (const backup of backupInstances) {
    await setAutoStartPolicy(backup.id, true);
    console.log(`Enabled auto-start policy for backup instance: ${backup.id}`);
  }
}
```

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
- **Description**: Update instance state, alias, or perform control operations
- **Authentication**: API Key required
- **Request Body**: `{ "alias": "new alias", "action": "start|stop|restart", "restart": true|false }`
- **Features**: Does not interrupt running instances, only updates specified fields
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
```

#### PUT /instances/{id}
- **Description**: Completely update instance URL configuration
- **Authentication**: API Key required
- **Request Body**: `{ "url": "new client:// or server:// format URL" }`
- **Features**: Will restart instance and reset traffic statistics
- **Restrictions**: API Key instance (ID `********`) does not support this operation
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
- `server://0.0.0.0:9090/localhost:8080?tls=1` - Server with TLS enabled

#### Client Mode
```
client://<server_host>:<server_port>/<local_host>:<local_port>?<parameters>
```

Examples:
- `client://example.com:8080/localhost:3000` - Connect to remote server, listen locally on port 3000
- `client://vpn.example.com:443/localhost:22?tls=1` - Connect to VPN server via TLS

#### Supported Parameters

| Parameter | Description | Values | Default |
|-----------|-------------|---------|---------|
| `tls` | TLS encryption level | `0`(none), `1`(self-signed), `2`(certificate) | `0` |
| `log` | Log level | `trace`, `debug`, `info`, `warn`, `error` | `info` |
| `crt` | Certificate path | File path | None |
| `key` | Private key path | File path | None |
