# NodePass API参考

## 概述

NodePass在主控模式（Master Mode）下提供了RESTful API，使前端应用能够以编程方式进行控制和集成。本节提供API端点、集成模式和最佳实践的全面文档。

## 主控模式API

当NodePass以主控模式（`master://`）运行时，它会暴露REST API，允许前端应用：

1. 创建和管理NodePass服务器和客户端实例
2. 监控连接状态和统计信息
3. 控制运行中的实例（启动、停止、重启）
4. 配置实例自启动策略以实现自动化管理
5. 通过参数配置行为

### 基本URL

```
master://<api_addr>/<prefix>?<log>&<tls>
```

其中：
- `<api_addr>`是主控模式URL中指定的地址（例如`0.0.0.0:9090`）
- `<prefix>`是可选的API前缀（如果未指定，则使用`/api`）

### 启动主控模式

使用默认设置启动主控模式的NodePass：

```bash
nodepass "master://0.0.0.0:9090?log=info"
```

使用自定义API前缀和启用TLS：

```bash
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"
```

### 可用端点

| 端点 | 方法 | 描述 |
|----------|--------|-------------|
| `/instances` | GET | 列出所有NodePass实例 |
| `/instances` | POST | 创建新的NodePass实例 |
| `/instances/{id}` | GET | 获取特定实例的详细信息 |
| `/instances/{id}` | PATCH | 更新实例状态或控制操作 |
| `/instances/{id}` | PUT | 更新实例URL配置 |
| `/instances/{id}` | DELETE | 删除特定实例 |
| `/events` | GET | 使用SSE订阅实例事件通知 |
| `/info` | GET | 获取主控服务信息 |
| `/openapi.json` | GET | OpenAPI规范 |
| `/docs` | GET | Swagger UI文档界面 |

### API认证

NodePass主控API现在支持API Key认证，可以防止未经授权的访问。系统会在首次启动时自动生成一个API Key。

#### API Key特点

1. **自动生成**：首次启动主控模式时自动创建
2. **持久化存储**：API Key与其他实例配置一起保存在`nodepass.gob`文件中
3. **重启后保留**：重启主控后API Key保持不变
4. **选择性保护**：仅保护关键API端点，公共文档仍可访问

#### 受保护的端点

以下端点需要API Key认证：
- `/instances`（所有方法）
- `/instances/{id}`（所有方法：GET、PATCH、PUT、DELETE）
- `/events`
- `/info`

以下端点可公开访问（无需API Key）：
- `/openapi.json`
- `/docs`

#### 如何使用API Key

在API请求中包含API Key：

```javascript
// 使用API Key进行实例管理请求
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

#### 如何获取和重新生成API Key

API Key可以在系统启动日志中找到，也可以通过以下方式重新生成：

```javascript
// 重新生成API Key（需要知道当前的API Key）
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
  return result.url; // 新的API Key
}
```

**注意**: API Key ID 固定为 `********`（八个星号）。在内部实现中，这是一个特殊的实例ID，用于存储和管理API Key。

### 使用SSE实时事件监控

NodePass现在支持服务器发送事件(SSE)功能，用于实时监控实例状态变化。这使前端应用能够即时接收实例创建、更新和删除的通知，无需轮询。

#### 使用SSE端点

SSE端点位于：
```
GET /events
```

此端点建立持久连接，使用SSE协议格式实时传递事件。如果启用了API Key认证，需要在请求头中包含有效的API Key。

#### 事件类型

支持以下事件类型：

1. `initial` - 连接建立时发送，包含所有实例的当前状态
2. `create` - 创建新实例时发送
3. `update` - 实例更新时发送（状态变更、启动/停止操作）
4. `delete` - 实例被删除时发送
5. `shutdown` - 主控服务即将关闭时发送，通知前端应用关闭连接
6. `log` - 实例产生新日志内容时发送，包含日志文本

#### JavaScript客户端实现

以下是JavaScript前端消费SSE端点的示例：

```javascript
function connectToEventSource() {
  const eventSource = new EventSource(`${API_URL}/events`, {
    // 如果需要认证，原生EventSource不支持自定义请求头
    // 需要使用fetch API实现自定义SSE客户端
  });
  
  // 如果使用API Key，需要使用自定义实现代替原生EventSource
  // 下面是使用原生EventSource的示例
  eventSource.addEventListener('instance', (event) => {
    const data = JSON.parse(event.data);
    
    switch (data.type) {
      case 'initial':
        console.log('初始实例状态:', data.instance);
        updateInstanceUI(data.instance);
        break;
      case 'create':
        console.log('实例已创建:', data.instance);
        addInstanceToUI(data.instance);
        break;
      case 'update':
        console.log('实例已更新:', data.instance);
        updateInstanceUI(data.instance);
        break;
      case 'delete':
        console.log('实例已删除:', data.instance);
        removeInstanceFromUI(data.instance.id);
        break;
      case 'log':
        console.log(`实例 ${data.instance.id} 日志:`, data.logs);
        appendLogToInstanceUI(data.instance.id, data.logs);
        break;
      case 'shutdown':
        console.log('主控服务即将关闭');
        // 关闭事件源并显示通知
        eventSource.close();
        showShutdownNotification();
        break;
    }
  });
  
  eventSource.addEventListener('error', (error) => {
    console.error('SSE连接错误:', error);
    // 延迟后尝试重新连接
    setTimeout(() => {
      eventSource.close();
      connectToEventSource();
    }, 5000);
  });
  
  return eventSource;
}

