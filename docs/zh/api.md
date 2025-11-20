# NodePass API 参考

## 概述

NodePass 主控模式（Master Mode）下提供 RESTful API，支持前端集成和自动化。本文档涵盖所有接口、数据结构和最佳实践。

## 主控模式 API

主控模式（`master://`）下，NodePass 支持：

1. 创建和管理服务端/客户端实例
2. 实时监控状态、流量、健康检查
3. 控制实例（启动、停止、重启、重置流量）
4. 配置自启动策略
5. 灵活参数配置

### 基础 URL

```
master://<api_addr>/<prefix>?<log>&<tls>
```

- `<api_addr>`：监听地址（如 `0.0.0.0:9090`）
- `<prefix>`：API 路径前缀（默认 `/api`）

### 启动主控模式

```bash
nodepass "master://0.0.0.0:9090?log=info"
nodepass "master://0.0.0.0:9090/admin?log=info&tls=1"
```

### 主要接口

| Endpoint           | Method | 说明                 |
|--------------------|--------|----------------------|
| `/instances`       | GET    | 获取所有实例         |
| `/instances`       | POST   | 创建新实例           |
| `/instances/{id}`  | GET    | 获取实例详情         |
| `/instances/{id}`  | PATCH  | 更新/控制实例        |
| `/instances/{id}`  | PUT    | 更新实例 URL         |
| `/instances/{id}`  | DELETE | 删除实例             |
| `/events`          | GET    | SSE 实时事件流       |
| `/info`            | GET    | 获取主控服务信息     |
| `/info`            | POST   | 更新主控别名         |
| `/tcping`          | GET    | TCP连接测试          |
| `/openapi.json`    | GET    | OpenAPI 规范         |
| `/docs`            | GET    | Swagger UI 文档      |

### API 鉴权

API Key 认证默认启用，首次启动自动生成并保存在 `nodepass.gob`。

- 受保护接口：`/instances`、`/instances/{id}`、`/events`、`/info`、`/tcping`
- 公共接口：`/openapi.json`、`/docs`
- 认证方式：请求头加 `X-API-Key: <key>`
- 重置 Key：PATCH `/instances/********`，body `{ "action": "restart" }`

### 实例数据结构

```json
{
  "id": "a1b2c3d4",
  "alias": "别名",
  "type": "client|server",
  "status": "running|stopped|error",
  "url": "...",
  "config": "server://0.0.0.0:8080/localhost:3000?log=info&tls=1&dns=1.1.1.1,8.8.8.8&max=1024&mode=0&quic=0&dial=auto&read=1h&rate=100&slot=65536&proxy=0&notcp=0&noudp=0",
  "restart": true,
  "meta": {
    "peer": {
      "sid": "550e8400-e29b-41d4-a716-446655440000",
      "type": "1",
      "alias": "远程服务"
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

- `mode`：实例运行模式
- `ping`/`pool`：健康检查数据
- `tcps`/`udps`：当前活动连接数统计
- `tcprx`/`tcptx`/`udprx`/`udptx`：累计流量统计
- `config`：实例配置URL，包含完整的启动配置
- `restart`：自启动策略
- `meta`：元数据信息，用于实例组织和对端识别
  - `peer`：对端连接信息（远程端点详情）
    - `sid`：远程服务的服务ID，使用UUID v4格式（如 `550e8400-e29b-41d4-a716-446655440000`）
    - `type`：远程服务类型，使用标准枚举值
      - `"0"`：单端转发模式（Single-end Forwarding）
      - `"1"`：内网穿透模式（NAT Traversal）
      - `"2"`：隧道转发模式（Tunnel Forwarding）
    - `alias`：远程端点的服务别名（无格式限制）
  - `tags`：自定义键值对标签，用于灵活分类和筛选

### 实例 URL 格式

- 服务端：`server://<bind_addr>:<bind_port>/<target_host>:<target_port>?<参数>`
- 客户端：`client://<server_host>:<server_port>/<local_host>:<local_port>?<参数>`
- 支持参数：`log`、`tls`、`crt`、`key`、`dns`、`min`、`max`、`mode`、`quic`、`dial`、`read`、`rate`、`slot`、`proxy`、`notcp`、`noudp`

### URL 查询参数

