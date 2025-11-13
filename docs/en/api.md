# NodePass API Reference

## Overview

NodePass provides RESTful API in Master Mode, supporting frontend integration and automation. This document covers all interfaces, data structures, and best practices.

## Master Mode API

In Master Mode (`master://`), NodePass supports:

1. Creating and managing server/client instances
2. Real-time monitoring of status, traffic, and health checks
3. Instance control (start, stop, restart, reset traffic)
4. Auto-restart policy configuration
5. Flexible parameter configuration

### Base URL

```
master://<api_addr>/<prefix>?<log>&<tls>
```

- `<api_addr>`: Listen address (e.g., `0.0.0.0:9090`)
- `<prefix>`: API path prefix (default `/api`)

### Starting Master Mode

```bash
nodepass "master://0.0.0.0:9090?log=info"
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"
```

### Main Endpoints

| Endpoint           | Method | Description              |
|--------------------|--------|--------------------------|
| `/instances`       | GET    | Get all instances        |
| `/instances`       | POST   | Create new instance      |
| `/instances/{id}`  | GET    | Get instance details     |
| `/instances/{id}`  | PATCH  | Update/control instance  |
| `/instances/{id}`  | PUT    | Update instance URL      |
| `/instances/{id}`  | DELETE | Delete instance          |
| `/events`          | GET    | SSE real-time event stream |
| `/info`            | GET    | Get master service info  |
| `/info`            | POST   | Update master alias      |
| `/tcping`          | GET    | TCP connection test      |
| `/openapi.json`    | GET    | OpenAPI specification    |
| `/docs`            | GET    | Swagger UI documentation |

### API Authentication

API Key authentication is enabled by default, automatically generated and saved in `nodepass.gob` on first startup.

- Protected endpoints: `/instances`, `/instances/{id}`, `/events`, `/info`, `/tcping`
- Public endpoints: `/openapi.json`, `/docs`
- Authentication method: Add `X-API-Key: <key>` to request headers
- Reset Key: PATCH `/instances/********`, body `{ "action": "restart" }`

### Instance Data Structure

```json
{
  "id": "a1b2c3d4",
  "alias": "alias",
  "type": "client|server",
  "status": "running|stopped|error",
  "url": "...",
  "config": "server://0.0.0.0:8080/localhost:3000?log=info&tls=1&max=1024&mode=0&read=1h&rate=0&slot=65536&proxy=0",
  "restart": true,
  "meta": {
    "peer": {
      "sid": "550e8400-e29b-41d4-a716-446655440000",
      "type": "1",
      "alias": "remote-service"
    },
    "tags": {
      "environment": "production",
      "region": "us-west",
      "owner": "team-alpha"
    }
  },
  "mode": 0,
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

- `mode`: Instance mode
- `ping`/`pool`: Health check data
- `tcps`/`udps`: Current active connection count statistics
- `tcprx`/`tcptx`/`udprx`/`udptx`: Cumulative traffic statistics
- `config`: Instance configuration URL with complete startup configuration
- `restart`: Auto-restart policy
- `meta`: Metadata information for instance organization and peer identification
  - `peer`: Peer connection information (remote endpoint details)
    - `sid`: Service ID of the remote service, using UUID v4 format (e.g., `550e8400-e29b-41d4-a716-446655440000`)
    - `type`: Remote service type, using standard enumeration values
      - `"0"`: Single-end Forwarding mode
      - `"1"`: NAT Traversal mode
      - `"2"`: Tunnel Forwarding mode
    - `alias`: Service alias of the remote endpoint (no format restriction)
  - `tags`: Custom key-value tags for flexible categorization and filtering

### Instance URL Format

- Server: `server://<bind_addr>:<bind_port>/<target_host>:<target_port>?<parameters>`
- Client: `client://<server_host>:<server_port>/<local_host>:<local_port>?<parameters>`
- Supported parameters: `log`, `tls`, `crt`, `key`, `min`, `max`, `mode`, `read`, `rate`

### URL Query Parameters

- `log`: Log level (`none`, `debug`, `info`, `warn`, `error`, `event`)
- `tls`: TLS encryption mode (`0`, `1`, `2`) - Server/Master mode only
- `crt`/`key`: Certificate/key file paths (when `tls=2`)
- `min`/`max`: Connection pool capacity (`min` set by client, `max` set by server and passed to client during handshake)
- `mode`: Runtime mode control (`0`, `1`, `2`) - Controls operation behavior
  - For server: `0`=auto, `1`=reverse mode, `2`=forward mode
  - For client: `0`=auto, `1`=single-end forwarding, `2`=dual-end handshake
- `read`: Data read timeout duration (e.g., 1h, 30m, 15s)
- `rate`: Bandwidth rate limit in Mbps (0=unlimited)
- `proxy`: PROXY protocol support (`0`, `1`) - When enabled, sends PROXY protocol v1 header before data transmission

### Real-time Event Stream (SSE)

- Event types: `initial`, `create`, `update`, `delete`, `shutdown`, `log`
- `log` events only push normal logs, traffic/health check logs are filtered
- Connect to `/events` for real-time instance changes and logs

### Additional Notes

- All instances, traffic, health checks, aliases, and auto-restart policies are persistently stored and automatically restored after restart
- Detailed API specification at `/openapi.json`, Swagger UI at `/docs`

