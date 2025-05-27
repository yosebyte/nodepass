# Installation Guide

This guide provides detailed instructions for installing NodePass using different methods. Choose the option that best suits your environment and requirements.

## System Requirements

- Go 1.24 or higher (for building from source)
- Network connectivity between server and client endpoints
- Admin privileges may be required for binding to ports below 1024

## Installation Options

### Option 1: Pre-built Binaries

The easiest way to get started with NodePass is to download a pre-built binary for your platform.

1. Visit the [releases page](https://github.com/yosebyte/nodepass/releases) on GitHub
2. Download the appropriate binary for your operating system (Windows, macOS, Linux)
3. Extract the archive if necessary
4. Make the binary executable (Linux/macOS):
   ```bash
   chmod +x nodepass
   ```
5. Move the binary to a location in your PATH:
   - Linux/macOS: `sudo mv nodepass /usr/local/bin/`
   - Windows: Add the location to your PATH environment variable

### Option 2: Using Go Install

If you have Go installed on your system, you can use the `go install` command:

```bash
go install github.com/yosebyte/nodepass/cmd/nodepass@latest
```

This command downloads the source code, compiles it, and installs the binary in your Go bin directory (usually `$GOPATH/bin`).

### Option 3: Building from Source

For the latest development version or to customize the build:

```bash
# Clone the repository
git clone https://github.com/yosebyte/nodepass.git

# Navigate to the project directory
cd nodepass

# Build the binary
go build -o nodepass ./cmd/nodepass

# Optional: Install to your GOPATH/bin
go install ./cmd/nodepass
```

### Option 4: Using Container Image

NodePass is available as a container image on GitHub Container Registry, perfect for containerized environments:

```bash
# Pull the container image
docker pull ghcr.io/yosebyte/nodepass:latest

# Run in server mode
docker run -d --name nodepass-server -p 10101:10101 -p 8080:8080 \
  ghcr.io/yosebyte/nodepass server://0.0.0.0:10101/0.0.0.0:8080

# Run in client mode
docker run -d --name nodepass-client \
  -e MIN_POOL_CAPACITY=32 \
  -e MAX_POOL_CAPACITY=512 \
  -p 8080:8080 \
  ghcr.io/yosebyte/nodepass client://nodepass-server:10101/127.0.0.1:8080
```

### Option 5: Using Management Script (Linux Only)

For Linux systems, we provide a one-click script:

```bash
bash <(curl -sSL https://run.nodepass.eu/np.sh)
```

- This script provides easy-to-use master mode (API mode) installation, configuration, and management functions.
- For details, please refer to [https://github.com/NodePassProject/npsh](https://github.com/NodePassProject/npsh)

## Verifying Installation

After installation, verify that NodePass is correctly installed by checking the version:

```bash
nodepass
```

## Next Steps

Now that you have NodePass installed, you can:

- Learn about its basic [usage](/docs/en/usage.md)
- Explore [configuration options](/docs/en/configuration.md)
- Try out some [examples](/docs/en/examples.md)

## Troubleshooting Installation Issues

If you encounter any issues during installation:

- Ensure your system meets the minimum requirements
- Check that you have the correct permissions to install software
- For Go-related issues, verify your Go installation with `go version`
- For container-related issues, ensure Docker is properly installed and running
- See our [troubleshooting guide](/docs/en/troubleshooting.md) for more help
