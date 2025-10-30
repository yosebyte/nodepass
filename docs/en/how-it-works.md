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
   - **Handshake Phase**: After server validates the tunnel key, it delivers configuration to client:
     - Data flow direction mode (determines whether client receives or sends traffic)
     - Maximum connection pool capacity (centrally managed and allocated by server)
     - TLS security level (ensures client uses correct encryption mode)
   - Listens for signals indicating incoming connections
   - Creates data connections using the TLS security level specified by the server
   - Forwards data between the secure channel and local target
   - Supports bidirectional data flow: data flow direction is automatically selected based on target address

5. **Client Single-End Forwarding Mode**:
   - Automatically enabled when tunnel address is a local address (e.g., 127.0.0.1)
   - Client directly listens on local port without server control channel coordination
   - Uses direct connection establishment for both TCP and UDP protocols
   - Suitable for pure local forwarding scenarios, reducing network overhead and latency
   - Supports high-performance single-end forwarding with optimized connection handling

5. **Protocol Support**:
   - **TCP**: Full bidirectional streaming with persistent connections, optimized for direct connection establishment in client single-end forwarding mode
   - **UDP**: Datagram forwarding with configurable buffer sizes and timeouts

## Data Transmission Flow

NodePass establishes a bidirectional data flow through its tunnel architecture, supporting both TCP and UDP protocols. The system supports three data flow modes:

### Data Flow Mode Explanation
- **Server Receives Mode**: Server listens on target address, client listens locally, data flows from target address to client local
- **Server Sends Mode**: Server connects to remote target address, client listens locally, data flows from client local to remote target
- **Client Single-End Forwarding Mode**: Client directly listens locally and forwards to target address without server coordination, using direct connection establishment for optimized forwarding

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

3. **Direct Connection Establishment**:
   ```
   [Client] → [Create Direct Connection to Target Address] → [Establish Target Connection]
   ```
   - For TCP: Directly establishes TCP connection to target address for each tunnel connection
   - For UDP: Creates UDP socket for datagram exchange with target address
   - Eliminates connection pool overhead, providing simpler and more direct forwarding path

4. **Optimized Forwarding**:
   ```
   [Local Connection] → [Direct Target Connection] → [Data Exchange] → [Connection Cleanup]
   ```
   - For TCP: Direct connection establishment followed by efficient data exchange
   - For UDP: Direct datagram forwarding to target address with minimal latency
   - Simplified data path ensuring reliable and efficient forwarding

### Specific protocol characteristics
- **TCP Exchange**: 
  - Persistent connections for full-duplex communication
  - Continuous data streaming until connection termination
  - Automatic error handling with reconnection capability
  - **Client Single-End Forwarding Optimization**: Direct connection establishment for each tunnel connection, ensuring reliable and efficient forwarding

- **UDP Exchange**:
  - One-shot datagram forwarding with configurable buffer sizes (`UDP_DATA_BUF_SIZE`)
  - Read timeout control for response waiting (`read` parameter or default 0)
  - Optimized for low-latency, stateless communication
  - **Client Single-End Forwarding Optimization**: Direct forwarding mechanism with minimal latency

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

NodePass implements an efficient connection pooling system for managing network connections, which forms the core of its performance advantages:

### Design Philosophy
The connection pool design follows the principle of "warm-up over cold start," eliminating network latency through pre-established connections. This design philosophy draws from modern high-performance server best practices, amortizing the cost of connection establishment to the system startup phase rather than bearing this overhead on the critical path.

### Pool Design
1. **Pool Types**:
   - **Client Pool**: Pre-establishes connections to the remote endpoint with active connection management
   - **Server Pool**: Manages incoming connections from clients with passive connection acceptance

2. **Pool Components**:
   - **Connection Storage**: Thread-safe map of connection IDs to net.Conn objects, supporting high-concurrency access
   - **ID Channel**: Buffered channel for available connection IDs, enabling lock-free rapid allocation
   - **Capacity Management**: Dynamic adjustment based on usage patterns, implementing intelligent scaling
     - Minimum capacity set by client, ensuring basic connection guarantee for client
     - Maximum capacity delivered by server during handshake, enabling global resource coordination
   - **Interval Control**: Time-based throttling between connection creations, preventing network resource overload
   - **Connection Factory**: Customizable connection creation function, supporting different TLS modes and network configurations