- `log`：日志级别（`none`、`debug`、`info`、`warn`、`error`、`event`）
- `tls`：TLS加密模式（`0`、`1`、`2`）- 仅服务端/主控模式
- `crt`/`key`：证书/密钥文件路径（当`tls=2`时）
- `dns`：自定义DNS服务器（逗号分隔的IP地址，默认：`1.1.1.1,8.8.8.8`）- 仅服务端/客户端模式
- `min`/`max`：连接池容量（`min`由客户端设置，`max`由服务端设置并在握手时传递给客户端）
- `mode`：运行模式控制（`0`、`1`、`2`）- 控制操作行为
  - 对于服务端：`0`=自动，`1`=反向模式，`2`=正向模式
  - 对于客户端：`0`=自动，`1`=单端转发，`2`=双端握手
- `quic`：QUIC传输模式（`0`=TCP连接池，`1`=QUIC连接池，默认：`0`）- 仅服务端配置，客户端在握手时接收配置
- `dial`：出站连接的源IP地址（默认：`auto`）- 仅服务端/客户端模式
- `read`：数据读取超时时长（如1h、30m、15s，默认：`0`表示无超时）
- `rate`：带宽速率限制，单位Mbps（0=无限制）
- `slot`：最大并发连接数限制（默认：`65536`，0=无限制）
- `proxy`：PROXY协议支持（`0`、`1`）- 启用后在数据传输前发送PROXY协议v1头部
- `notcp`：TCP支持控制（`0`=启用，`1`=禁用）- 仅服务端/客户端模式
- `noudp`：UDP支持控制（`0`=启用，`1`=禁用）- 仅服务端/客户端模式

### 实时事件流（SSE）

- 事件类型：`initial`、`create`、`update`、`delete`、`shutdown`、`log`
- `log` 事件仅推送普通日志，流量/健康检查日志已被过滤
- 连接 `/events` 可实时获取实例变更和日志

### 其他说明

- 所有实例、流量、健康检查、别名、自启动策略均持久化存储，重启后自动恢复
- API 详细规范见 `/openapi.json`，Swagger UI 见 `/docs`

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

// 获取主控ID（Master ID）
async function getMasterID() {
  const response = await fetch(`${API_URL}/instances/${apiKeyID}`, {
    method: 'GET',
    headers: {
      'X-API-Key': 'current-api-key'
    }
  });
  
  const result = await response.json();
  return result.data.config; // 主控ID（16位十六进制）
}
```

**注意**: 
- API Key ID 固定为 `********`（八个星号）。在内部实现中，这是一个特殊的实例ID，用于存储和管理API Key。
- API Key实例的 `config` 字段存储**主控ID**（Master ID），这是一个16位十六进制字符串（如 `1a2b3c4d5e6f7890`），用于唯一标识主控服务。
- 主控ID在首次启动时自动生成并持久化保存，在主控服务的整个生命周期中保持不变。

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

#### 处理实例日志

在前端应用中，可以通过监听`log`事件来处理实例日志。以下是一个示例函数，用于将日志追加到特定实例的UI中：

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

日志集成最佳实践：

1. **缓冲管理**：限制日志条目的数量，以防止内存问题
2. **ANSI颜色解析**：解析日志中的ANSI颜色代码，以提高可读性
3. **过滤选项**：提供按严重性或内容过滤日志的选项
4. **搜索功能**：允许用户在实例日志中搜索
5. **日志持久化**：可选地将日志保存到本地存储，以便在页面刷新后查看

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

#### 自动备份功能

NodePass主控模式提供自动备份功能，定期备份状态文件以防止数据丢失：

- **备份文件**：自动创建 `nodepass.gob.backup` 备份文件
- **备份周期**：每1小时自动备份一次（可通过环境变量 `NP_RELOAD_INTERVAL` 配置）
- **备份策略**：使用单一备份文件，新备份会覆盖旧备份
- **备份内容**：包含所有实例配置、状态、自启动策略和统计数据
- **故障恢复**：当主文件损坏时，可手动使用备份文件恢复
- **自动启动**：备份功能随主控服务自动启动，无需额外配置

备份文件位置：与主状态文件 `nodepass.gob` 相同目录下的 `nodepass.gob.backup`

**注意：** 虽然实例配置现在已经持久化并自动备份，前端应用仍应保留自己的实例配置记录作为额外的备份策略。

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
   
   // 更新实例元数据
   async function updateInstanceMetadata(instanceId, metadata) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey // 如果启用了API Key 
       },
       body: JSON.stringify({ meta: metadata })
     });
     
     const data = await response.json();
     return data.success;
   }
   ```