```javascript
// Regenerate API Key (requires knowing current API Key)
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
  return result.url; // New API Key
}

// Get Master ID
async function getMasterID() {
  const response = await fetch(`${API_URL}/instances/${apiKeyID}`, {
    method: 'GET',
    headers: {
      'X-API-Key': 'current-api-key'
    }
  });
  
  const result = await response.json();
  return result.data.config; // Master ID (16-character hex)
}
```

**Note**: 
- API Key ID is fixed as `********` (eight asterisks). In the internal implementation, this is a special instance ID used to store and manage the API Key.
- The API Key instance's `config` field stores the **Master ID**, which is a 16-character hexadecimal string (e.g., `1a2b3c4d5e6f7890`) used to uniquely identify the master service.
- The Master ID is automatically generated on first startup and persisted, remaining constant throughout the master service's lifecycle.

### Using SSE for Real-time Event Monitoring

NodePass now supports Server-Sent Events (SSE) functionality for real-time monitoring of instance state changes. This enables frontend applications to receive immediate notifications of instance creation, updates, and deletions without the need for polling.

#### Using the SSE Endpoint

The SSE endpoint is located at:
```
GET /events
```

This endpoint establishes a persistent connection and delivers events in real-time using the SSE protocol format. If API Key authentication is enabled, you need to include a valid API Key in the request headers.

#### Event Types

The following event types are supported:

1. `initial` - Sent when connection is established, contains current state of all instances
2. `create` - Sent when a new instance is created
3. `update` - Sent when an instance is updated (state changes, start/stop operations)
4. `delete` - Sent when an instance is deleted
5. `shutdown` - Sent when the master service is about to shut down, notifying frontend applications to close connections
6. `log` - Sent when an instance produces new log content, contains log text

#### Handling Instance Logs

In frontend applications, you can handle instance logs by listening to `log` events. Here's an example function to append logs to a specific instance's UI:

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

  // You can parse ANSI color codes or format logs here
  logEntry.textContent = logText;

  // Add to container
  logContainer.appendChild(logEntry);

  // Scroll to latest log
  logContainer.scrollTop = logContainer.scrollHeight;
}
```

Best practices for log integration:

1. **Buffer management**: Limit the number of log entries to prevent memory issues
2. **ANSI color parsing**: Parse ANSI color codes in logs for better readability
3. **Filtering options**: Provide options to filter logs by severity or content
4. **Search functionality**: Allow users to search within instance logs
5. **Log persistence**: Optionally save logs to local storage for viewing after page refresh

#### JavaScript Client Implementation

Here's an example of consuming the SSE endpoint in JavaScript frontend:

```javascript
function connectToEventSource() {
  const eventSource = new EventSource(`${API_URL}/events`, {
    // If authentication is needed, native EventSource doesn't support custom headers
    // Need to use fetch API to implement custom SSE client
  });
  
  // If using API Key, need to use custom implementation instead of native EventSource
  // Below is an example using native EventSource
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
        // Close event source and show notification
        eventSource.close();
        showShutdownNotification();
        break;
    }
  });
  
  eventSource.addEventListener('error', (error) => {
    console.error('SSE connection error:', error);
    // Try to reconnect after delay
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
            // Handle events - see switch code above
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

#### Advantages of SSE over Polling

Using SSE for monitoring instance state provides several advantages over traditional polling:

1. **Reduced latency**: Changes are delivered in real-time
2. **Reduced server load**: Eliminates unnecessary polling requests
3. **Bandwidth efficiency**: Data is only sent when changes occur
4. **Native browser support**: Built-in browser support without additional libraries
5. **Automatic reconnection**: Browser automatically reconnects when connection is lost

#### Best Practices for SSE Implementation

When implementing SSE in frontend:

1. **Handle reconnection**: While browsers automatically attempt reconnection, implement custom logic to ensure persistent connection
2. **Efficient event handling**: Keep event handlers fast to avoid UI blocking
3. **Implement fallback**: Implement polling fallback for environments that don't support SSE
4. **Handle errors**: Properly handle connection errors and disconnections
5. **Log management**: Maintain log buffers for each instance to prevent unlimited growth

## Frontend Integration Guide

When integrating NodePass with frontend applications, consider the following important aspects:

### Instance Persistence

NodePass Master Mode now supports instance persistence using gob serialization format. Instances and their states are saved to a `nodepass.gob` file in the same directory as the executable and automatically restored when the master restarts.

Key persistence features:
- Instance configurations are automatically saved to disk
- Instance states (running/stopped) are preserved
- Auto-restart policies are maintained across master restarts
- Traffic statistics are maintained between restarts
- Instances with auto-restart enabled automatically start when master restarts
- No need to manually re-register after restart

#### Automatic Backup Feature

NodePass Master Mode provides automatic backup functionality to periodically backup state files to prevent data loss:

- **Backup File**: Automatically creates `nodepass.gob.backup` backup file
- **Backup Interval**: Automatically backs up every 1 hour (configurable via `NP_RELOAD_INTERVAL` environment variable)
- **Backup Strategy**: Uses single backup file, new backups overwrite old backups
- **Backup Content**: Includes all instance configurations, states, auto-restart policies, and statistics
- **Disaster Recovery**: When the main file is corrupted, backup file can be manually used for recovery
- **Auto Start**: Backup functionality starts automatically with master service, no additional configuration required

Backup file location: `nodepass.gob.backup` in the same directory as the main state file `nodepass.gob`

**Note:** While instance configurations are now persistent and automatically backed up, frontend applications should still maintain their own instance configuration records as an additional backup strategy.

### Instance Lifecycle Management

To properly manage lifecycles:

1. **Creation**: Store instance configuration and URL
   ```javascript
   async function createNodePassInstance(config) {
     const response = await fetch(`${API_URL}/instances`, {
       method: 'POST',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // If API Key is enabled
       },
       body: JSON.stringify({
         url: `server://0.0.0.0:${config.port}/${config.target}?tls=${config.tls}`
       })
     });
     
     const data = await response.json();
     
     // Configure auto-restart policy based on type
     if (data.success) {
       const shouldAutoRestart = config.type === 'server' || config.critical === true;
       await setAutoStartPolicy(data.data.id, shouldAutoRestart);
     }
     
     // Store in frontend persistence storage
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
   
   A. **Using SSE (Recommended)**: Receive real-time events through persistent connection
   ```javascript
   function connectToEventSource() {
     const eventSource = new EventSource(`${API_URL}/events`, {
       // If authentication is needed, use custom implementation
     });
     
     // Or use custom implementation with API Key
     // connectToEventSourceWithApiKey(apiKey);
     
     eventSource.addEventListener('instance', (event) => {
       const data = JSON.parse(event.data);
       // Handle different event types: initial, create, update, delete, log
       // ...handling logic see "Using SSE for Real-time Event Monitoring" section above
     });
     
     // Error handling and reconnection logic
     // ...see previous examples
     
     return eventSource;
   }
   ```
   
   B. **Traditional Polling (Alternative)**: Use in environments that don't support SSE
   ```javascript
   function startInstanceMonitoring(instanceId, interval = 5000) {
     return setInterval(async () => {
       try {
         const response = await fetch(`${API_URL}/instances/${instanceId}`, {
           headers: {
             'X-API-Key': apiKey // If API Key is enabled
           }
         });
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

   **Recommendation:** Prioritize SSE method as it provides more efficient real-time monitoring and reduces server load. Only use polling when client doesn't support SSE or specific environment compatibility is needed.

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
       method: 'PATCH',  // Note: API has been updated to use PATCH method instead of PUT
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // If API Key is enabled 
       },
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
   
   // Update instance metadata
   async function updateInstanceMetadata(instanceId, metadata) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // If API Key is enabled 
       },
       body: JSON.stringify({ meta: metadata })
     });
     
     const data = await response.json();
     return data.success;
   }
   ```

