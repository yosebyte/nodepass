# Contributing to NodePass

Thank you for your interest in contributing to NodePass! We welcome all kinds of contributions, from bug reports and feature requests to code improvements and documentation updates.

## Table of Contents

- [Organization](#organization)
- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Guidelines](#contributing-guidelines)
- [Code Style and Standards](#code-style-and-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Community and Support](#community-and-support)

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md). We are committed to providing a welcoming and inclusive environment for all contributors.

## Getting Started

### Prerequisites

- **Go** (as specified in `go.mod`)
- **Git** for version control
- **Docker** (optional, for container-based development and testing)
- Basic knowledge of TCP/UDP networking concepts
- Familiarity with TLS/SSL concepts for security features

### Understanding the Architecture

NodePass is built on a three-tier architecture:

- **Server Mode**: Accepts incoming tunnel connections with configurable security
- **Client Mode**: Establishes outbound connections to tunnel servers  
- **Master Mode**: Provides RESTful API for dynamic instance management

Key components:
- `/cmd/nodepass/`: Main application entry point and core dispatch logic
- `/internal/`: Core implementation packages (server, client, master, common utilities)
- `/docs/`: Comprehensive documentation in English and Chinese
- External dependencies: NodePassProject ecosystem libraries for certificates, connections, logging, and pooling

## Development Setup

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/nodepass.git
cd nodepass

# Add the upstream repository
git remote add upstream https://github.com/yosebyte/nodepass.git
```

### 2. Install Dependencies

```bash
# Download and install dependencies
go mod download

# Verify dependencies
go mod verify
```

### 3. Build and Test

```bash
# Build the application
go build -o nodepass ./cmd/nodepass

# Test the build
./nodepass "server://localhost:10101/127.0.0.1:8080?log=debug&tls=0"
```

### 4. Development with Docker (Optional)

```bash
# Build development container
docker build --build-arg VERSION=dev -t nodepass:dev .

# Run in container
docker run --rm -p 10101:10101 nodepass:dev "server://:10101/127.0.0.1:8080?log=debug&tls=0"
```

## Contributing Guidelines

### Types of Contributions

- **🐛 Bug Reports**: Help us identify and fix issues
- **✨ Feature Requests**: Suggest new features or improvements
- **📝 Documentation**: Improve existing docs or add new ones
- **🔧 Code Contributions**: Bug fixes, feature implementations, refactoring
- **🌐 Translations**: Help translate documentation to other languages
- **🧪 Testing**: Add test cases and improve test coverage

### Reporting Issues

When reporting bugs or requesting features, please:

1. **Search existing issues** to avoid duplicates
2. **Use our issue templates** when available
3. **Provide detailed information**:
   - NodePass version and build information
   - Operating system and architecture
   - Network configuration details
   - Complete command-line arguments used
   - Expected vs. actual behavior
   - Relevant log output (use `log=debug` for detailed logs)
   - Steps to reproduce the issue

### Feature Requests

For new features:

1. **Check the roadmap** and existing feature requests
2. **Describe the use case** clearly
3. **Explain the expected behavior**
4. **Consider backwards compatibility**
5. **Discuss implementation approach** if you plan to contribute code

## Code Style and Standards

### Go Code Style

We follow standard Go conventions with project-specific guidelines:

- Write idiomatic Go code following [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for consistent formatting and `go vet` to catch common errors
- Follow the single responsibility principle and prefer composition over inheritance
- Use descriptive variable and function names with proper Go naming conventions
- Include both Chinese and English comments for public APIs (maintaining project tradition)
- Implement proper error handling with context wrapping
- Use the project's logging framework consistently with appropriate log levels
- Protect shared state with mutexes and use channels for goroutine coordination
- Always handle goroutine cleanup with proper defer and recover patterns

### Configuration and CLI

- Use URL-based configuration syntax: `scheme://[password@]host:port/target?param=value`
- Support environment variables for sensitive configuration
- Provide sensible defaults for all optional parameters
- Validate configuration early in the application lifecycle

### Performance Considerations

- Minimize allocations in hot paths
- Use connection pooling for frequent connections
- Implement graceful degradation under load
- Profile memory and CPU usage for critical paths
- Use buffered I/O where appropriate

## Testing

### Testing Strategy

Currently, the project focuses on integration testing through real-world usage scenarios. We welcome contributions to improve test coverage:

#### Manual Testing

1. **Basic Functionality**: Test server, client, and master modes with debug logging
2. **TLS Modes**: Verify all three TLS security levels (0, 1, 2)
3. **Protocol Support**: Test TCP tunneling and UDP forwarding with various applications

#### Future Testing Goals

We encourage contributions in these areas:
- Unit Tests for individual functions and methods
- Integration Tests for component interactions
- Benchmark Tests for performance regression detection
- Fuzzing Tests for security and robustness
- End-to-End Tests for complete workflow validation

### Testing Guidelines

When adding tests:
- Use Go's standard testing package with `*_test.go` naming convention
- Write table-driven tests where applicable
- Include both positive and negative test cases
- Test error conditions and edge cases
- Use meaningful test names that describe the scenario

## Documentation

### Documentation Standards

- **Write in clear, simple English**
- **Include practical examples** for all features
- **Maintain both English and Chinese versions** when possible
- **Use consistent formatting** and structure
- **Test all code examples** to ensure they work

### Documentation Structure

- `README.md`: Project overview and quick start
- `docs/en/`: English documentation
- `docs/zh/`: Chinese documentation
- Inline code comments for complex logic
- API documentation for master mode endpoints

### Contributing to Documentation

1. **Update both language versions** when possible
2. **Test all examples** before submitting
3. **Use proper Markdown formatting**
4. **Include relevant screenshots** for UI components
5. **Cross-reference related documentation**

## Submitting Changes

### Pull Request Process

1. **Create a feature branch** from the latest `main`:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the guidelines above

3. **Test your changes** thoroughly by building the application and running real scenarios

4. **Commit your changes** with descriptive messages using [Conventional Commits](https://www.conventionalcommits.org/) format:
   - `feat`: New feature
   - `fix`: Bug fix  
   - `docs`: Documentation changes
   - `refactor`: Code refactoring
   - `perf`: Performance improvements
   - `test`: Adding or updating tests

5. **Push to your fork** and create a pull request

### Commit Message Guidelines

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Examples:**
- `feat(server): add support for IPv6 addresses`
- `fix(client): resolve connection timeout issues in high-latency networks`
- `docs: update installation guide with Docker instructions`
- `refactor(common): simplify address parsing logic`

### Pull Request Guidelines

**Before submitting:**
- [ ] Code follows the project style guidelines
- [ ] All tests pass (or explain why they should be skipped)
- [ ] Documentation is updated if needed
- [ ] Commit messages follow the conventional format
- [ ] No merge conflicts with the main branch

**In your pull request:**
- [ ] Provide a clear description of changes
- [ ] Reference any related issues
- [ ] Include testing instructions
- [ ] Add screenshots for UI changes
- [ ] List any breaking changes

### Review Process

1. **Automated checks** run on all pull requests
2. **Code review** by project maintainers
3. **Testing** in various environments
4. **Documentation review** for user-facing changes
5. **Final approval** and merge

## Community and Support

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **Telegram Channel**: [@NodePassChannel](https://t.me/NodePassChannel) - Updates and announcements
- **Telegram Group**: [@NodePassGroup](https://t.me/NodePassGroup) - Community discussion
- **Discord**: [Join our server](https://discord.gg/2cnXcnDMGc) - Real-time chat

### Getting Help

If you need help:

1. **Check the documentation** in the `docs/` directory
2. **Search existing issues** for similar problems
3. **Ask in our community channels** for general questions
4. **Create a GitHub issue** for bugs or feature requests

### Recognition

We appreciate all contributions! Contributors will be:

- **Listed in our contributors** section
- **Mentioned in release notes** for significant contributions
- **Invited to become maintainers** for consistent, high-quality contributions

### Maintainer Responsibilities

Current maintainers handle:

- **Code review** and pull request management
- **Release planning** and version management
- **Community management** and support
- **Security** issue handling
- **Roadmap** planning and prioritization

---

Thank you for contributing to NodePass! Your contributions help make universal TCP/UDP tunneling more accessible and reliable for everyone.

For questions about contributing, please reach out through our community channels or create a GitHub issue.