### Advanced Design Features
1. **Zero-Latency Connections**:
   - Pre-established connection pools eliminate TCP three-way handshake delays
   - TLS handshakes complete during connection pool initialization, avoiding runtime encryption negotiation overhead
   - Connection warm-up strategies ensure hot connections are always available in the pool

2. **Intelligent Load Awareness**:
   - Dynamic pool management based on real-time connection utilization
   - Predictive connection creation based on historical usage patterns
   - Adaptive timeout and retry mechanisms responding to network fluctuations

### Connection Lifecycle
1. **Connection Creation**:
   - Connections are created up to the configured capacity, ensuring resource controllability
   - Each connection is assigned a unique ID, supporting precise connection tracking and management
   - IDs and connections are stored in the pool with copy-on-write and delayed deletion strategies

2. **Connection Acquisition**:
   - Client retrieves connections using connection IDs, supporting precise matching and fast lookups
   - Server retrieves the next available connection from the pool using round-robin or least-used strategies
   - Connections are validated before being returned, including network status and TLS session checks

3. **Connection Usage**:
   - Connection is removed from the pool when acquired, avoiding reuse conflicts
   - Used for data exchange between endpoints with efficient zero-copy transmission
   - One-time use model ensures connection state cleanliness

4. **Connection Termination**:
   - Connections are closed immediately after use, preventing resource leaks
   - Proper release of system resources including file descriptors and memory buffers
   - Error handling ensures clean termination under exceptional conditions

### Session Management and State Maintenance
1. **Stateful UDP Processing**:
   - Converts stateless UDP protocol into stateful session handling
   - Intelligent session timeout management, balancing resource usage and responsiveness
   - Session reuse mechanisms, reducing connection establishment overhead

2. **TCP Connection Management**:
   - Connection pool management for efficient resource utilization
   - One-time use model for connection pool entries to ensure state cleanliness
   - Connection health monitoring and automatic cleanup

3. **Cross-Protocol Unified Management**:
   - Unified connection lifecycle management, simplifying system complexity
   - Protocol-agnostic monitoring and statistics, providing consistent observability experience
   - Flexible protocol conversion capabilities, supporting heterogeneous network environments

## Signal Communication and Coordination Mechanisms

NodePass's signaling system embodies the essence of distributed system design:

### Signal Design Principles
1. **Event-Driven Architecture**:
   - Event-based asynchronous communication patterns, avoiding blocking waits
   - Publish-subscribe pattern for signal distribution, supporting multiple subscribers
   - Signal priority management, ensuring timely processing of critical events

2. **Reliability Guarantees**:
   - Signal persistence mechanisms, preventing critical signal loss
   - Retry and acknowledgment mechanisms, ensuring reliable signal delivery
   - Idempotent signal design, avoiding side effects from repeated execution

3. **Performance Optimization**:
   - Batch signal processing, reducing system call overhead
   - Signal compression and merging, optimizing network bandwidth usage
   - Asynchronous signal processing, avoiding blocking of main processing flows

### Distributed Coordination
1. **Consistency Guarantees**:
   - Distributed locking mechanisms, ensuring atomicity of critical operations
   - State synchronization protocols, maintaining data consistency across multiple nodes
   - Conflict resolution strategies, handling race conditions in concurrent operations

2. **Fault Handling**:
   - Node failure detection, timely discovery and isolation of failed nodes
   - Automatic failover, ensuring service continuity
   - State recovery mechanisms, supporting rapid recovery after failures

### Pool Management
1. **Capacity Control**:
   - Minimum capacity guarantee: Ensures sufficient warm connections are always available
   - Maximum capacity limit: Prevents excessive resource consumption, protecting system stability
   - Dynamic scaling based on demand patterns, responding to traffic changes

2. **Interval Control**:
   - Minimum interval limit: Prevents connection creation storms, protecting network resources
   - Maximum interval limit: Ensures timely response to connection demands
   - Adaptive time-based throttling to optimize resource usage