5. **Metadata Management**: Organize and categorize instances with metadata
   ```javascript
   // Set peer connection information
   async function setPeerInfo(instanceId, peerInfo) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey
       },
       body: JSON.stringify({
         meta: {
           peer: {
             sid: peerInfo.serviceId, // UUID v4 format
             type: peerInfo.type, // "0" | "1" | "2"
             alias: peerInfo.alias
           },
           tags: {} // Preserve existing tags
         }
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // Add or update instance tags
   async function updateInstanceTags(instanceId, tags) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey
       },
       body: JSON.stringify({
         meta: {
           peer: {}, // Preserve existing peer info
           tags: tags
         }
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // Complete metadata update
   async function updateCompleteMetadata(instanceId, peerInfo, tags) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey
       },
       body: JSON.stringify({
         meta: {
           peer: peerInfo,
           tags: tags
         }
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   ```

6. **Auto-restart Policy Management**: Configure automatic startup behavior
   ```javascript
   async function setAutoStartPolicy(instanceId, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey
       },
       body: JSON.stringify({ restart: enableAutoStart })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   async function controlInstanceWithAutoStart(instanceId, action, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey
       },
       body: JSON.stringify({ 
         action: action,
         restart: enableAutoStart 
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   async function updateInstanceComplete(instanceId, alias, action, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey
       },
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

#### Metadata Management Usage Examples

Here are comprehensive examples showing how to use metadata for instance organization and management:

```javascript
// Example 1: Establish peer-to-peer tunnel with metadata
async function establishPeerTunnel(localConfig, remoteConfig) {
  // Create local server instance
  const localInstance = await createNodePassInstance({
    type: 'server',
    port: localConfig.port,
    target: localConfig.target
  });
  
  // Create remote client instance
  const remoteInstance = await createNodePassInstance({
    type: 'client',
    serverHost: localConfig.serverHost,
    port: remoteConfig.port,
    target: remoteConfig.target
  });
  
  if (localInstance.success && remoteInstance.success) {
    // Set peer information on local instance
    await updateCompleteMetadata(
      localInstance.data.id,
      {
        sid: remoteConfig.serviceId, // UUID format
        type: "2", // Tunnel forwarding
        alias: remoteConfig.serviceName
      },
      {
        tunnel_type: 'peer-to-peer',
        protocol: 'tcp',
        encryption: 'tls'
      }
    );
    
    // Set peer information on remote instance
    await updateCompleteMetadata(
      remoteInstance.data.id,
      {
        sid: localConfig.serviceId, // UUID format
        type: "2", // Tunnel forwarding
        alias: localConfig.serviceName
      },
      {
        tunnel_type: 'peer-to-peer',
        protocol: 'tcp',
        encryption: 'tls'
      }
    );
    
    console.log('Peer tunnel established with metadata');
  }
}

