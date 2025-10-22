# Troubleshooting Guide

This guide helps you diagnose and resolve common issues you might encounter when using NodePass. For each problem, we provide possible causes and step-by-step solutions.

## Connection Issues

### Unable to Establish Tunnel Connection

**Symptoms**: Client cannot connect to the server's tunnel endpoint, or connection is immediately dropped.

**Possible Causes and Solutions**:

1. **Network Connectivity Issues**
   - Verify basic connectivity with `ping` or `telnet` to the server address
   - Check if the specified port is reachable: `telnet server.example.com 10101`
   - Ensure no firewall is blocking the tunnel port (typically 10101)

2. **Server Not Running**
   - Verify the NodePass server is running with `ps aux | grep nodepass` on Linux/macOS
   - Check server logs for any startup errors
   - Try restarting the server process

3. **Incorrect Addressing**
   - Double-check the tunnel address format in your client command
   - Ensure you're using the correct hostname/IP and port
   - If using DNS names, verify they resolve to the correct IP addresses

4. **TLS Configuration Mismatch**
   - If server requires TLS but client doesn't support it, connection will fail
   - Check server logs for TLS handshake errors
   - Ensure certificates are correctly configured if using TLS mode 2

### Data Not Flowing Through the Tunnel

**Symptoms**: Tunnel connection established, but application data isn't reaching the destination.

**Possible Causes and Solutions**:

1. **Target Service Not Running**
   - Verify the target service is running on both server and client sides
   - Check if you can connect directly to the service locally

2. **Port Conflicts**
   - Ensure the target port isn't already in use by another application
   - Use `netstat -tuln` to check for port usage

3. **Protocol Mismatch**
   - Verify you're tunneling the correct protocol (TCP vs UDP)
   - Some applications require specific protocol support

4. **Incorrect Target Address**
   - Double-check the target address in both server and client commands
   - For server-side targets, ensure they're reachable from the server
   - For client-side targets, ensure they're reachable from the client

### Connection Stability Issues

**Symptoms**: Tunnel works initially but disconnects frequently or becomes unresponsive.

**Possible Causes and Solutions**:

1. **Network Instability**
   - Check for packet loss or high latency in your network
   - Consider a more stable network connection for production deployments

2. **Resource Constraints**
   - Monitor CPU and memory usage on both client and server
   - Adjust pool parameters if resources are being exhausted (see Performance section)
   - Check file descriptor limits with `ulimit -n` on Linux/macOS

3. **Timeout Configuration**
   - Adjust `NP_UDP_DIAL_TIMEOUT` if using UDP with slow response times
   - Increase `read` parameter in URL for long-running transfers (default: 10m)
   - Consider adjusting `NP_TCP_DIAL_TIMEOUT` for unstable network conditions

4. **Overloaded Server**
   - Check server logs for signs of connection overload
   - Adjust `max` parameter and `NP_SEMAPHORE_LIMIT` to handle the load
   - Consider scaling horizontally with multiple NodePass instances

## STUN NAT Traversal Issues

### STUN Discovery Fails

**Symptoms**: Client cannot discover external endpoint, or hybrid mode fails with STUN errors.

**Possible Causes and Solutions**:

1. **STUN Server Unreachable**
   - Verify the STUN server address is correct and accessible
   - Test connectivity: `nc -u stun.l.google.com 19302`
   - Try alternative STUN servers (stun1-4.l.google.com:19302)
   - Check if UDP traffic is blocked by firewall or ISP

2. **Invalid STUN Response**
   - Ensure the server at the tunnel address is a valid STUN server
   - Google's STUN servers: stun.l.google.com, stun1-4.l.google.com (port 19302)
   - STUN response must contain XOR-MAPPED-ADDRESS attribute
   - Check for network middleware interfering with UDP packets

3. **UDP Port Blocking**
   - Verify UDP traffic is not blocked by local firewall
   - Check router/firewall settings for UDP restrictions
   - Some corporate networks block all UDP traffic
   - Try using a different network (mobile hotspot) for testing

4. **Timeout Issues**
   - STUN discovery uses `NP_UDP_READ_TIMEOUT` (default: 3s)
   - Increase timeout if network latency is high
   - Check for packet loss affecting STUN request/response

### NAT Prevents External Connections

**Symptoms**: STUN discovers external endpoint but external clients cannot connect.

**Possible Causes and Solutions**:

1. **Symmetric NAT**
   - Symmetric NAT creates different mappings for different remote endpoints
   - STUN can discover the endpoint but connections from other IPs fail
   - Check NAT type: use online NAT detection tools
   - Consider using dual-end handshake mode (mode=2) with a server instead

2. **Port Restricted NAT**
   - NAT only allows connections from IPs that the client has contacted first
   - This is a security feature of many consumer routers
   - Solution: Have both peers perform STUN discovery and exchange endpoints
   - Or use server relay mode (mode=2) for guaranteed connectivity