5. **元数据管理**：使用元数据组织和分类实例
   ```javascript
   // 设置对端连接信息
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
             sid: peerInfo.serviceId, // UUID v4 格式
             type: peerInfo.type, // "0" | "1" | "2"
             alias: peerInfo.alias
           },
           tags: {} // 保留现有标签
         }
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // 添加或更新实例标签
   async function updateInstanceTags(instanceId, tags) {
     const response = await fetch(`${API_URL}/instances/${instanceId}`, {
       method: 'PATCH',
       headers: { 
         'Content-Type': 'application/json',
         'X-API-Key': apiKey
       },
       body: JSON.stringify({
         meta: {
           peer: {}, // 保留现有对端信息
           tags: tags
         }
       })
     });
     
     const data = await response.json();
     return data.success;
   }
   
   // 完整元数据更新
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

6. **自启动策略管理**：配置自动启动行为
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

#### 元数据管理使用示例

以下是展示如何使用元数据进行实例组织和管理的综合示例：

```javascript
// 示例1：建立带有元数据的点对点隧道
async function establishPeerTunnel(localConfig, remoteConfig) {
  // 创建本地服务器实例
  const localInstance = await createNodePassInstance({
    type: 'server',
    port: localConfig.port,
    target: localConfig.target
  });
  
  // 创建远程客户端实例
  const remoteInstance = await createNodePassInstance({
    type: 'client',
    serverHost: localConfig.serverHost,
    port: remoteConfig.port,
    target: remoteConfig.target
  });
  
  if (localInstance.success && remoteInstance.success) {
    // 在本地实例上设置对端信息
    await updateCompleteMetadata(
      localInstance.data.id,
      {
        sid: remoteConfig.serviceId, // UUID格式
        type: "2", // 隧道转发
        alias: remoteConfig.serviceName
      },
      {
        tunnel_type: 'peer-to-peer',
        protocol: 'tcp',
        encryption: 'tls'
      }
    );
    
    // 在远程实例上设置对端信息
    await updateCompleteMetadata(
      remoteInstance.data.id,
      {
        sid: localConfig.serviceId, // UUID格式
        type: "2", // 隧道转发
        alias: localConfig.serviceName
      },
      {
        tunnel_type: 'peer-to-peer',
        protocol: 'tcp',
        encryption: 'tls'
      }
    );
    
    console.log('已建立带有元数据的对等隧道');
  }
}

// 示例2：按环境和区域组织实例
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
    console.log(`已为实例 ${instance.id} 设置环境元数据标签`);
  }
}

// 示例3：通过元数据标签查询实例
async function findInstancesByTags(requiredTags) {
  const response = await fetch(`${API_URL}/instances`, {
    headers: { 'X-API-Key': apiKey }
  });
  const data = await response.json();
  
  if (data.success) {
    return data.data.filter(instance => {
      if (!instance.meta || !instance.meta.tags) return false;
      
      // 检查所有必需标签是否匹配
      return Object.entries(requiredTags).every(([key, value]) => 
        instance.meta.tags[key] === value
      );
    });
  }
  return [];
}

// 示例4：根据运行状态更新元数据
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

#### 元数据最佳实践

1. **对端信息**：使用 `peer` 对象跟踪实例之间的连接
   - `sid`：服务唯一标识符（必填，UUID v4格式，如 `550e8400-e29b-41d4-a716-446655440000`）
     - 使用标准UUID v4格式确保全局唯一性
     - 可使用JavaScript的 `crypto.randomUUID()` 或第三方库生成
   - `type`：服务类型标识（必填，字符串枚举值）
     - `"0"`：单端转发（Single-end Forwarding）- 适用于简单的客户端转发场景
     - `"1"`：内网穿透（NAT Traversal）- 适用于需要穿透NAT的场景
     - `"2"`：隧道转发（Tunnel Forwarding）- 适用于建立加密隧道的场景
   - `alias`：远程服务的友好名称（无格式限制，最多256字符）

