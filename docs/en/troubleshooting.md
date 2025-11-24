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
   - Increase `read` parameter in URL for long-running transfers (default: 0)
   - Consider adjusting `NP_TCP_DIAL_TIMEOUT` for unstable network conditions

4. **Overloaded Server**
   - Check server logs for signs of connection overload
   - Adjust `max` parameter and `NP_SEMAPHORE_LIMIT` to handle the load
   - Consider scaling horizontally with multiple NodePass instances

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

## DNS Issues

### DNS Resolution Failures

**Symptoms**: Connections fail with "no such host" or DNS lookup errors.

**Solutions**:

1. **Verify System DNS Configuration**
   - Verify resolution works: `nslookup example.com`
   - Check system's DNS settings (NodePass uses system resolver)
   - Ensure network connectivity is working

2. **Network Connectivity**
   - Check if firewall blocks UDP port 53
   - Verify domain reachability
   - Test with alternative domains to isolate issue

### DNS Caching Problems

**Symptoms**: Resolution returns stale IPs, connections go to wrong endpoints.

**Solutions**:

1. **Adjust Cache TTL** (default 5 minutes)
   - Dynamic environments: `dns=1m`
   - Stable environments: `dns=30m`

2. **Load Balancing Scenarios**
   - Use shorter TTL: `dns=30s`
   - Or use IP addresses directly to bypass DNS caching

### DNS Performance Optimization

**Symptoms**: High connection latency, slow startup.

**Solutions**:

1. **Optimize Cache TTL**
   - Increase TTL for stable environments: `dns=1h`
   - Reduce TTL for dynamic environments: `dns=1m`
   - Balance between freshness and performance

2. **Reduce DNS Queries**
   - Use IP addresses directly for performance-critical scenarios
   - Increase TTL for stable hostnames
   - Pre-resolve addresses when possible

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

## QUIC-Specific Issues

### QUIC Connection Failures

**Symptoms**: QUIC tunnel fails to establish when `quic=1` is enabled.

**Possible Causes and Solutions**:

1. **UDP Port Blocked**
   - Verify UDP port is accessible on both server and client
   - Check firewall rules: `sudo ufw allow 10101/udp` (Linux example)
   - Test UDP connectivity with `nc -u server.example.com 10101`
   - Some ISPs or networks block or throttle UDP traffic

2. **TLS Configuration Issues**
   - QUIC requires TLS to be enabled (minimum `tls=1`)
   - If `quic=1` is set but TLS is disabled, system auto-enables `tls=1`
   - For production, use `tls=2` with valid certificates
   - Check certificate validity for QUIC connections

3. **Client-Server QUIC Mismatch**
   - Both server and client must use same `quic` setting
   - Server with `quic=1` requires client with `quic=1`
   - Server with `quic=0` requires client with `quic=0`
   - Check logs for "QUIC connection not available" errors

4. **Mode Compatibility**
   - QUIC only works in dual-end handshake mode (mode=2)
   - Not available in single-end forwarding mode (mode=1)
   - System will fall back to TCP pool if mode incompatible

### QUIC Performance Issues

**Symptoms**: QUIC tunnel has lower performance than expected or worse than TCP pool.

**Possible Causes and Solutions**:

1. **Network Path Issues**
   - Some networks deprioritize or shape UDP traffic
   - Check if network middleboxes are interfering with QUIC
   - Consider testing with TCP pool (`quic=0`) for comparison
   - Monitor packet loss rates - QUIC performs better with low loss

2. **Pool Capacity Configuration**
   - Increase `min` and `max` parameters for higher throughput
   - QUIC streams share single UDP connection - adequate capacity needed
   - Monitor stream utilization with `log=debug`
   - Balance between stream count and resource usage

3. **Certificate Overhead**
   - TLS 1.3 handshake (mandatory for QUIC) can add initial latency
   - Use 0-RTT resumption for faster reconnection
   - Ensure proper certificate chain to avoid validation delays

4. **Application Compatibility**
   - Some applications may not work optimally over QUIC streams
   - Test with both TCP and QUIC pools to compare performance
   - Consider TCP pool for applications requiring strict ordering

### QUIC Stream Exhaustion

**Symptoms**: "Insufficient streams" errors or connection timeouts when using QUIC.

**Possible Causes and Solutions**:

1. **Pool Capacity Too Low**
   - Increase `max` parameter on server side
   - Increase `min` parameter on client side
   - Monitor active stream count in logs
   - Default capacity may be insufficient for high-concurrency scenarios

2. **Stream Leaks**
   - Check application properly closes connections
   - Monitor stream count over time for gradual increase
   - Restart instances to clear leaked streams
   - Review application code for connection handling

3. **QUIC Connection Dropped**
   - Check keep-alive settings (configured via `NP_REPORT_INTERVAL`)
   - Monitor for "QUIC connection not available" errors
   - NAT timeout may drop UDP connection - adjust NAT settings
   - Increase connection timeout if network latency is high

### QUIC vs TCP Pool Decision

**When to Use QUIC** (`quic=1`):
- Mobile networks or frequently changing network conditions
- High-latency connections (satellite, long-distance)
- NAT-heavy environments where UDP traversal is better
- Real-time applications benefiting from stream independence
- Scenarios where 0-RTT reconnection provides value

**When to Use TCP Pool** (`quic=0`):
- Networks that block or severely throttle UDP traffic
- Applications requiring strict TCP semantics
- Corporate environments with UDP restrictions
- Maximum compatibility requirements
- When testing shows better performance with TCP

**Comparison Testing**:
```bash
# Test TCP pool performance
nodepass "server://0.0.0.0:10101/backend:8080?quic=0&mode=2&log=event"
nodepass "client://server:10101/127.0.0.1:8080?quic=0&mode=2&log=event"

# Test QUIC pool performance
nodepass "server://0.0.0.0:10102/backend:8080?quic=1&mode=2&log=event"
nodepass "client://server:10102/127.0.0.1:8081?quic=1&mode=2&log=event"
```

Monitor traffic statistics and choose based on observed performance.

## Next Steps

If you encounter issues not covered in this guide:

- Check the [project repository](https://github.com/yosebyte/nodepass) for known issues
- Increase the log level to `debug` for more detailed information
- Review the [How It Works](/docs/en/how-it-works.md) section to better understand internal mechanisms
- Consider joining the community discussion for assistance from other users