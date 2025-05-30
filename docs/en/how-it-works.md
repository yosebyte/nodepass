# How NodePass Works

This page explains the internal architecture and data flow mechanisms of NodePass, providing insights into how the different components interact to create efficient and secure tunnels.

## Architecture Overview

NodePass creates a network architecture with separate channels for control and data:

1. **Control Channel (Tunnel)**:
   - Unencrypted TCP connection between client and server
   - Used exclusively for signaling and coordination
   - Maintains persistent connection for the lifetime of the tunnel

2. **Data Channel (Target)**:
   - Configurable TLS encryption options:
     - **Mode 0**: Unencrypted data transfer (fastest, least secure)
     - **Mode 1**: Self-signed certificate encryption (good security, no verification)
     - **Mode 2**: Verified certificate encryption (highest security, requires valid certificates)
   - Created on-demand for each connection or datagram
   - Used for actual application data transfer

3. **Server Mode Operation**:
   - Listens for control connections on the tunnel endpoint
   - When traffic arrives at the target endpoint, signals the client via the control channel
   - Establishes data channels with the specified TLS mode when needed
   - Supports bidirectional data flow: connections can be initiated from either server or client side

4. **Client Mode Operation**:
   - Connects to the server's control channel
   - Listens for signals indicating incoming connections
   - Creates data connections using the TLS security level specified by the server
   - Forwards data between the secure channel and local target
   - Supports bidirectional data flow: data flow direction is automatically selected based on target address

5. **Client Single-End Forwarding Mode**:
   - Automatically enabled when tunnel address is a local address (e.g., 127.0.0.1)
   - Client directly listens on local port without server control channel coordination
   - Uses connection pooling technology for TCP connections to significantly improve forwarding performance
   - Suitable for pure local forwarding scenarios, reducing network overhead and latency
   - Supports high-performance single-end forwarding for both TCP and UDP protocols

5. **Protocol Support**:
   - **TCP**: Full bidirectional streaming with persistent connections, supports connection pool optimization in client single-end forwarding mode
   - **UDP**: Datagram forwarding with configurable buffer sizes and timeouts

## Data Transmission Flow

NodePass establishes a bidirectional data flow through its tunnel architecture, supporting both TCP and UDP protocols. The system supports three data flow modes:

### Data Flow Mode Explanation
- **Server Receives Mode (dataFlow: "-")**: Server listens on target address, client listens locally, data flows from target address to client local
- **Server Sends Mode (dataFlow: "+")**: Server connects to remote target address, client listens locally, data flows from client local to remote target
- **Client Single-End Forwarding Mode**: Client directly listens locally and forwards to target address without server coordination, using connection pooling technology for high-performance forwarding

The data flow mode is automatically determined based on tunnel address and target address:
- If tunnel address is a local address (localhost, 127.0.0.1, etc.), enables Client Single-End Forwarding Mode
- If target address is a local address, uses Server Receives Mode
- If target address is a remote address, uses Server Sends Mode

### Server-Side Flow (Server Receives Mode)
1. **Connection Initiation**:
   ```
   [Target Client] → [Target Listener] → [Server: Target Connection Created]
   ```
   - For TCP: Client establishes persistent connection to target listener
   - For UDP: Server receives datagrams on UDP socket bound to target address

2. **Signal Generation**:
   ```
   [Server] → [Generate Unique Connection ID] → [Signal Client via Unencrypted TCP Tunnel]
   ```
   - For TCP: Generates a `//<connection_id>#1` signal
   - For UDP: Generates a `//<connection_id>#2` signal

3. **Connection Preparation**:
   ```
   [Server] → [Create Remote Connection in Pool with Configured TLS Mode] → [Wait for Client Connection]
   ```
   - Both protocols use the same connection pool mechanism with unique connection IDs
   - TLS configuration applied based on the specified mode (0, 1, or 2)

4. **Data Exchange**:
   ```
   [Target Connection] ⟷ [Exchange/Transfer] ⟷ [Remote Connection]
   ```
   - For TCP: Uses `conn.DataExchange()` for continuous bidirectional data streaming
   - For UDP: Individual datagrams are forwarded with configurable buffer sizes