2. **前端集成标准**：为确保一致性，前端应遵循以下标准
   
   **服务ID（sid）生成标准：**
   ```javascript
   // 使用浏览器原生API生成UUID v4
   const serviceId = crypto.randomUUID();
   // 示例输出: "550e8400-e29b-41d4-a716-446655440000"
   
   // 或使用第三方库（如uuid）
   import { v4 as uuidv4 } from 'uuid';
   const serviceId = uuidv4();
   ```
   
   **服务类型（type）使用标准：**
   ```javascript
   // 定义服务类型枚举
   const ServiceType = {
     SINGLE_END: "0",      // 单端转发：客户端单向转发，无需服务端回连
     NAT_TRAVERSAL: "1",   // 内网穿透：穿透NAT进行内网访问
     TUNNEL: "2"           // 隧道转发：建立端到端加密隧道
   };
   
   // 使用示例
   const peerInfo = {
     sid: crypto.randomUUID(),
     type: ServiceType.NAT_TRAVERSAL,
     alias: "Web服务器"
   };
   ```
   
   **类型选择指南：**
   - **单端转发（"0"）**：
     - 场景：客户端仅需要将流量转发到远程服务器
     - 特点：单向连接，无需服务端主动回连
     - 示例：本地应用连接到云端数据库
   
   - **内网穿透（"1"）**：
     - 场景：需要从外网访问内网服务
     - 特点：穿透NAT和防火墙限制
     - 示例：远程访问家庭NAS、内网Web服务
   
   - **隧道转发（"2"）**：
     - 场景：需要建立安全的端到端连接
     - 特点：加密传输，双向通信
     - 示例：分支机构与总部的安全互联

3. **标签组织**：设计一致的标签策略
   - 使用小写字母和下划线的键名（如 `cost_center`、`deployment_region`）
   - 将标签值限制为有意义的、可搜索的字符串
   - 常见标签类别：
     - 环境：`production`、`staging`、`development`
     - 位置：`us-west`、`eu-central`、`ap-southeast`
     - 所有权：`team-alpha`、`ops-team`、`platform-team`
     - 功能：`database-tunnel`、`web-proxy`、`api-gateway`
     - 重要性：`high`、`medium`、`low`

4. **字段长度限制**：元数据字段的长度要求
   - `peer.sid`：固定36字符（UUID v4格式，如 `550e8400-e29b-41d4-a716-446655440000`）
   - `peer.type`：固定1字符（枚举值：`"0"` | `"1"` | `"2"`）
   - `peer.alias`：最多256字符（无特定格式要求）
   - 标签键和值：每个最多256字符

5. **标签唯一性**：确保实例内的标签键唯一
   - 重复的键将导致400 Bad Request错误

6. **过滤和搜索**：使用元数据进行实例过滤
   - 客户端按标签过滤以显示仪表板视图
   - 通过对端信息查询实例以进行关系映射
   - 按标签分组实例以进行批量操作

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

1. **基本流量指标**：NodePass周期性地提供TCP和UDP流量在入站和出站方向上的累计值，前端应用需要存储和处理这些值以获得有意义的统计信息。
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