3. **Firewall Rules**
   - Local firewall may block incoming connections even with STUN
   - Check iptables/Windows Firewall rules for the discovered port
   - Allow incoming UDP/TCP traffic on the mapped port
   - Verify with: `netstat -tuln | grep <port>`

4. **Dynamic Port Mapping**
   - Some NATs change port mapping after period of inactivity
   - The discovered endpoint may become invalid
   - Implement periodic keepalive traffic to maintain mapping
   - Consider restarting NodePass to rediscover endpoint

### Hybrid Mode Not Activating

**Symptoms**: Client in mode=1 doesn't fallback to STUN when local binding fails.

**Possible Causes and Solutions**:

1. **Port Already in Use**
   - If local port is in use, binding fails as expected
   - Check: `netstat -tuln | grep <port>`
   - Release the port or choose a different tunnel address
   - Hybrid mode should activate automatically after binding failure

2. **Permission Issues**
   - Binding to ports <1024 requires root/administrator privileges
   - Use higher port numbers (>1024) for regular users
   - Or run NodePass with elevated privileges: `sudo nodepass ...`

3. **Configuration Error**
   - Ensure tunnel address points to a valid STUN server
   - Format: `client://stun.l.google.com:19302/target:port?mode=1`
   - Check logs for STUN-related error messages
   - Verify mode is set to 1 or 0 (auto) for hybrid mode support

4. **Network Interface Issues**
   - Binding may fail due to network interface not available
   - Check network connectivity: `ip addr` or `ifconfig`
   - Ensure the system has an active network connection
   - Try binding to 0.0.0.0 instead of specific interface

## Certificate Issues

### TLS Handshake Failures

**Symptoms**: Connection attempts fail with TLS handshake errors.

**Possible Causes and Solutions**:

1. **Invalid Certificate**
   - Verify certificate validity: `openssl x509 -in cert.pem -text -noout`
   - Ensure the certificate hasn't expired
   - Check that the certificate is issued for the correct domain/IP

2. **Missing or Inaccessible Certificate Files**
   - Confirm file paths to certificates and keys are correct
   - Verify file permissions allow the NodePass process to read them
   - Check for file corruption by opening certificates in a text editor

3. **Certificate Trust Issues**
   - If using custom CAs, ensure they are properly trusted
   - For self-signed certificates, confirm TLS mode 1 is being used
   - For verified certificates, ensure the CA chain is complete

4. **Key Format Problems**
   - Ensure private keys are in the correct format (usually PEM)
   - Check for passphrase protection on private keys (not supported directly)

### Certificate Renewal Issues

**Symptoms**: After certificate renewal, secure connections start failing.

**Possible Causes and Solutions**:

1. **New Certificate Not Loaded**
   - Restart NodePass to force loading of new certificates
   - Check if `RELOAD_INTERVAL` is set correctly to automatically detect changes

2. **Certificate Chain Incomplete**
   - Ensure the full certificate chain is included in the certificate file
   - Verify chain order: your certificate first, then intermediate certificates

3. **Key Mismatch**
   - Verify the new certificate matches the private key:
     ```bash
     openssl x509 -noout -modulus -in cert.pem | openssl md5
     openssl rsa -noout -modulus -in key.pem | openssl md5
     ```
   - If outputs differ, certificate and key don't match

## Performance Optimization

### High Latency

**Symptoms**: Connections work but have noticeable delays.

**Possible Causes and Solutions**:

1. **Pool Configuration**
   - Increase `min` parameter to have more connections ready
   - Decrease `MIN_POOL_INTERVAL` to create connections faster
   - Adjust `NP_SEMAPHORE_LIMIT` if connection queue is backing up

2. **Network Path**
   - Check for network congestion or high-latency links
   - Consider deploying NodePass closer to either the client or server
   - Use a traceroute to identify potential bottlenecks

3. **TLS Overhead**
   - If extreme low latency is required and security is less critical, consider using TLS mode 0
   - For a balance, use TLS mode 1 with session resumption

4. **Resource Contention**
   - Ensure the host system has adequate CPU and memory
   - Check for other processes competing for resources
   - Consider dedicated hosts for high-traffic deployments

### High CPU Usage

**Symptoms**: NodePass process consuming excessive CPU resources.

**Possible Causes and Solutions**:

1. **Pool Thrashing**
   - If pool is constantly creating and destroying connections, adjust timings
   - Increase `MIN_POOL_INTERVAL` to reduce connection creation frequency
   - Find a good balance for `min` and `max` pool parameters

2. **Excessive Logging**
   - Reduce log level from debug to info or warn for production use
   - Check if logs are being written to a slow device

3. **TLS Overhead**
   - TLS handshakes are CPU-intensive; consider session caching
   - Use TLS mode 1 instead of mode 2 if certificate validation is less critical

4. **Traffic Volume**
   - High throughput can cause CPU saturation
   - Consider distributing traffic across multiple NodePass instances
   - Vertical scaling (more CPU cores) may be necessary for very high throughput