### Client-Side Flow
1. **Signal Reception**:
   ```
   [Client] → [Read Signal from TCP Tunnel] → [Parse Connection ID]
   ```
   - Client differentiates between TCP and UDP signals based on URL scheme

2. **Connection Establishment**:
   ```
   [Client] → [Retrieve Connection from Pool] → [Connect to Remote Endpoint]
   ```
   - Connection management is protocol-agnostic at this stage

3. **Local Connection**:
   ```
   [Client] → [Connect to Local Target] → [Establish Local Connection]
   ```
   - For TCP: Establishes persistent TCP connection to local target
   - For UDP: Creates UDP socket for datagram exchange with local target

4. **Data Exchange**:
   ```
   [Remote Connection] ⟷ [Exchange/Transfer] ⟷ [Local Target Connection]
   ```
   - For TCP: Uses `conn.DataExchange()` for continuous bidirectional data streaming
   - For UDP: Reads single datagram, forwards it, waits for response with timeout, then returns response

### Client Single-End Forwarding Flow
1. **Mode Detection**:
   ```
   [Client] → [Detect Tunnel Address as Local Address] → [Enable Single-End Forwarding Mode]
   ```
   - Automatically detects if tunnel address is localhost, 127.0.0.1, or other local addresses
   - Enables single-end forwarding mode, skipping server control channel establishment

2. **Local Listening**:
   ```
   [Client] → [Start Listener on Tunnel Port] → [Wait for Local Connections]
   ```
   - Directly starts TCP or UDP listener on specified tunnel port
   - No need to connect to remote server, achieving zero-latency startup

3. **Connection Pool Initialization** (TCP Only):
   ```
   [Client] → [Initialize Target Connection Pool] → [Pre-establish Connections to Target Address]
   ```
   - Creates high-performance connection pool for TCP forwarding
   - Pre-establishes multiple connections to target address, significantly reducing connection establishment latency
   - Connection pool size can be dynamically adjusted based on concurrent demand

4. **High-Performance Forwarding**:
   ```
   [Local Connection] → [Get Target Connection from Pool] → [Direct Data Exchange] → [Connection Reuse or Release]
   ```
   - For TCP: Quickly gets pre-established target connection from pool for efficient data exchange
   - For UDP: Directly forwards datagrams to target address without connection pool
   - Optimized data path minimizing forwarding overhead and latency

### Protocol-Specific Characteristics
- **TCP Exchange**: 
  - Persistent connections for full-duplex communication
  - Continuous data streaming until connection termination
  - Error handling with automatic reconnection
  - **Client Single-End Forwarding Optimization**: Pre-established connections through connection pooling technology, significantly reducing connection establishment latency

- **UDP Exchange**:
  - One-time datagram forwarding with configurable buffer sizes (`UDP_DATA_BUF_SIZE`)
  - Read timeout control for response waiting (`UDP_READ_TIMEOUT`)
  - Optimized for low-latency, stateless communications
  - **Client Single-End Forwarding Optimization**: Direct forwarding mechanism without connection pool, achieving minimal latency

## Signal Communication Mechanism

NodePass uses a sophisticated URL-based signaling protocol through the TCP tunnel:

### Signal Types
1. **Tunnel Signal**:
   - Format: `#<tls>`
   - Purpose: Informs the client about the tls code
   - Timing: Sent on tunnel handshake

2. **TCP Launch Signal**:
   - Format: `//<connection_id>#1`
   - Purpose: Requests the client to establish a TCP connection for a specific ID
   - Timing: Sent when a new TCP connection to the target service is received

3. **UDP Launch Signal**:
   - Format: `//<connection_id>#2`
   - Purpose: Requests the client to handle UDP traffic for a specific ID
   - Timing: Sent when UDP data is received on the target port

### Signal Flow
1. **Signal Generation**:
   - Server creates URL-formatted signals for specific events
   - Signal is terminated with a newline character for proper parsing

2. **Signal Transmission**:
   - Server writes signals to the TCP tunnel connection
   - Uses a mutex to prevent concurrent writes to the tunnel

3. **Signal Reception**:
   - Client uses a buffered reader to read signals from the tunnel
   - Signals are trimmed and parsed into URL format

4. **Signal Processing**:
   - Client places valid signals in a buffered channel (signalChan)
   - A dedicated goroutine processes signals from the channel
   - Semaphore pattern prevents signal overflow