2. **数据持久化**：由于API只提供累计值，前端必须实现适当的存储和计算逻辑
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
                            instance.critical === true;
    
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
  "config": "server://0.0.0.0:8080/localhost:3000?log=info&tls=1&dns=1.1.1.1,8.8.8.8&max=1024&mode=0&quic=0&dial=auto&read=1h&rate=100&slot=65536&proxy=0&notcp=0&noudp=0", // 完整配置URL
  "restart": true,            // 自启动策略
  "meta": {                   // 用于组织和对端跟踪的元数据
    "peer": {
      "sid": "550e8400-e29b-41d4-a716-446655440000",    // 远程服务ID（UUID格式）
      "type": "1",             // 远程服务类型（0=单端转发，1=内网穿透，2=隧道转发）
      "alias": "远程服务"       // 远程服务友好名称
    },
    "tags": {                  // 自定义键值对标签
      "environment": "production",
      "region": "us-west",
      "team": "platform"
    }
  },
  "mode": 0,                  // 运行模式
  "tcprx": 1024,              // TCP接收字节数
  "tcptx": 2048,              // TCP发送字节数
  "udprx": 512,               // UDP接收字节数
  "udptx": 256                // UDP发送字节数
}
```

**注意：** 
- `alias` 字段为可选，如果未设置则为空字符串
- `config` 字段包含实例的完整配置URL，由系统自动生成
- `mode` 字段表示实例当前的运行模式
- `restart` 字段控制实例的自启动行为
- `meta` 字段包含用于实例组织的结构化元数据
  - `peer` 对象跟踪点对点连接的远程端点信息
    - `sid`：服务唯一标识符，必须使用UUID v4格式（36字符，如 `550e8400-e29b-41d4-a716-446655440000`）
    - `type`：服务类型标识，字符串枚举值（`"0"` | `"1"` | `"2"`）
      - `"0"`：单端转发（Single-end Forwarding） - 客户端单向转发流量
      - `"1"`：内网穿透（NAT Traversal） - 穿透NAT进行内网访问
      - `"2"`：隧道转发（Tunnel Forwarding） - 建立端到端加密隧道
    - `alias`：自定义字符串，最多256字符，无格式限制
  - `tags` 映射允许使用自定义键值对进行灵活分类
  - 标签键和值最大长度为256个字符
  - 标签键在实例内必须唯一

### 实例配置字段

NodePass主控会自动为每个实例维护 `config` 字段：

- **自动生成**：在实例创建和更新时自动生成，无需手动维护
- **完整配置**：包含实例的完整URL，带有所有默认参数
- **配置继承**：log和tls配置继承自主控设置
- **默认参数**：其他参数使用系统默认值
- **只读性质**：自动生成的字段，通过API无法直接修改

**示例 config 字段值：**
```
server://0.0.0.0:8080/localhost:3000?log=info&tls=1&max=1024&mode=0&read=1h&rate=0&slot=65536&proxy=0
```

此功能特别适用于：
- 配置备份和导出
- 实例配置的完整性检查
- 自动化部署脚本
- 配置文档生成

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
  "alias": "dev",             // 主控别名
  "os": "linux",              // 操作系统类型
  "arch": "amd64",            // 系统架构
  "cpu": 45,                  // CPU使用率百分比（仅Linux系统）
  "mem_total": 8589934592,    // 内存容量（字节，仅Linux系统）
  "mem_used": 2684354560,     // 内存已用（字节，仅Linux系统）
  "swap_total": 3555328000,   // 交换区总量（字节，仅Linux系统）
  "swap_used": 3555328000,    // 交换区已用（字节，仅Linux系统）
  "netrx": 1048576000,        // 网络接收字节数（累计值，仅Linux）
  "nettx": 2097152000,        // 网络发送字节数（累计值，仅Linux）
  "diskr": 4194304000,        // 磁盘读取字节数（累计值，仅Linux）
  "diskw": 8388608000,        // 磁盘写入字节数（累计值，仅Linux）
  "sysup": 86400,             // 系统运行时间（秒，仅Linux）
  "ver": "1.2.0",             // NodePass版本
  "name": "example.com",      // 隧道主机名
  "uptime": 11525,            // API运行时间（秒）
  "log": "info",              // 日志级别
  "tls": "1",                 // TLS启用状态
  "crt": "/path/to/cert",     // 证书路径
  "key": "/path/to/key"       // 密钥路径
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

// 显示服务运行时间和系统资源使用情况
function displaySystemStatus() {
  getSystemInfo().then(info => {
    console.log(`服务已运行: ${info.uptime} 秒`);
    
    // 格式化运行时间为更友好的显示
    const hours = Math.floor(info.uptime / 3600);
    const minutes = Math.floor((info.uptime % 3600) / 60);
    const seconds = info.uptime % 60;
    console.log(`服务已运行: ${hours}小时${minutes}分${seconds}秒`);
    
    // 显示系统资源使用情况（仅Linux系统）
    if (info.os === 'linux') {
      if (info.cpu !== -1) {
        console.log(`CPU使用率: ${info.cpu}%`);
      }
      if (info.mem_total > 0) {
        const memUsagePercent = (info.mem_used / info.mem_total * 100).toFixed(1);
        const memFreeGB = ((info.mem_total - info.mem_used) / 1024 / 1024 / 1024).toFixed(1);
        const memTotalGB = (info.mem_total / 1024 / 1024 / 1024).toFixed(1);
        console.log(`内存使用率: ${memUsagePercent}% (${memFreeGB}GB 可用，共 ${memTotalGB}GB)`);
      }
      if (info.swap_total > 0) {
        const swapUsagePercent = (info.swap_used / info.swap_total * 100).toFixed(1);
        const swapFreeGB = ((info.swap_total - info.swap_used) / 1024 / 1024 / 1024).toFixed(1);
        const swapTotalGB = (info.swap_total / 1024 / 1024 / 1024).toFixed(1);
        console.log(`交换区使用率: ${swapUsagePercent}% (${swapFreeGB}GB 可用，共 ${swapTotalGB}GB)`);
      }
    } else {
      console.log('CPU、内存、交换区、网络I/O、磁盘I/O和系统运行时间监控功能仅在Linux系统上可用');
    }
    
    // 显示网络I/O统计（累计值）
    if (info.os === 'linux') {
      console.log(`网络接收: ${(info.netrx / 1024 / 1024).toFixed(2)} MB（累计）`);
      console.log(`网络发送: ${(info.nettx / 1024 / 1024).toFixed(2)} MB（累计）`);
      console.log(`磁盘读取: ${(info.diskr / 1024 / 1024).toFixed(2)} MB（累计）`);
      console.log(`磁盘写入: ${(info.diskw / 1024 / 1024).toFixed(2)} MB（累计）`);
      console.log(`系统运行时间: ${Math.floor(info.sysup / 3600)}小时`);
    }
  });
}
```