// 使用API Key创建SSE连接的示例
function connectToEventSourceWithApiKey(apiKey) {
  // 原生EventSource不支持自定义请求头，需要使用fetch API
  fetch(`${API_URL}/events`, {
    method: 'GET',
    headers: {
      'X-API-Key': apiKey,
      'Cache-Control': 'no-cache'
    }
  }).then(response => {
    if (!response.ok) {
      throw new Error(`HTTP错误: ${response.status}`);
    }
    
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    
    function processStream() {
      reader.read().then(({ value, done }) => {
        if (done) {
          console.log('连接已关闭');
          // 尝试重新连接
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
            // 处理事件 - 见上面的switch代码
          }
        }
        
        processStream();
      }).catch(error => {
        console.error('读取错误:', error);
        // 尝试重新连接
        setTimeout(() => connectToEventSourceWithApiKey(apiKey), 5000);
      });
    }
    
    processStream();
  }).catch(error => {
    console.error('连接错误:', error);
    // 尝试重新连接
    setTimeout(() => connectToEventSourceWithApiKey(apiKey), 5000);
  });
}
```

#### 处理实例日志

新增的`log`事件类型允许实时接收和显示实例的日志输出。这对于监控和调试非常有用：

```javascript
// 处理日志事件
function appendLogToInstanceUI(instanceId, logText) {
  // 找到或创建日志容器
  let logContainer = document.getElementById(`logs-${instanceId}`);
  if (!logContainer) {
    logContainer = document.createElement('div');
    logContainer.id = `logs-${instanceId}`;
    document.getElementById('instance-container').appendChild(logContainer);
  }
  
  // 创建新的日志条目
  const logEntry = document.createElement('div');
  logEntry.className = 'log-entry';
  
  // 可以在这里解析ANSI颜色代码或格式化日志
  logEntry.textContent = logText;
  
  // 添加到容器
  logContainer.appendChild(logEntry);
  
  // 滚动到最新日志
  logContainer.scrollTop = logContainer.scrollHeight;
}
```

#### SSE相比轮询的优势

使用SSE监控实例状态比传统轮询提供多种优势：

1. **减少延迟**：变更实时传递
2. **减轻服务器负载**：消除不必要的轮询请求
3. **带宽效率**：只在发生变更时发送数据
4. **原生浏览器支持**：无需额外库的内置浏览器支持
5. **自动重连**：浏览器在连接丢失时自动重连

#### SSE实现的最佳实践

在前端实现SSE时：

1. **处理重连**：虽然浏览器会自动尝试重连，但应实现自定义逻辑以确保持久连接
2. **高效处理事件**：保持事件处理快速，避免UI阻塞
3. **实现回退机制**：在不支持SSE的环境中，实现轮询回退
4. **处理错误**：正确处理连接错误和断开
5. **日志管理**：为每个实例维护日志缓冲区，避免无限制增长

## 前端集成指南

在将NodePass与前端应用集成时，请考虑以下重要事项：

### 实例持久化

NodePass主控模式现在支持使用gob序列化格式进行实例持久化。实例及其状态会保存到与可执行文件相同目录下的`nodepass.gob`文件中，并在主控重启时自动恢复。

主要持久化特性：
- 实例配置自动保存到磁盘
- 实例状态（运行/停止）得到保留
- 自启动策略在主控重启间保持不变
- 流量统计数据在重启之间保持
- 启用自启动策略的实例在主控重启时自动启动
- 重启后无需手动重新注册

**注意：** 虽然实例配置现在已经持久化，前端应用仍应保留自己的实例配置记录作为备份策略。

### 实例生命周期管理

为了合理管理生命周期：

1. **创建**：存储实例配置和URL
   ```javascript
   async function createNodePassInstance(config) {
     const response = await fetch(`${API_URL}/instances`, {
       method: 'POST',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // 如果启用了API Key
       },
       body: JSON.stringify({
         url: `server://0.0.0.0:${config.port}/${config.target}?tls=${config.tls}`
       })
     });
     
     const data = await response.json();
     
     // 根据类型为新实例配置自启动策略
     if (data.success) {
       const shouldAutoRestart = config.type === 'server' || config.critical === true;
       await setAutoStartPolicy(data.data.id, shouldAutoRestart);
     }
     
     // 存储在前端持久化存储中
     saveInstanceConfig({
       id: data.data.id,
       originalConfig: config,
       url: data.data.url
     });
     
     return data;
   }
   ```

2. **状态监控**：监控实例状态变化
   
   NodePass提供两种监控实例状态的方法：
   
   A. **使用SSE（推荐）**：通过持久连接接收实时事件
   ```javascript
   function connectToEventSource() {
     const eventSource = new EventSource(`${API_URL}/events`, {
       // 如果需要认证，需要使用自定义实现
     });
     
     // 或者使用带API Key的自定义实现
     // connectToEventSourceWithApiKey(apiKey);
     
     eventSource.addEventListener('instance', (event) => {
       const data = JSON.parse(event.data);
       // 处理不同类型的事件：initial, create, update, delete, log
       // ...处理逻辑见前面的"使用SSE实时事件监控"部分
     });
     
     // 错误处理和重连逻辑
     // ...详见前面的示例
     
     return eventSource;
   }
   ```
   
   B. **传统轮询（备选）**：在不支持SSE的环境中使用
   ```javascript
   function startInstanceMonitoring(instanceId, interval = 5000) {
     return setInterval(async () => {
       try {
         const response = await fetch(`${API_URL}/instances/${instanceId}`, {
           headers: {
             'X-API-Key': apiKey // 如果启用了API Key
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

   **选择建议：** 优先使用SSE方式，它提供更高效的实时监控，减轻服务器负担。仅在客户端不支持SSE或需要特定环境兼容性时使用轮询方式。

3. **实例别名管理**：为实例设置易读的名称
   ```javascript
   // 批量设置实例别名
   async function setInstanceAliases(instances) {
     for (const instance of instances) {
       // 根据实例类型和用途生成有意义的别名
       const alias = `${instance.type}-${instance.region || 'default'}-${instance.port || 'auto'}`;
       await updateInstanceAlias(instance.id, alias);
     }
   }
   
   // 根据别名查找实例
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

4. **控制操作**：启动、停止、重启实例
   ```javascript
   async function controlInstance(instanceId, action) {
     // action可以是: start, stop, restart
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',  // 注意：API已更新为使用PATCH方法而非PUT
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // 如果启用了API Key 
       },
       body: JSON.stringify({ action })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // 更新实例别名
   async function updateInstanceAlias(instanceId, alias) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // 如果启用了API Key 
       },
       body: JSON.stringify({ alias })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // 更新实例URL配置
   async function updateInstanceURL(instanceId, newURL) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PUT',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // 如果启用了API Key 
       },
       body: JSON.stringify({ url: newURL })
     });
     
     const data = await response.json();
     return data.success;
   }
   ```

5. **自启动策略管理**：配置自动启动行为
   ```javascript
   async function setAutoStartPolicy(instanceId, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // 如果启用了API Key
       },
       body: JSON.stringify({ restart: enableAutoStart })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // 组合操作：控制实例并更新自启动策略
   async function controlInstanceWithAutoStart(instanceId, action, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // 如果启用了API Key
       },
       body: JSON.stringify({ 
         action: action,
         restart: enableAutoStart 
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // 组合操作：同时更新别名、控制实例和自启动策略
   async function updateInstanceComplete(instanceId, alias, action, enableAutoStart) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // 如果启用了API Key
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

#### 自启动策略完整使用示例

以下是一个全面的示例，展示了如何在实际场景中实现自启动策略管理：

```javascript
// 场景：建立带有自启动策略的负载均衡服务器集群
async function setupServerCluster(serverConfigs) {
  const clusterInstances = [];
  
  for (const config of serverConfigs) {
    try {
      // 创建服务器实例
      const instance = await createNodePassInstance({
        type: 'server',
        port: config.port,
        target: config.target,
        critical: config.isPrimary, // 主服务器为关键实例
        tls: config.enableTLS
      });
      
      if (instance.success) {
        // 设置有意义的实例别名
        const alias = `${config.role}-server-${config.port}`;
        await updateInstanceAlias(instance.data.id, alias);
        
        // 根据服务器角色配置自启动策略
        const autoStartPolicy = config.isPrimary || config.role === 'essential';
        await setAutoStartPolicy(instance.data.id, autoStartPolicy);
        
        // 启动实例
        await controlInstance(instance.data.id, 'start');
        
        clusterInstances.push({
          id: instance.data.id,
          alias: alias,
          role: config.role,
          autoStartEnabled: autoStartPolicy
        });
        
        console.log(`服务器 ${alias} 已创建，自启动策略: ${autoStartPolicy}`);
      }
    } catch (error) {
      console.error(`创建服务器 ${config.role} 失败:`, error);
    }
  }
  
  return clusterInstances;
}

// 监控集群健康状态并动态调整自启动策略
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
      // 如果关键实例宕机，为备份实例启用自启动策略
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
    console.log(`已为备份实例启用自启动策略: ${backup.id}`);
  }
}
```

### 流量统计

主控API提供流量统计数据，但需要注意以下重要事项：

1. **启用调试模式**：流量统计功能仅在启用调试模式时可用。

   ```bash
   # 启用调试模式的主控
   nodepass master://0.0.0.0:10101?log=debug
   ```

   如果未启用调试模式，API将不会收集或返回流量统计数据。

2. **基本流量指标**：NodePass周期性地提供TCP和UDP流量在入站和出站方向上的累计值，前端应用需要存储和处理这些值以获得有意义的统计信息。
   ```javascript
   function processTrafficStats(instanceId, currentStats) {
     // 存储当前时间戳
     const timestamp = Date.now();
     
     // 如果我们有该实例的前一个统计数据，计算差值
     if (previousStats[instanceId]) {
       const timeDiff = timestamp - previousStats[instanceId].timestamp;
       const tcpInDiff = currentStats.tcp_in - previousStats[instanceId].tcp_in;
       const tcpOutDiff = currentStats.tcp_out - previousStats[instanceId].tcp_out;
       const udpInDiff = currentStats.udp_in - previousStats[instanceId].udp_in;
       const udpOutDiff = currentStats.udp_out - previousStats[instanceId].udp_out;
       
       // 存储历史数据用于图表展示
       storeTrafficHistory(instanceId, {
         timestamp,
         tcp_in_rate: tcpInDiff / timeDiff * 1000, // 每秒字节数
         tcp_out_rate: tcpOutDiff / timeDiff * 1000,
         udp_in_rate: udpInDiff / timeDiff * 1000,
         udp_out_rate: udpOutDiff / timeDiff * 1000
       });
     }
     
     // 更新前一个统计数据，用于下次计算
     previousStats[instanceId] = {
       timestamp,
       tcp_in: currentStats.tcp_in,
       tcp_out: currentStats.tcp_out,
       udp_in: currentStats.udp_in,
       udp_out: currentStats.udp_out
     };
   }
   ```

3. **数据持久化**：由于API只提供累计值，前端必须实现适当的存储和计算逻辑
   ```javascript
   // 前端流量历史存储结构示例
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
     
     // 保持历史数据量可管理
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

### 实例ID持久化

由于NodePass现在使用gob格式持久化存储实例状态，实例ID在主控重启后**不再发生变化**。这意味着：

1. 前端应用可以安全地使用实例ID作为唯一标识符
2. 实例配置、状态和统计数据在重启后自动恢复
3. 不再需要实现实例ID变化的处理逻辑

这极大简化了前端集成，消除了以前处理实例重新创建和ID映射的复杂性。

### 自启动策略管理

NodePass现在支持为实例配置自启动策略，实现自动化实例管理并提高可靠性。自启动策略功能具备以下特性：

1. **自动实例恢复**：启用自启动策略的实例在主控服务重启时会自动启动
2. **选择性自启动**：根据实例的重要性或角色配置哪些实例应该自动启动
3. **持久化策略存储**：自启动策略在主控重启间保存和恢复
4. **细粒度控制**：每个实例都可以有自己的自启动策略设置

#### 自启动策略工作原理

- **策略分配**：每个实例都有一个`restart`布尔字段，决定其自启动行为
- **主控启动**：主控启动时，自动启动所有`restart: true`的实例
- **策略持久化**：自启动策略与其他实例数据一起保存在`nodepass.gob`文件中
- **运行时管理**：自启动策略可以在实例运行时修改

#### 自启动策略最佳实践

1. **为服务器实例启用**：服务器实例通常应启用自启动策略以确保高可用性
2. **选择性客户端自启动**：仅为关键客户端连接启用自启动策略
3. **测试场景**：为临时或测试实例禁用自启动策略
4. **负载均衡**：使用自启动策略维持最小实例数量以分配负载

```javascript
// 示例：根据实例角色配置自启动策略
async function configureAutoStartPolicies(instances) {
  for (const instance of instances) {
    // 为服务器和关键客户端启用自启动
    const shouldAutoStart = instance.type === 'server' || 
                            instance.tags?.includes('critical');
    
    await setAutoStartPolicy(instance.id, shouldAutoStart);
  }
}
```

## 实例数据结构

API响应中的实例对象包含以下字段：

```json
{
  "id": "a1b2c3d4",           // 实例唯一标识符
  "alias": "web-server-01",   // 实例别名（可选，用于显示友好名称）
  "type": "server",           // 实例类型：server 或 client
  "status": "running",        // 实例状态：running、stopped 或 error
  "url": "server://...",      // 实例配置URL
  "restart": true,            // 自启动策略
  "tcprx": 1024,             // TCP接收字节数
  "tcptx": 2048,             // TCP发送字节数
  "udprx": 512,              // UDP接收字节数
  "udptx": 256               // UDP发送字节数
}
```

**注意：** 
- `alias` 字段为可选，如果未设置则为空字符串
- 流量统计字段（tcprx、tcptx、udprx、udptx）仅在启用调试模式时有效
- `restart` 字段控制实例的自启动行为

## 系统信息端点

`/info` 端点提供了关于NodePass主控服务的系统信息。这个端点对于监控、故障排除和系统状态验证非常有用。

### 请求

```
GET /info
```

需要 API Key 认证：是

### 响应

响应包含以下系统信息字段：

```json
{
  "os": "linux",          // 操作系统类型
  "arch": "amd64",        // 系统架构
  "ver": "1.2.0",         // NodePass版本
  "name": "example.com",  // 隧道主机名
  "uptime": 11525,         // API运行时间（秒）
  "log": "info",          // 日志级别
  "tls": "1",             // TLS启用状态
  "crt": "/path/to/cert", // 证书路径
  "key": "/path/to/key"   // 密钥路径
}
```

### 使用示例

```javascript
// 获取系统信息
async function getSystemInfo() {
  const response = await fetch(`${API_URL}/info`, {
    method: 'GET',
    headers: {
      'X-API-Key': apiKey
    }
  });
  
  return await response.json();
}

// 显示服务运行时间
function displayServiceUptime() {
  getSystemInfo().then(info => {
    console.log(`服务已运行: ${info.uptime} 秒`);
    // 也可以格式化为更友好的显示
    const hours = Math.floor(info.uptime / 3600);
    const minutes = Math.floor((info.uptime % 3600) / 60);
    const seconds = info.uptime % 60;
    console.log(`服务已运行: ${hours}小时${minutes}分${seconds}秒`);
  });
}
```

### 监控最佳实践

- **定期检查**：定期轮询此端点以确保服务正常运行
- **版本验证**：在部署更新后检查版本号
- **运行时间监控**：监控运行时间以检测意外重启
- **日志级别验证**：确认当前日志级别符合预期

## API端点文档

有关详细的API文档（包括请求和响应示例），请使用`/docs`端点提供的内置Swagger UI文档。这个交互式文档提供了以下全面信息：

- 可用的端点
- 必需的参数
- 响应格式
- 请求和响应示例
- 架构定义

### 访问Swagger UI

要访问Swagger UI文档：

```
http(s)://<api_addr>[<prefix>]/docs
```

例如：
```
http://localhost:9090/api/docs
```

Swagger UI提供了一种方便的方式，直接在浏览器中探索和测试API。您可以针对运行中的NodePass主控实例执行API调用，并查看实际响应。

## 完整的API参考

### 实例管理端点详细说明

#### GET /instances
- **描述**：获取所有实例列表
- **认证**：需要API Key
- **响应**：实例数组
- **示例**：
```javascript
const instances = await fetch(`${API_URL}/instances`, {
  headers: { 'X-API-Key': apiKey }
});
```

#### POST /instances
- **描述**：创建新实例
- **认证**：需要API Key
- **请求体**：`{ "url": "client://或server://格式的URL" }`
- **响应**：新创建的实例对象
- **示例**：
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
- **描述**：获取特定实例详情
- **认证**：需要API Key
- **响应**：实例对象
- **示例**：
```javascript
const instance = await fetch(`${API_URL}/instances/abc123`, {
  headers: { 'X-API-Key': apiKey }
});
```

#### PATCH /instances/{id}
- **描述**：更新实例状态、别名或执行控制操作
- **认证**：需要API Key
- **请求体**：`{ "alias": "新别名", "action": "start|stop|restart", "restart": true|false }`
- **特点**：不中断正在运行的实例，仅更新指定字段
- **示例**：
```javascript
// 更新别名和自启动策略
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    alias: 'Web服务器',
    restart: true 
  })
});