5. **Signal Execution**:
   - Remote signals update the client's remote address configuration
   - Launch signals trigger the `clientOnce()` method to establish connections

### Signal Resilience
- Buffered channel with configurable capacity prevents signal loss during high load
- Semaphore implementation ensures controlled concurrency
- Error handling for malformed or unexpected signals

## Connection Pool Architecture

NodePass implements an efficient connection pooling system for managing network connections:

### Pool Design
1. **Pool Types**:
   - **Client Pool**: Pre-establishes connections to the remote endpoint
   - **Server Pool**: Manages incoming connections from clients

2. **Pool Components**:
   - **Connection Storage**: Thread-safe map of connection IDs to net.Conn objects
   - **ID Channel**: Buffered channel for available connection IDs
   - **Capacity Management**: Dynamic adjustment based on usage patterns
   - **Interval Control**: Time-based throttling between connection creations
   - **Connection Factory**: Customizable connection creation function

### Connection Lifecycle
1. **Connection Creation**:
   - Connections are created up to the configured capacity
   - Each connection is assigned a unique ID
   - IDs and connections are stored in the pool

2. **Connection Acquisition**:
   - Client retrieves connections using connection IDs
   - Server retrieves the next available connection from the pool
   - Connections are validated before being returned

3. **Connection Usage**:
   - Connection is removed from the pool when acquired
   - Used for data exchange between endpoints
   - No connection reuse (one-time use model)

4. **Connection Termination**:
   - Connections are closed after use
   - Resources are properly released
   - Error handling ensures clean termination

### Pool Management
1. **Capacity Control**:
   - `MIN_POOL_CAPACITY`: Ensures minimum available connections
   - `MAX_POOL_CAPACITY`: Prevents excessive resource consumption
   - Dynamic scaling based on demand patterns

2. **Interval Control**:
   - `MIN_POOL_INTERVAL`: Minimum time between connection creation attempts
   - `MAX_POOL_INTERVAL`: Maximum time between connection creation attempts
   - Adaptive time-based throttling to optimize resource usage

3. **Dynamic Pool Adaptation**:
   The connection pool employs a dual-adaptive mechanism to ensure optimal performance:
   
   **A. Capacity Adjustment**
   - Pool capacity dynamically adjusts based on real-time usage patterns
   - If connection creation success rate is low (<20%), capacity decreases to minimize resource waste
   - If connection creation success rate is high (>80%), capacity increases to accommodate higher traffic
   - Gradual scaling prevents oscillation and provides stability
   - Respects configured minimum and maximum capacity boundaries
   
   **B. Interval Adjustment**
   - Creation intervals adapt based on pool idle connection count
   - When idle connections are low (<20% of capacity), intervals decrease toward min interval
   - When idle connections are high (>80% of capacity), intervals increase toward max interval
   - Prevents overwhelming network resources during periods of low demand
   - Accelerates connection creation during high demand periods when pool is depleting

## Master API Architecture

In master mode, NodePass provides a RESTful API for centralized management:

### API Components
1. **HTTP/HTTPS Server**:
   - Listens on configured address and port
   - Optional TLS encryption with same modes as tunnel server
   - Configurable API prefix path

2. **Instance Management**:
   - In-memory registry of NodePass instances
   - UID-based instance identification
   - State tracking for each instance (running, stopped, etc.)

3. **RESTful Endpoints**:
   - Standard CRUD operations for instances
   - Instance control actions (start, stop, restart)
   - Health status reporting
   - OpenAPI specification for API documentation

### Instance Lifecycle Management
1. **Instance Creation**:
   - URL-based configuration similar to command line
   - Dynamic initialization based on instance type
   - Parameter validation before instance creation

2. **Instance Control**:
   - Start/stop/restart capabilities
   - Graceful shutdown with configurable timeout
   - Resource cleanup on termination

3. **API Security**:
   - TLS encryption options for API connections
   - Same security modes as tunnel server
   - Certificate management for HTTPS

## Next Steps

- For practical examples of deploying NodePass, see the [examples page](/docs/en/examples.md)
- To fine-tune NodePass for your specific needs, explore the [configuration options](/docs/en/configuration.md)
- If you encounter any issues, check the [troubleshooting guide](/docs/en/troubleshooting.md)