### 监控最佳实践

- **定期检查**：定期轮询此端点以确保服务正常运行
- **版本验证**：在部署更新后检查版本号
- **运行时间监控**：监控运行时间以检测意外重启
- **日志级别验证**：确认当前日志级别符合预期
- **资源监控**：在Linux系统上，监控CPU、内存、交换区、网络I/O、磁盘I/O使用情况以确保最佳性能
  - CPU使用率通过解析`/proc/stat`计算（非空闲时间百分比）
  - 内存信息通过解析`/proc/meminfo`获取（总量和已用量，单位为字节，已用量计算为总量减去可用量）
  - 交换区信息通过解析`/proc/meminfo`获取（总量和已用量，单位为字节，已用量计算为总量减去空闲量）
  - 网络I/O通过解析`/proc/net/dev`计算（累计字节数，排除虚拟接口）
  - 磁盘I/O通过解析`/proc/diskstats`计算（累计字节数，仅统计主设备）
  - 系统运行时间通过解析`/proc/uptime`获取
  - 值为-1或0表示系统信息不可用（非Linux系统）
  - 网络和磁盘I/O字段提供的是累计值，前端应用需要存储历史数据并计算差值来得到实时速率（字节/秒）

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
- **描述**：更新实例状态、别名、元数据或执行控制操作
- **认证**：需要API Key
- **请求体**：`{ "alias": "新别名", "action": "start|stop|restart|reset", "restart": true|false, "meta": {...} }`
- **元数据结构**：
  - `peer`：对象，包含以下字段（均为可选）：
    - `sid`：服务ID（UUID v4格式，36字符，如 `550e8400-e29b-41d4-a716-446655440000`）
    - `type`：服务类型（枚举值：`"0"` | `"1"` | `"2"`）
      - `"0"`：单端转发（Single-end Forwarding）
      - `"1"`：内网穿透（NAT Traversal）
      - `"2"`：隧道转发（Tunnel Forwarding）
    - `alias`：服务别名（最多256字符，无格式限制）
  - `tags`：自定义键值对对象（键和值最多256字符，键必须唯一）
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
    alias: "生产服务器",
    restart: true
  })
});

// 执行控制操作
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

