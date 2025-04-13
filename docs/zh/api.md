# NodePass API参考

## 概述

NodePass在主控模式（Master Mode）下提供了RESTful API，使前端应用能够以编程方式进行控制和集成。本节提供API端点、集成模式和最佳实践的全面文档。

## 主控模式API

当NodePass以主控模式（`master://`）运行时，它会暴露REST API，允许前端应用：

1. 创建和管理NodePass服务器和客户端实例
2. 监控连接状态和统计信息
3. 控制运行中的实例（启动、停止、重启）
4. 通过参数配置行为

### 基本URL

```
http(s)://<api_addr>[<prefix>]/v1/
```

其中：
- `<api_addr>`是主控模式URL中指定的地址（例如`0.0.0.0:9090`）
- `<prefix>`是可选的API前缀（默认为`/api`）

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
| `/v1/instances` | GET | 列出所有NodePass实例 |
| `/v1/instances` | POST | 创建新的NodePass实例 |
| `/v1/instances/{id}` | GET | 获取特定实例的详细信息 |
| `/v1/instances/{id}` | PUT | 更新或控制特定实例 |
| `/v1/instances/{id}` | DELETE | 删除特定实例 |
| `/v1/openapi.json` | GET | OpenAPI规范 |
| `/v1/docs` | GET | Swagger UI文档 |

### API认证

主控API目前尚未实现认证机制。在生产环境部署时，建议：
- 使用带有认证的反向代理
- 通过网络策略限制访问
- 启用TLS加密（`tls=1`或`tls=2`）

## 前端集成指南

在将NodePass与前端应用集成时，请考虑以下重要事项：

### 实例持久化

**重要提示：**NodePass主控模式**不会在重启之间持久保存实例配置**。当主控模式进程重启时，所有实例信息都会丢失。

前端应用应该：
1. 在自己的持久存储中保存实例配置
2. 在检测到NodePass主控重启时重新注册所有实例
3. 比较返回的实例ID与存储的ID，以检测和处理重启

重新注册逻辑示例：
```javascript
function checkAndRestoreInstances() {
  try {
    // 简单的健康检查，检测主控是否正在运行
    const response = await fetch(`${API_URL}/v1/instances`);
    
    if (response.status === 200) {
      const data = await response.json();
      
      // 如果没有实例但我们有存储的配置，主控可能已重启
      if (data.data.instances.length === 0 && storedInstances.length > 0) {
        console.log("检测到主控重启，重新注册实例...");
        
        for (const instance of storedInstances) {
          const newInstance = await createInstance(instance.url);
          // 使用新分配的ID更新存储的ID
          updateStoredInstanceId(instance.id, newInstance.data.id);
        }
      }
    }
  } catch (error) {
    console.error("NodePass主控不可达:", error);
  }
}
```

### 实例生命周期管理

为了合理管理生命周期：

1. **创建**：存储实例配置和URL
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
     
     // 存储在前端持久化存储中
     saveInstanceConfig({
       id: data.data.id,
       originalConfig: config,
       url: data.data.url
     });
     
     return data;
   }
   ```

2. **状态监控**：定期轮询状态
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

3. **控制操作**：启动、停止、重启实例
   ```javascript
   async function controlInstance(instanceId, action) {
     // action可以是: start, stop, restart
     const response = await fetch(`${API_URL}/v1/instances/${instanceId}`, {
       method: 'PUT',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({ action })
     });
     
     const data = await response.json();
     return data.success;
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

### 实例ID变化

在主控模式重启后实例ID会改变。要处理这个问题：

1. **通过URL跟踪**：使用实例URL作为稳定标识符
   ```javascript
   function findInstanceByUrl(url) {
     return storedInstances.find(instance => instance.url === url);
   }
   ```

2. **ID映射**：维护应用程序稳定ID与NodePass实例ID之间的映射
   ```javascript
   const instanceMapping = {};
   
   function updateInstanceMapping(appInstanceId, nodepassInstanceId) {
     instanceMapping[appInstanceId] = nodepassInstanceId;
   }
   
   function getNodePassId(appInstanceId) {
     return instanceMapping[appInstanceId];
   }
   ```

3. **恢复程序**：实现ID变化时的恢复程序
   ```javascript
   async function recoverInstances() {
     // 从NodePass获取所有当前实例
     const response = await fetch(`${API_URL}/v1/instances`);
     const data = await response.json();
     
     // 通过URL匹配实例
     for (const storedInstance of storedInstances) {
       const matchingInstance = data.data.instances.find(
         instance => instance.url === storedInstance.url
       );
       
       if (matchingInstance) {
         // 更新ID映射
         updateInstanceMapping(storedInstance.appId, matchingInstance.id);
       } else {
         // 实例不存在，重新创建
         const newInstance = await createInstance(storedInstance.url);
         updateInstanceMapping(storedInstance.appId, newInstance.data.id);
       }
     }
   }
   ```

## API端点文档

有关详细的API文档（包括请求和响应示例），请使用`/v1/docs`端点提供的内置Swagger UI文档。这个交互式文档提供了以下全面信息：

- 可用的端点
- 必需的参数
- 响应格式
- 请求和响应示例
- 架构定义

### 访问Swagger UI

要访问Swagger UI文档：

```
http(s)://<api_addr>[<prefix>]/v1/docs
```

例如：
```
http://localhost:9090/api/v1/docs
```

Swagger UI提供了一种方便的方式，直接在浏览器中探索和测试API。您可以针对运行中的NodePass主控实例执行API调用，并查看实际响应。

## 最佳实践

### 可扩展管理

对于管理多个NodePass实例：

1. **批量操作**：实现批量操作以管理多个实例
   ```javascript
   async function bulkControlInstances(instanceIds, action) {
     const promises = instanceIds.map(id => controlInstance(id, action));
     return Promise.all(promises);
   }
   ```

2. **连接池化**：对API请求使用连接池
   ```javascript
   const http = require('http');
   const agent = new http.Agent({ keepAlive: true, maxSockets: 50 });
   
   async function optimizedFetch(url, options = {}) {
     return fetch(url, { ...options, agent });
   }
   ```

3. **缓存**：缓存实例详情以减少API调用
   ```javascript
   const instanceCache = new Map();
   const CACHE_TTL = 60000; // 1分钟
   
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

### 监控和健康检查

实现全面监控：

1. **API健康检查**：验证主控API是否响应
   ```javascript
   async function isApiHealthy() {
     try {
       const response = await fetch(`${API_URL}/v1/instances`, {
         method: 'GET',
         timeout: 5000 // 5秒超时
       });
       
       return response.status === 200;
     } catch (error) {
       return false;
     }
   }
   ```

2. **实例健康检查**：监控单个实例健康状态
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

## 总结

NodePass主控模式API提供了强大的接口，用于以编程方式管理NodePass实例。在与前端应用集成时，特别注意：

1. **实例持久化** - 存储配置并处理重启
2. **实例ID变化** - 实现稳定的标识策略
3. **适当的错误处理** - 从API错误中优雅恢复
4. **流量统计** - 收集并可视化连接指标（需要启用调试模式）

这些指南将帮助您构建前端应用与NodePass之间的健壮集成。

有关NodePass内部机制的信息，请参阅[工作原理](/docs/zh/how-it-works.md)部分，其中包括：
- 连接池详细信息
- 信号通信协议
- 数据传输