// Example 2: Organize instances by environment and region
async function organizeInstancesByEnvironment(instances) {
  for (const instance of instances) {
    const tags = {
      environment: instance.isProduction ? 'production' : 'development',
      region: instance.deploymentRegion,
      team: instance.owningTeam,
      cost_center: instance.costCenter,
      criticality: instance.isCritical ? 'high' : 'normal'
    };
    
    await updateInstanceTags(instance.id, tags);
    console.log(`Tagged instance ${instance.id} with environment metadata`);
  }
}

// Example 3: Query instances by metadata tags
async function findInstancesByTags(requiredTags) {
  const response = await fetch(`${API_URL}/instances`, {
    headers: { 'X-API-Key': apiKey }
  });
  const data = await response.json();
  
  if (data.success) {
    return data.data.filter(instance => {
      if (!instance.meta || !instance.meta.tags) return false;
      
      // Check if all required tags match
      return Object.entries(requiredTags).every(([key, value]) => 
        instance.meta.tags[key] === value
      );
    });
  }
  return [];
}

// Example 4: Update metadata based on operational status
async function updateMetadataOnStatusChange(instanceId, newStatus) {
  const instance = await fetch(`${API_URL}/instances/${instanceId}`, {
    headers: { 'X-API-Key': apiKey }
  });
  const data = await instance.json();
  
  if (data.success && data.data.meta) {
    const updatedTags = {
      ...data.data.meta.tags,
      last_status_change: new Date().toISOString(),
      current_status: newStatus,
      status_change_count: (parseInt(data.data.meta.tags.status_change_count || '0') + 1).toString()
    };
    
    await updateInstanceTags(instanceId, updatedTags);
  }
}
```

#### Metadata Best Practices

1. **Peer Information**: Use the `peer` object to track connections between instances
   - `sid`: Service unique identifier (required, UUID v4 format, e.g., `550e8400-e29b-41d4-a716-446655440000`)
     - Use standard UUID v4 format to ensure global uniqueness
     - Can use JavaScript's `crypto.randomUUID()` or third-party libraries to generate
   - `type`: Service type identifier (required, string enumeration value)
     - `"0"`: Single-end Forwarding - For simple client forwarding scenarios
     - `"1"`: NAT Traversal - For scenarios requiring NAT traversal
     - `"2"`: Tunnel Forwarding - For establishing encrypted tunnels
   - `alias`: Friendly name of the remote service (no format restriction, max 256 chars)

2. **Frontend Integration Standards**: To ensure consistency, frontends should follow these standards
   
   **Service ID (sid) Generation Standards:**
   ```javascript
   // Use browser native API to generate UUID v4
   const serviceId = crypto.randomUUID();
   // Example output: "550e8400-e29b-41d4-a716-446655440000"
   
   // Or use third-party library (e.g., uuid)
   import { v4 as uuidv4 } from 'uuid';
   const serviceId = uuidv4();
   ```
   
   **Service Type (type) Usage Standards:**
   ```javascript
   // Define service type enumeration
   const ServiceType = {
     SINGLE_END: "0",      // Single-end forwarding: client unidirectional forwarding, no server callback needed
     NAT_TRAVERSAL: "1",   // NAT traversal: traverse NAT for internal network access
     TUNNEL: "2"           // Tunnel forwarding: establish end-to-end encrypted tunnel
   };
   
   // Usage example
   const peerInfo = {
     sid: crypto.randomUUID(),
     type: ServiceType.NAT_TRAVERSAL,
     alias: "Web Server"
   };
   ```
   
   **Type Selection Guide:**
   - **Single-end Forwarding ("0")**:
     - Scenario: Client only needs to forward traffic to remote server
     - Feature: One-way connection, no server callback required
     - Example: Local app connecting to cloud database
   
   - **NAT Traversal ("1")**:
     - Scenario: Need to access internal network services from external network
     - Feature: Traverse NAT and firewall restrictions
     - Example: Remote access to home NAS, internal web services
   
   - **Tunnel Forwarding ("2")**:
     - Scenario: Need to establish secure end-to-end connection
     - Feature: Encrypted transmission, bidirectional communication
     - Example: Secure interconnection between branch offices and headquarters

3. **Tags Organization**: Design a consistent tagging strategy
   - Use lowercase keys with underscores (e.g., `cost_center`, `deployment_region`)
   - Limit tag values to meaningful, searchable strings
   - Common tag categories:
     - Environment: `production`, `staging`, `development`
     - Location: `us-west`, `eu-central`, `ap-southeast`
     - Ownership: `team-alpha`, `ops-team`, `platform-team`
     - Function: `database-tunnel`, `web-proxy`, `api-gateway`
     - Criticality: `high`, `medium`, `low`

4. **Field Length Limits**: All metadata fields have length requirements
   - `peer.sid`: Fixed 36 characters (UUID v4 format, e.g., `550e8400-e29b-41d4-a716-446655440000`)
   - `peer.type`: Fixed 1 character (enumeration value: `"0"` | `"1"` | `"2"`)
   - `peer.alias`: Max 256 chars (no specific format required)
   - Tag keys and values: Max 256 chars each

5. **Tag Uniqueness**: Ensure tag keys are unique within an instance
   - Duplicate keys will result in a 400 Bad Request error

6. **Filtering and Search**: Use metadata for instance filtering
   - Client-side filtering by tags for dashboard views
   - Query instances by peer information for relationship mapping
   - Group instances by tags for batch operations

#### Complete Auto-restart Policy Usage Example

Here's a comprehensive example showing how to implement auto-restart policy management in real scenarios:

```javascript
// Scenario: Establish load-balanced server cluster with auto-restart policies
async function setupServerCluster(serverConfigs) {
  const clusterInstances = [];
  
  for (const config of serverConfigs) {
    try {
      // Create server instance
      const instance = await createNodePassInstance({
        type: 'server',
        port: config.port,
        target: config.target,
        critical: config.isPrimary, // Primary servers are critical instances
        tls: config.enableTLS
      });
      
      if (instance.success) {
        // Set meaningful instance alias
        const alias = `${config.role}-server-${config.port}`;
        await updateInstanceAlias(instance.data.id, alias);
        
        // Configure auto-restart policy based on server role
        const autoStartPolicy = config.isPrimary || config.role === 'essential';
        await setAutoStartPolicy(instance.data.id, autoStartPolicy);
        
        // Start instance
        await controlInstance(instance.data.id, 'start');
        
        clusterInstances.push({
          id: instance.data.id,
          alias: alias,
          role: config.role,
          autoStartEnabled: autoStartPolicy
        });
        
        console.log(`Server ${alias} created, auto-restart policy: ${autoStartPolicy}`);
      }
    } catch (error) {
      console.error(`Failed to create server ${config.role}:`, error);
    }
  }
  
  return clusterInstances;
}