// 更新元数据（包含对端信息和标签）
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    meta: {
      peer: {
        sid: "550e8400-e29b-41d4-a716-446655440000", // UUID格式
        type: "1", // 内网穿透
        alias: "远程API服务器"
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

// 仅更新标签（对端信息保持不变）
await fetch(`${API_URL}/instances/abc123`, {
  method: 'PATCH',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ 
    meta: {
      peer: {},  // 空对象保留现有对端信息
      tags: {
        environment: "staging",
        updated_at: new Date().toISOString()
      }
    }
  })
});
```

#### PUT /instances/{id}
- **描述**：完全更新实例URL配置
- **认证**：需要API Key
- **请求体**：`{ "url": "新的client://或server://格式的URL" }`
- **特点**：会重启实例。
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
- **响应**：包含系统信息、版本、运行时间、CPU和RAM使用率等

#### POST /info
- **描述**：更新主控别名
- **认证**：需要API Key
- **请求体**：`{ "alias": "新别名" }`
- **响应**：完整的主控信息（与GET /info相同）
- **说明**：主控别名存储在API Key实例（ID为`********`）的 `alias` 字段中
- **示例**：
```javascript
// 更新主控别名
const response = await fetch(`${API_URL}/info`, {
  method: 'POST',
  headers: { 
    'Content-Type': 'application/json',
    'X-API-Key': apiKey 
  },
  body: JSON.stringify({ alias: '我的NodePass服务器' })
});

const data = await response.json();
console.log('更新后的别名:', data.alias);
// 响应包含完整的系统信息，包括更新后的别名
```

**主控ID获取**：主控ID存储在API Key实例的 `config` 字段中，可以通过以下方式获取：
```javascript
// 获取主控ID
async function getMasterID() {
  const response = await fetch(`${API_URL}/instances/********`, {
    headers: { 'X-API-Key': apiKey }
  });
  const data = await response.json();
  return data.data.config; // 返回16位十六进制的主控ID
}
```

#### GET /tcping
- **描述**：TCP连接测试，检测目标地址的连通性和延迟
- **认证**：需要API Key
- **参数**：
  - `target`（必需）：目标地址，格式为 `host:port`
- **响应**：
  ```json
  {
    "target": "example.com:80",
    "connected": true,
    "latency": 45,
    "error": null
  }
  ```
- **示例**：`GET /api/tcping?target=fast.com:443`

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
- `server://0.0.0.0:9090/localhost:8080?tls=1&mode=1` - 启用TLS的服务器，强制反向模式

#### 客户端模式 (Client Mode)
```
client://<server_host>:<server_port>/<local_host>:<local_port>?<parameters>
```

示例：
- `client://example.com:8080/localhost:3000` - 连接到远程服务器，本地监听3000端口
- `client://remote.example.com:443/localhost:22?mode=2&min=32` - 通过远程服务器，强制双端模式

#### 支持的参数

| 参数 | 描述 | 值 | 默认值 | 适用范围 |
|------|------|----|----|---------|
| `log` | 日志级别 | `none`, `debug`, `info`, `warn`, `error`, `event` | `info` | 两者 |
| `tls` | TLS加密级别 | `0`(无), `1`(自签名), `2`(证书) | `0` | 仅服务端 |
| `crt` | 证书路径 | 文件路径 | 无 | 仅服务端 |
| `key` | 私钥路径 | 文件路径 | 无 | 仅服务端 |
| `dns` | 自定义DNS服务器 | IP地址，逗号分隔 | `1.1.1.1,8.8.8.8` | 两者 |
| `min` | 最小连接池容量 | 整数 > 0 | `64` | 仅客户端双端握手模式 |
| `max` | 最大连接池容量 | 整数 > 0 | `1024` | 双端握手模式 |
| `mode` | 运行模式控制 | `0`(自动), `1`(强制模式1), `2`(强制模式2) | `0` | 两者 |
| `quic` | QUIC协议支持 | `0`(禁用), `1`(启用) | `0` | 仅服务端 |
| `dial` | 出站源IP地址 | IP地址或 `auto` | `auto` | 两者 |
| `read` | 读取超时时间 | 时间长度 (如 `10m`, `30s`, `1h`) | `0` | 两者 |
| `rate` | 带宽速率限制 | 整数 (Mbps), 0=无限制 | `0` | 两者 |
| `slot` | 连接槽位数 | 整数 (1-65536) | `65536` | 两者 |
| `proxy` | PROXY协议支持 | `0`(禁用), `1`(启用) | `0` | 两者 |
| `notcp` | TCP支持控制 | `0`(启用), `1`(禁用) | `0` | 两者 |
| `noudp` | UDP支持控制 | `0`(启用), `1`(禁用) | `0` | 两者 |