3. **Dynamic Pool Adaptation**:
   The connection pool employs a dual-adaptive mechanism to ensure optimal performance:
   
   **A. Capacity Adjustment**
   - Pool capacity dynamically adjusts based on real-time usage patterns, implementing intelligent scaling
   - Feedback adjustment based on connection creation success rate: contracts capacity during low success rates to reduce resource waste
   - Expands capacity during high success rates to meet growing demands
   - Gradual scaling prevents system oscillation, providing smooth performance transitions
   - Strictly respects configured capacity boundaries, ensuring system controllability
   
   **B. Interval Adjustment**
   - Creation intervals adapt based on pool idle connection count in real-time
   - Accelerates connection creation during low idle rates, ensuring adequate supply
   - Slows creation pace during high idle rates, avoiding resource waste
   - Prevents pressure on network resources during low-demand periods
   - Accelerates connection creation during high-demand periods when pool is depleting, ensuring service quality

4. **Performance Optimization Strategies**:
   - **Predictive Scaling**: Forecasts future demands based on historical usage patterns
   - **Tiered Connection Management**: Different priority connections use different management strategies
   - **Batch Operation Optimization**: Bulk creation and destruction of connections, reducing system call overhead
   - **Connection Affinity**: Intelligent connection allocation based on geographic location or network topology

## Data Exchange Mechanisms

NodePass's data exchange mechanisms embody modern network programming best practices:

### High-Performance Data Transfer
1. **Zero-Copy Architecture**:
   - Data transfers directly in kernel space, avoiding multiple copies in user space
   - Reduces CPU overhead and memory bandwidth consumption
   - Supports optimized transmission for large files and high-throughput scenarios

2. **Asynchronous I/O Model**:
   - Non-blocking event-driven architecture maximizes concurrent processing capabilities
   - Efficient event loops based on epoll/kqueue
   - Intelligent read/write buffer management, balancing memory usage and performance

3. **Traffic Statistics and Monitoring**:
   - Real-time byte-level traffic statistics, supporting precise bandwidth control
   - Protocol-specific traffic analysis, facilitating performance tuning
   - Connection-level performance metrics, supporting fine-grained monitoring
   - Real-time tracking of active TCP and UDP connection counts for capacity planning and performance analysis

### Protocol Optimization
1. **TCP Optimization**:
   - Intelligent TCP_NODELAY configuration, reducing small packet delays
   - Keep-alive mechanisms ensure long connection reliability
   - Adaptive selection of congestion control algorithms

2. **UDP Optimization**:
   - Session-based UDP processing, supporting stateful datagram exchange
   - Intelligent timeout management, balancing responsiveness and resource usage
   - Datagram deduplication and out-of-order processing

## Master API Architecture

In master mode, NodePass provides a RESTful API for centralized management, embodying cloud-native architectural design principles:

### Architectural Design Philosophy
Master mode adopts a "unified management, distributed execution" architecture pattern, separating the control plane from the data plane. This design gives the system enterprise-grade manageability and observability while maintaining high-performance data transmission.

### API Components
1. **HTTP/HTTPS Server**:
   - Listens on configured address and port, supporting flexible network deployment
   - Optional TLS encryption with same security modes as tunnel server, ensuring management channel security
   - Configurable API prefix path, supporting reverse proxy and API gateway integration

2. **Instance Management**:
   - High-performance memory-based instance registry, supporting fast queries and updates
   - UID-based instance identification, ensuring global uniqueness
   - State tracking for each instance (running, stopped, etc.), supporting real-time status monitoring

3. **RESTful Endpoints**:
   - Standard CRUD operations following REST design principles
   - Instance control actions (start, stop, restart), supporting remote lifecycle management
   - Health status reporting, providing real-time system health information
   - OpenAPI specification support, facilitating API documentation generation and client development

### Instance Lifecycle Management
1. **Instance Creation**:
   - URL-based configuration similar to command line, reducing learning curve
   - Dynamic initialization based on instance type, supporting multiple deployment modes
   - Parameter validation before instance creation, ensuring configuration correctness

2. **Instance Control**:
   - Start/stop/restart capabilities, supporting remote operations
   - Graceful shutdown with configurable timeout, ensuring data integrity
   - Resource cleanup on termination, preventing resource leaks