### Memory Leaks

**Symptoms**: NodePass memory usage grows continuously over time.

**Possible Causes and Solutions**:

1. **Connection Leaks**
   - Ensure `NP_SHUTDOWN_TIMEOUT` is sufficient to properly close connections
   - Check for proper error handling in custom scripts or management code
   - Monitor connection counts with system tools like `netstat`

2. **Pool Size Issues**
   - If `max` parameter is very large, memory usage will be higher
   - Monitor actual pool usage vs. configured capacity
   - Adjust capacity based on actual concurrent connection needs

3. **Debug Logging**
   - Extensive debug logging can consume memory in high-traffic scenarios
   - Use appropriate log levels for production environments

## UDP-Specific Issues

### UDP Data Loss

**Symptoms**: UDP packets are not reliably forwarded through the tunnel.

**Possible Causes and Solutions**:

1. **Buffer Size Limitations**
   - If UDP packets are large, increase `UDP_DATA_BUF_SIZE`
   - Default of 8192 bytes may be too small for some applications

2. **Timeout Issues**
   - If responses are slow, increase `NP_UDP_DIAL_TIMEOUT`
   - Adjust `read` parameter for longer session timeouts
   - For applications with variable response times, find an optimal balance

3. **High Packet Rate**
   - UDP is handled one datagram at a time; very high rates may cause issues
   - Consider increasing pool capacity for high-traffic UDP applications

4. **Protocol Expectations**
   - Some UDP applications expect specific behavior regarding packet order or timing
   - NodePass provides best-effort forwarding but cannot guarantee UDP properties beyond what the network provides

### UDP Connection Tracking

**Symptoms**: UDP sessions disconnect prematurely or fail to establish.

**Possible Causes and Solutions**:

1. **Connection Mapping**
   - Verify client configurations match server expectations
   - Check for firewalls that may be timing out UDP session tracking

2. **Application UDP Timeout**
   - Some applications have built-in UDP session timeouts
   - May need to adjust application-specific keepalive settings

## Master API Issues

### API Accessibility Problems

**Symptoms**: Cannot connect to the master API endpoint.

**Possible Causes and Solutions**:

1. **Endpoint Configuration**
   - Verify API address and port in the master command
   - Check if the API server is bound to the correct network interface

2. **TLS Configuration**
   - If using HTTPS (TLS modes 1 or 2), ensure client tools support TLS
   - For testing, use `curl -k` to skip certificate validation

3. **Custom Prefix Issues**
   - If using a custom API prefix, ensure it's included in all requests
   - Check URL formatting in API clients and scripts

### Instance Management Failures

**Symptoms**: Cannot create, control, or delete instances through the API.

**Possible Causes and Solutions**:

1. **JSON Format Issues**
   - Verify request body is valid JSON
   - Check for required fields in API requests

2. **URL Parsing Problems**
   - Ensure instance URLs are properly formatted and URL-encoded if necessary
   - Verify URL parameters use the correct format

3. **Instance State Conflicts**
   - Cannot delete running instances without stopping them first
   - Check current instance state with GET before performing actions

4. **Permission Issues**
   - Ensure the NodePass master has sufficient permissions to create processes
   - Check file system permissions for any referenced certificates or keys

## Data Recovery

### Master State File Corruption

**Symptoms**: Master mode fails to start showing state file corruption errors, or instance data is lost.

**Possible Causes and Solutions**:

1. **Recovery using automatic backup file**
   - NodePass automatically creates backup file `nodepass.gob.backup` every hour
   - Stop the NodePass master service
   - Copy backup file as main file: `cp nodepass.gob.backup nodepass.gob`
   - Restart the master service

2. **Manual state file recovery**
   ```bash
   # Stop NodePass service
   pkill nodepass
   
   # Backup corrupted file (optional)
   mv nodepass.gob nodepass.gob.corrupted
   
   # Use backup file
   cp nodepass.gob.backup nodepass.gob
   
   # Restart service
   nodepass "master://0.0.0.0:9090?log=info"
   ```

3. **When backup file is also corrupted**
   - Remove corrupted state files: `rm nodepass.gob*`
   - Restart master, which will create new state file
   - Need to reconfigure all instances and settings

4. **Preventive backup recommendations**
   - Regularly backup `nodepass.gob` to external storage
   - Adjust backup frequency: set environment variable `export NP_RELOAD_INTERVAL=30m`
   - Monitor state file size, abnormal growth may indicate issues

**Best Practices**:
- In production environments, recommend regularly backing up `nodepass.gob` to different storage locations
- Use configuration management tools to save text-form backups of instance configurations

## Next Steps

If you encounter issues not covered in this guide:

- Check the [project repository](https://github.com/yosebyte/nodepass) for known issues
- Increase the log level to `debug` for more detailed information
- Review the [How It Works](/docs/en/how-it-works.md) section to better understand internal mechanisms
- Consider joining the community discussion for assistance from other users