// Monitor cluster health and dynamically adjust auto-restart policies
async function monitorClusterHealth(clusterInstances) {
  const healthyInstances = [];
  
  for (const cluster of clusterInstances) {
    const instance = await fetch(`${API_URL}/instances/${cluster.id}`, {
      headers: { 'X-API-Key': apiKey }
    });
    const data = await instance.json();
    
    if (data.success && data.data.status === 'running') {
      healthyInstances.push(cluster);
    } else {
      // If critical instance is down, enable auto-restart for backup instances
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
    console.log(`Enabled auto-restart policy for backup instance: ${backup.id}`);
  }
}
```

### Traffic Statistics

The Master API provides traffic statistics, but there are important considerations:

1. **Basic Traffic Metrics**: NodePass periodically provides cumulative values for TCP and UDP traffic in both inbound and outbound directions. Frontend applications need to store and process these values to obtain meaningful statistics.
   ```javascript
   function processTrafficStats(instanceId, currentStats) {
     // Store current timestamp
     const timestamp = Date.now();
     
     // If we have previous stats for this instance, calculate differences
     if (previousStats[instanceId]) {
       const timeDiff = timestamp - previousStats[instanceId].timestamp;
       const tcpInDiff = currentStats.tcp_in - previousStats[instanceId].tcp_in;
       const tcpOutDiff = currentStats.tcp_out - previousStats[instanceId].tcp_out;
       const udpInDiff = currentStats.udp_in - previousStats[instanceId].udp_in;
       const udpOutDiff = currentStats.udp_out - previousStats[instanceId].udp_out;
       
       // Store historical data for chart display
       storeTrafficHistory(instanceId, {
         timestamp,
         tcp_in_rate: tcpInDiff / timeDiff * 1000, // bytes per second
         tcp_out_rate: tcpOutDiff / timeDiff * 1000,
         udp_in_rate: udpInDiff / timeDiff * 1000,
         udp_out_rate: udpOutDiff / timeDiff * 1000
       });
     }
     
     // Update previous stats for next calculation
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
   // Frontend traffic history storage structure example
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
     
     // Keep history data manageable
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

### Instance ID Persistence

Since NodePass now uses gob format for persistent storage of instance state, instance IDs **no longer change** after master restart. This means:

1. Frontend applications can safely use instance IDs as unique identifiers
2. Instance configurations, states, and statistics are automatically restored after restart
3. No longer need to implement logic for handling instance ID changes

This greatly simplifies frontend integration, eliminating the previous complexity of handling instance recreation and ID mapping.

### Auto-restart Policy Management

NodePass now supports configuring auto-restart policies for instances, enabling automated instance management and improving reliability. The auto-restart policy feature has the following characteristics:

1. **Automatic Instance Recovery**: Instances with auto-restart enabled automatically start when the master service restarts
2. **Selective Auto-start**: Configure which instances should auto-start based on their importance or role
3. **Persistent Policy Storage**: Auto-restart policies are saved and restored across master restarts
4. **Fine-grained Control**: Each instance can have its own auto-restart policy setting

#### How Auto-restart Policy Works

- **Policy Assignment**: Each instance has a `restart` boolean field that determines its auto-start behavior
- **Master Startup**: When master starts, it automatically starts all instances with `restart: true`
- **Policy Persistence**: Auto-restart policies are saved with other instance data in the `nodepass.gob` file
- **Runtime Management**: Auto-restart policies can be modified while instances are running

#### Auto-restart Policy Best Practices

1. **Enable for Server Instances**: Server instances should typically have auto-restart enabled for high availability
2. **Selective Client Auto-start**: Only enable auto-restart for critical client connections
3. **Testing Scenarios**: Disable auto-restart for temporary or testing instances
4. **Load Balancing**: Use auto-restart policies to maintain minimum instance count for load distribution

```javascript
// Example: Configure auto-restart policies based on instance role
async function configureAutoStartPolicies(instances) {
  for (const instance of instances) {
    // Enable auto-start for servers and critical clients
    const shouldAutoStart = instance.type === 'server' || 
                            instance.critical === true;
    
    await setAutoStartPolicy(instance.id, shouldAutoStart);
  }
}
```

## Instance Data Structure

The instance object in API responses contains the following fields:

```json
{
  "id": "a1b2c3d4",           // Instance unique identifier
  "alias": "web-server-01",   // Instance alias (optional, for friendly display name)
  "type": "server",           // Instance type: server or client
  "status": "running",        // Instance status: running, stopped, or error
  "url": "server://...",      // Instance configuration URL
  "config": "server://0.0.0.0:8080/localhost:3000?log=info&tls=1&max=1024&mode=0&read=1h&rate=0&slot=65536&proxy=0", // Complete configuration URL
  "restart": true,            // Auto-restart policy
  "meta": {                   // Metadata for organization and peer tracking
    "peer": {
      "sid": "550e8400-e29b-41d4-a716-446655440000",  // Remote service ID (UUID format)
      "type": "1",                // Remote service type (0=Single-end, 1=NAT Traversal, 2=Tunnel)
      "alias": "remote-service"   // Remote service friendly name
    },
    "tags": {                 // Custom key-value tags
      "environment": "production",
      "region": "us-west",
      "team": "platform"
    }
  },
  "mode": 0,                  // Instance mode
  "tcprx": 1024,              // TCP received bytes
  "tcptx": 2048,              // TCP transmitted bytes
  "udprx": 512,               // UDP received bytes
  "udptx": 256                // UDP transmitted bytes
}
```

**Note:** 
- `alias` field is optional, empty string if not set
- `config` field contains the instance's complete configuration URL, auto-generated by the system
- `mode` field indicates the current runtime mode of the instance
- `restart` field controls the auto-restart behavior of the instance
- `meta` field contains structured metadata for instance organization
  - `peer` object tracks remote endpoint information for peer-to-peer connections
    - `sid`: Service unique identifier, must use UUID v4 format (36 chars, e.g., `550e8400-e29b-41d4-a716-446655440000`)
    - `type`: Service type identifier, string enumeration value (`"0"` | `"1"` | `"2"`)
      - `"0"`: Single-end Forwarding - Client unidirectional forwarding
      - `"1"`: NAT Traversal - Traverse NAT for internal network access
      - `"2"`: Tunnel Forwarding - Establish end-to-end encrypted tunnel
    - `alias`: Custom string, max 256 chars, no format restriction
  - `tags` map allows flexible categorization with custom key-value pairs
  - Tag keys and values have a 256-character maximum length
  - Tag keys must be unique within an instance

### Instance Configuration Field

NodePass Master automatically maintains the `config` field for each instance:

- **Auto Generation**: Automatically generated when instances are created and updated, no manual maintenance required
- **Complete Configuration**: Contains the instance's complete URL with all default parameters
- **Configuration Inheritance**: log and tls configurations are inherited from master settings
- **Default Parameters**: Other parameters use system defaults
- **Read-Only Nature**: Auto-generated field that cannot be directly modified through the API

**Example config field value:**
```
server://0.0.0.0:8080/localhost:3000?log=info&tls=1&max=1024&mode=0&read=1h&rate=0&slot=65536&proxy=0
```

This feature is particularly useful for:
- Configuration backup and export
- Instance configuration integrity checks
- Automated deployment scripts
- Configuration documentation generation

## System Information Endpoint

The `/info` endpoint provides system information about the NodePass master service. This endpoint is useful for monitoring, troubleshooting, and system status validation.

### Request

```
GET /info
```

Requires API Key authentication: Yes

### Response

The response contains the following system information fields:

```json
{
  "alias": "dev",             // Master alias
  "os": "linux",              // Operating system type
  "arch": "amd64",            // System architecture
  "cpu": 45,                  // CPU usage percentage (Linux only)
  "mem_total": 8589934592,    // Total memory in bytes (Linux only)
  "mem_used": 2684354560,     // Used memory in bytes (Linux only)
  "swap_total": 3555328000,   // Total swap space in bytes (Linux only)
  "swap_used": 3555328000,    // Used swap space in bytes (Linux only)
  "netrx": 1048576000,        // Network received bytes (cumulative, Linux only)
  "nettx": 2097152000,        // Network transmitted bytes (cumulative, Linux only)
  "diskr": 4194304000,        // Disk read bytes (cumulative, Linux only)
  "diskw": 8388608000,        // Disk write bytes (cumulative, Linux only)
  "sysup": 86400,             // System uptime in seconds (Linux only)
  "ver": "1.2.0",             // NodePass version
  "name": "example.com",      // Tunnel hostname
  "uptime": 11525,            // API uptime in seconds
  "log": "info",              // Log level
  "tls": "1",                 // TLS enabled status
  "crt": "/path/to/cert",     // Certificate path
  "key": "/path/to/key"       // Key path
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

// Display service uptime and system resource usage
function displaySystemStatus() {
  getSystemInfo().then(info => {
    console.log(`Service uptime: ${info.uptime} seconds`);
    
    // Format uptime for friendlier display
    const hours = Math.floor(info.uptime / 3600);
    const minutes = Math.floor((info.uptime % 3600) / 60);
    const seconds = info.uptime % 60;
    console.log(`Service uptime: ${hours}h ${minutes}m ${seconds}s`);
    
    // Display system resource usage (Linux only)
    if (info.os === 'linux') {
      if (info.cpu !== -1) {
        console.log(`CPU usage: ${info.cpu}%`);
      }
      if (info.mem_total > 0) {
        const memUsagePercent = (info.mem_used / info.mem_total * 100).toFixed(1);
        const memFreeGB = ((info.mem_total - info.mem_used) / 1024 / 1024 / 1024).toFixed(1);
        const memTotalGB = (info.mem_total / 1024 / 1024 / 1024).toFixed(1);
        console.log(`Memory usage: ${memUsagePercent}% (${memFreeGB}GB free of ${memTotalGB}GB total)`);
      }
      if (info.swap_total > 0) {
        const swapUsagePercent = (info.swap_used / info.swap_total * 100).toFixed(1);
        const swapFreeGB = ((info.swap_total - info.swap_used) / 1024 / 1024 / 1024).toFixed(1);
        const swapTotalGB = (info.swap_total / 1024 / 1024 / 1024).toFixed(1);
        console.log(`Swap usage: ${swapUsagePercent}% (${swapFreeGB}GB free of ${swapTotalGB}GB total)`);
      }
    } else {
      console.log('CPU, memory, swap space, network I/O, disk I/O, and system uptime monitoring is only available on Linux systems');
    }
    
    // Display network I/O statistics (cumulative values)
    if (info.os === 'linux') {
      console.log(`Network received: ${(info.netrx / 1024 / 1024).toFixed(2)} MB (cumulative)`);
      console.log(`Network transmitted: ${(info.nettx / 1024 / 1024).toFixed(2)} MB (cumulative)`);
      console.log(`Disk read: ${(info.diskr / 1024 / 1024).toFixed(2)} MB (cumulative)`);
      console.log(`Disk write: ${(info.diskw / 1024 / 1024).toFixed(2)} MB (cumulative)`);
      console.log(`System uptime: ${Math.floor(info.sysup / 3600)} hours`);
    }
  });
}
```

### Monitoring Best Practices

- **Regular checks**: Poll this endpoint regularly to ensure service is running properly
- **Version verification**: Check version number after deploying updates
- **Uptime monitoring**: Monitor uptime to detect unexpected restarts
- **Log level verification**: Ensure current log level meets expectations
- **Resource monitoring**: On Linux systems, monitor CPU, memory, swap space, network I/O, disk I/O usage to ensure optimal performance
  - CPU usage is calculated by parsing `/proc/stat` (percentage of non-idle time)
  - Memory information is obtained by parsing `/proc/meminfo` (total and used memory in bytes, calculated as total minus available)
  - Swap space information is obtained by parsing `/proc/meminfo` (total and used swap space in bytes, calculated as total minus free)
  - Network I/O is calculated by parsing `/proc/net/dev` (cumulative bytes, excluding virtual interfaces)
  - Disk I/O is calculated by parsing `/proc/diskstats` (cumulative bytes, major devices only)
  - System uptime is obtained by parsing `/proc/uptime`
  - Values of -1 or 0 indicate system information is unavailable (non-Linux systems)
  - Network and disk I/O fields provide cumulative values; frontend applications need to store historical data and calculate differences to get real-time rates (bytes/second)

## API Endpoint Documentation

For detailed API documentation including request and response examples, use the built-in Swagger UI documentation available at the `/docs` endpoint. This interactive documentation provides comprehensive information about:

- Available endpoints
- Required parameters
- Response formats
- Request and response examples
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

The Swagger UI provides a convenient way to explore and test the API directly in your browser. You can execute API calls against your running NodePass master instance and see actual responses.

## Complete API Reference

### Instance Management Endpoints Detail

#### GET /instances
- **Description**: Get list of all instances
- **Authentication**: Requires API Key
- **Response**: Array of instances
- **Example**:
```javascript
const instances = await fetch(`${API_URL}/instances`, {
  headers: { 'X-API-Key': apiKey }
});
```

#### POST /instances
- **Description**: Create new instance
- **Authentication**: Requires API Key
- **Request body**: `{ "url": "client:// or server:// format URL" }`
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
- **Authentication**: Requires API Key
- **Response**: Instance object
- **Example**:
```javascript
const instance = await fetch(`${API_URL}/instances/abc123`, {
  headers: { 'X-API-Key': apiKey }
});
```

#### PATCH /instances/{id}
- **Description**: Update instance state, alias, metadata, or perform control operations
- **Authentication**: Requires API Key
- **Request body**: `{ "alias": "new alias", "action": "start|stop|restart|reset", "restart": true|false, "meta": {...} }`
- **Metadata Structure**:
  - `peer`: Object with fields (all optional):
    - `sid`: Service ID (UUID v4 format, 36 chars, e.g., `550e8400-e29b-41d4-a716-446655440000`)
    - `type`: Service type (enumeration value: `"0"` | `"1"` | `"2"`)
      - `"0"`: Single-end Forwarding
      - `"1"`: NAT Traversal
      - `"2"`: Tunnel Forwarding
    - `alias`: Service alias (max 256 chars, no format restriction)
  - `tags`: Object with custom key-value pairs (keys and values max 256 chars, keys must be unique)
- **Example**:
```javascript
// Update alias and restart policy
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    alias: "production-server",
    restart: true
  })
});

// Perform control action
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    action: "restart"
  })
});

// Update metadata with peer information and tags
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    meta: {
      peer: {
        sid: "550e8400-e29b-41d4-a716-446655440000", // UUID format
        type: "1", // NAT Traversal
        alias: "remote-api-server"
      },
      tags: {
        environment: "production",
        region: "us-east",
        team: "backend",
        criticality: "high"
      }
    }
  })
});

// Update only tags (peer info remains unchanged)
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    meta: {
      peer: {},  // Empty object preserves existing peer info
      tags: {
        environment: "staging",
        updated_at: new Date().toISOString()
      }
    }
  })
});
```

#### PUT /instances/{id}
- **Description**: Completely update instance URL configuration
- **Authentication**: Requires API Key
- **Request body**: `{ "url": "new client:// or server:// format URL" }`
- **Features**: Will restart the instance.
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
- **Authentication**: Requires API Key
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
- **Authentication**: Requires API Key
- **Response**: Server-Sent Events stream
- **Event types**: `initial`, `create`, `update`, `delete`, `shutdown`, `log`

#### GET /info
- **Description**: Get master service information
- **Authentication**: Requires API Key
- **Response**: Contains system information, version, uptime, CPU and RAM usage, etc.

#### POST /info
- **Description**: Update master alias
- **Authentication**: Requires API Key
- **Request body**: `{ "alias": "new alias" }`
- **Response**: Complete master information (same as GET /info)
- **Note**: Master alias is stored in the `alias` field of the API Key instance (ID `********`)
- **Example**:
```javascript
// Update master alias
const response = await fetch(`${API_URL}/info`, {
  method: 'POST',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ alias: 'My NodePass Server' })
});

const data = await response.json();
console.log('Updated alias:', data.alias);
// Response contains full system info with updated alias
```

**Retrieving Master ID**: The Master ID is stored in the `config` field of the API Key instance and can be retrieved as follows:
```javascript
// Get Master ID
async function getMasterID() {
  const response = await fetch(`${API_URL}/instances/********`, {
    headers: { 'X-API-Key': apiKey }
  });
  const data = await response.json();
  return data.data.config; // Returns 16-character hex Master ID
}
```

#### GET /tcping
- **Description**: TCP connection test, checks connectivity and latency to target address
- **Authentication**: Requires API Key
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
- **Response**: JSON format API specification

#### GET /docs
- **Description**: Swagger UI documentation interface
- **Authentication**: No authentication required
- **Response**: HTML format interactive documentation

### Instance URL Format Specification

Instance URLs must follow these formats:

#### Server Mode
```
server://<bind_address>:<bind_port>/<target_host>:<target_port>?<parameters>
```

Examples:
- `server://0.0.0.0:8080/localhost:3000` - Listen on port 8080, forward to local port 3000
- `server://0.0.0.0:9090/localhost:8080?tls=1&mode=1` - TLS-enabled server, force reverse mode

#### Client Mode
```
client://<server_host>:<server_port>/<local_host>:<local_port>?<parameters>
```

Examples:
- `client://example.com:8080/localhost:3000` - Connect to remote server, listen locally on port 3000
- `client://remote.example.com:443/localhost:22?mode=2&min=32` - Through remote server, force dual-end mode

#### Supported Parameters

| Parameter | Description | Values | Default | Scope |
|-----------|-------------|--------|---------|-------|
| `log` | Log level | `none`, `debug`, `info`, `warn`, `error`, `event` | `info` | Both |
| `tls` | TLS encryption level | `0`(none), `1`(self-signed), `2`(certificate) | `0` | Server only |
| `crt` | Certificate path | File path | None | Server only |
| `key` | Private key path | File path | None | Server only |
| `min` | Minimum pool capacity | Integer > 0 | `64` | Client dual-end handshake mode only |
| `max` | Maximum pool capacity | Integer > 0 | `1024` | Dual-end handshake mode |
| `mode` | Runtime mode control | `0`(auto), `1`(force mode 1), `2`(force mode 2) | `0` | Both |
| `quic` | QUIC protocol support | `0`(disabled), `1`(enabled) | `0` | Both |
| `read` | Read timeout duration | Time duration (e.g., `10m`, `30s`, `1h`) | `0` | Both |
| `rate` | Bandwidth rate limit | Integer (Mbps), 0=unlimited | `0` | Both |
| `slot` | Connection slot count | Integer (1-65536) | `65536` | Both |
| `proxy` | PROXY protocol support | `0`(disabled), `1`(enabled) | `0` | Both |
| `notcp` | TCP support control | `0`(enabled), `1`(disabled) | `0` | Both |
| `noudp` | UDP support control | `0`(enabled), `1`(disabled) | `0` | Both |