3. **API Security**:
   - TLS encryption options for API connections, protecting management communication security
   - Same security modes as tunnel server, unified security policies
   - Certificate management support, simplifying HTTPS deployment

## System Architecture Advancement

### Layered Decoupling Design
NodePass adopts layered design principles of modern software architecture:

1. **Transport Layer Separation**:
   - Complete separation of control and data channels, avoiding control information interference with data transmission
   - Independent optimization for different protocols, TCP and UDP each using optimal strategies
   - Multiplexing support, single tunnel carrying multiple application connections

2. **Pluggable Security Layer**:
   - Modular TLS implementation, supporting flexible selection of different security levels
   - Automated certificate management, reducing operational complexity
   - Key rotation mechanisms, enhancing long-term security

3. **Cloud-Native Management Layer**:
   - API-first design philosophy, all functions accessible through APIs
   - Container-friendly configuration methods, supporting modern DevOps practices
   - Stateless design, facilitating horizontal scaling

### Performance Optimization Philosophy
1. **Latency Optimization**:
   - Pre-connection pools eliminate cold start latency
   - Intelligent routing reduces network hops
   - Batch processing reduces system call overhead

2. **Throughput Optimization**:
   - Zero-copy data transmission maximizes bandwidth utilization
   - Concurrent connection management supports high-concurrency scenarios
   - Adaptive buffer sizing optimizes memory usage

3. **Resource Optimization**:
   - Intelligent connection reuse reduces resource consumption
   - Dynamic capacity adjustment adapts to load changes
   - Garbage collection optimization reduces pause times

### Reliability Guarantees
1. **Fault Isolation**:
   - Connection-level fault isolation, single point failures don't affect overall service
   - Automatic reconnection mechanisms, transparently handling network fluctuations
   - Graceful degradation strategies, ensuring core functionality under resource constraints

2. **State Management**:
   - Distributed state synchronization, ensuring consistency across multiple instances
   - Persistence of critical state, supporting failure recovery
   - Versioned configuration management, supporting rollback operations

## NodePass Architecture Innovation Summary

### Technical Innovation Points
1. **Connection Pool Warm-up Technology**:
   - Revolutionarily eliminates cold start latency in network tunnels
   - Transforms traditional "connect-on-demand" to "pre-warm-and-ready"
   - Significantly improves first connection response speed

2. **Separated Architecture Design**:
   - Complete separation of control plane and data plane
   - Independent optimization of signaling and data channels
   - Achieves perfect combination of high performance and high manageability

3. **Adaptive Resource Management**:
   - Intelligent scaling based on real-time load
   - Predictive resource allocation strategies
   - Self-healing resilient system design

### Industry-Leading Advantages
1. **Performance Advantages**:
   - Zero-latency connection establishment, industry-leading response speed
   - High concurrency processing capabilities, supporting enterprise-grade application scenarios
   - Intelligent routing optimization, shortest path data transmission

2. **Reliability Advantages**:
   - Multi-layer fault isolation and recovery mechanisms
   - High availability guarantees of distributed architecture
   - Graceful degradation service quality assurance

3. **Security Advantages**:
   - End-to-end encryption protection
   - Multi-layer security protection system
   - Compliance with enterprise-grade security standards

### Applicable Scenarios and Value
1. **Enterprise Applications**:
   - Service mesh for microservice architectures
   - Network connections in hybrid cloud environments
   - Cross-regional service access

2. **Development and Operations**:
   - Rapid setup of local development environments
   - Flexible configuration of test environments
   - Traffic management in production environments

3. **Network Optimization**:
   - Significant reduction in network latency
   - Notable improvement in bandwidth utilization
   - Reliable guarantee of connection stability

NodePass, through its innovative architectural design and technical implementation, provides a high-performance, high-reliability, high-security tunnel solution for modern network applications, representing the future direction of network tunnel technology.

## Next Steps

- For practical examples of deploying NodePass, see the [examples page](/docs/en/examples.md)
- To fine-tune NodePass for your specific needs, explore the [configuration options](/docs/en/configuration.md)
- If you encounter any issues, check the [troubleshooting guide](/docs/en/troubleshooting.md)