// 控制实例操作
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
- **描述**：完全更新实例URL配置
- **认证**：需要API Key
- **请求体**：`{ "url": "新的client://或server://格式的URL" }`
- **特点**：会重启实例并重置流量统计
- **限制**：API Key实例（ID为`********`）不支持此操作
- **示例**：
```javascript
// 更新实例URL
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
- **描述**：删除实例
- **认证**：需要API Key
- **响应**：204 No Content
- **限制**：API Key实例（ID为`********`）不可删除
- **示例**：
```javascript
await fetch(`${API_URL}/instances/abc123`, {
  method: 'DELETE',
  headers: { 'X-API-Key': apiKey }
});
```

### 其他端点

#### GET /events
- **描述**：建立SSE连接以接收实时事件
- **认证**：需要API Key
- **响应**：Server-Sent Events流
- **事件类型**：`initial`, `create`, `update`, `delete`, `shutdown`, `log`

#### GET /info
- **描述**：获取主控服务信息
- **认证**：需要API Key
- **响应**：包含系统信息、版本、运行时间等

#### GET /openapi.json
- **描述**：获取OpenAPI 3.1.1规范
- **认证**：无需认证
- **响应**：JSON格式的API规范

#### GET /docs
- **描述**：Swagger UI文档界面
- **认证**：无需认证
- **响应**：HTML格式的交互式文档

### 实例URL格式规范

实例URL必须遵循以下格式：

#### 服务器模式 (Server Mode)
```
server://<bind_address>:<bind_port>/<target_host>:<target_port>?<parameters>
```

示例：
- `server://0.0.0.0:8080/localhost:3000` - 在8080端口监听，转发到本地3000端口
- `server://0.0.0.0:9090/localhost:8080?tls=1` - 启用TLS的服务器

#### 客户端模式 (Client Mode)
```
client://<server_host>:<server_port>/<local_host>:<local_port>?<parameters>
```

示例：
- `client://example.com:8080/localhost:3000` - 连接到远程服务器，本地监听3000端口
- `client://vpn.example.com:443/localhost:22?tls=1` - 通过TLS连接到VPN服务器

#### 支持的参数

| 参数 | 描述 | 值 | 默认值 |
|------|------|----|----|
| `tls` | TLS加密级别 | `0`(无), `1`(自签名), `2`(证书) | `0` |
| `log` | 日志级别 | `trace`, `debug`, `info`, `warn`, `error` | `info` |
| `crt` | 证书路径 | 文件路径 | 无 |
| `key` | 私钥路径 | 文件路径 | 无 |
