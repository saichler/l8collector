# L8Collector

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.23.8-blue.svg)](https://golang.org/dl/)

L8Collector is a multi-protocol network data collection service built on the Layer8 ecosystem and Pollaris model. It provides a unified framework for collecting data from various network devices and systems using different protocols including SNMP, SSH, and Kubernetes.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Scalability & Performance](#scalability--performance)
- [Supported Protocols](#supported-protocols)
- [Dependencies](#dependencies)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Testing](#testing)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Overview

L8Collector is designed as a service within the Layer8 ecosystem that enables automated data collection from network infrastructure. It implements a pluggable architecture that supports multiple collection protocols and can be easily extended to support additional protocols.

The collector operates on a device-centric model where each device can have multiple hosts, and each host can be polled using different protocols based on the device configuration.

## Features

- **Multi-Protocol Support**: SNMP v2c, SSH, and Kubernetes data collection
- **Concurrent Collection**: Parallel data collection from multiple devices and hosts
- **Service-Oriented Architecture**: Built as a microservice with Layer8 framework
- **Device Management**: Centralized device and host management
- **Job Queuing**: Efficient job scheduling and execution
- **Remote Job Execution**: ExecuteService for distributed job processing across cluster nodes
- **Parameter Substitution**: Dynamic argument replacement in Kubernetes commands
- **Error Handling**: Robust error handling and connection management
- **String Handling Optimization**: Unified string concatenation using l8utils/strings package
- **Testing Framework**: Comprehensive unit testing with coverage reporting

## Architecture

The L8Collector follows a modular architecture with the following key components:

### Core Components

- **CollectorService**: Main service orchestrating the collection process
- **ExecuteService**: Service for remote job execution and distribution across cluster nodes
- **HostCollector**: Manages collection from individual hosts
- **DeviceCollector**: Handles device-level collection logic
- **JobsQueue**: Manages collection job scheduling and execution
- **DeviceCenter**: Central management for device configurations
- **DeviceService**: Service layer for device operations

### Protocol Collectors

- **SNMPv2Collector**: SNMP version 2c data collection
- **SshCollector**: SSH-based command execution and data collection
- **Kubernetes**: Kubernetes cluster data collection via kubectl

### Interfaces

- **ProtocolCollector**: Common interface for all protocol implementations
- **IResources**: Resource management interface
- **IVNic**: Virtual network interface for Layer8 communication

## Scalability & Performance

L8Collector is designed for enterprise-scale deployments with robust concurrency and resource management:

### Device Capacity

- **Conservative estimate**: 1,000-2,000 devices per collector instance
- **Optimistic estimate**: 5,000-10,000 devices per collector instance
- Scales horizontally using Layer8 service mesh for larger deployments

### Concurrency Model

- **1:1 Host-to-Goroutine mapping**: Each host runs in a dedicated goroutine
- **Per-protocol collectors**: SNMP, SSH, K8s collectors created per host/device
- **Thread-safe collections**: Uses concurrent-safe maps for device management
- **Non-blocking execution**: Jobs run sequentially per host, hosts run in parallel

### Resource Consumption

Per device resource usage:
- **Memory**: ~1-2KB overhead + job queue storage
- **Goroutines**: 1+ per host (typically 1 host per device)
- **Network connections**: 1 per protocol per host
- **CPU**: Based on polling cadence (minimum 3-second intervals)

### Performance Optimizations

- **SNMP Timeout Protection**: 15-second timeout for Entity MIB walks prevents hanging
- **Job Queue Optimization**: Priority-based scheduling with configurable cadences
- **Connection Pooling**: Efficient reuse of protocol connections
- **Resource Limits**: Configurable limits prevent resource exhaustion

### Key Limiting Factors

1. **Network connections**: OS file descriptor limits (typically 65K)
2. **Memory growth**: Job queues and result storage
3. **Polling frequency**: Minimum 3-second cadence prevents overload
4. **SNMP complexity**: Large MIB walks (mitigated with timeouts)

### Horizontal Scaling

The architecture scales horizontally through:
- Multiple collector instances handling different device sets
- Layer8 service mesh for distributed coordination  
- Load balancing across available collector nodes
- Automatic failover and job redistribution

### Code Quality Improvements

The codebase has been optimized for maintainability and performance:

#### String Handling Optimization (Latest Update)
- **Unified String Concatenation**: All string concatenations replaced with `github.com/saichler/l8utils/go/utils/strings.String` package
- **Raw Value Support**: Eliminated unnecessary `strconv.Itoa()` calls in string concatenations as the strings package natively supports raw integer values
- **Performance Benefits**: Reduced memory allocations and improved string building performance
- **Code Consistency**: Standardized approach to string manipulation across all protocol collectors

#### Updated Components
- **HostCollector**: Optimized error message construction and protocol validation
- **SSH Collector**: Enhanced connection error handling and logging with efficient string building
- **SNMP Collector**: Improved error reporting for walk operations and connection issues  
- **Kubernetes Collector**: Streamlined script generation and error message formatting

#### Benefits
- **Reduced Memory Pressure**: More efficient string concatenation reduces garbage collection overhead
- **Improved Readability**: Consistent string handling patterns across the codebase
- **Enhanced Performance**: Native support for various data types eliminates conversion overhead
- **Better Maintainability**: Centralized string handling logic through utility package

## Supported Protocols

### SNMP v2c
- Community-based authentication
- Configurable timeout and retry settings
- OID-based data collection
- Support for SNMP walks and gets

### SSH
- Username/password authentication
- Command execution with prompt detection
- Session management
- Configurable prompts and timeouts

### Kubernetes
- kubectl-based data collection
- Context-aware configuration
- Base64-encoded kubeconfig support
- Cluster resource monitoring
- Dynamic parameter substitution using `$variable` syntax in commands

## Dependencies

### Core Dependencies
- **Go 1.23.8+**: Programming language runtime
- **github.com/gosnmp/gosnmp**: SNMP protocol implementation
- **golang.org/x/crypto/ssh**: SSH client implementation
- **github.com/google/uuid**: UUID generation

### Layer8 Ecosystem Dependencies
- **l8pollaris**: Polling and data modeling framework
- **l8services**: Service management framework
- **l8types**: Common type definitions
- **l8utils**: Utility libraries (including optimized string handling)
- **l8srlz**: Serialization framework
- **l8parser**: Data parsing framework

## Installation

### Prerequisites
- Go 1.23.8 or later
- Git
- Network access to target devices

### Build from Source

```bash
# Clone the repository
git clone https://github.com/saichler/l8collector.git
cd l8collector/go

# Initialize Go modules
go mod init
GOPROXY=direct GOPRIVATE=github.com go mod tidy
go mod vendor

# Build the application
go build ./...
```

### Using the Test Script

```bash
cd go
chmod +x test.sh
./test.sh
```

This script will:
- Clean and reinitialize Go modules
- Fetch dependencies
- Run security checks
- Execute unit tests with coverage
- Generate coverage reports

## Configuration

### Device Configuration

Devices are configured using the Layer8 Pollaris model with the following structure:

```go
type Device struct {
    DeviceId string
    Hosts    []*Host
    // Additional device properties
}

type Connection struct {
    Addr           string    // Device address
    Port           int32     // Connection port
    Protocol       Protocol  // SNMP, SSH, K8s
    Timeout        int32     // Connection timeout
    ReadCommunity  string    // SNMP community (for SNMP)
    Username       string    // SSH username
    Password       string    // SSH password
    KubeConfig     string    // Kubernetes config (base64 encoded)
    KukeContext    string    // Kubernetes context
    Prompt         []string  // SSH prompts
}
```

### Environment Configuration

The service integrates with the Layer8 ecosystem and uses the standard Layer8 configuration mechanisms:

- Service registration and discovery
- Resource management
- Logging configuration
- Network interface management

## Usage

### Service Integration

L8Collector is designed to run as a service within the Layer8 ecosystem:

```go
// Service activation
collectorService := &CollectorService{}
err := collectorService.Activate(serviceName, serviceArea, resources, listener)

// Device polling
device := &types.Device{
    DeviceId: "device-001",
    Hosts: []*types.Host{
        // Host configurations
    },
}

// Start polling
response := collectorService.Post(object.New(device), vnic)
```

### Remote Job Execution

The ExecuteService enables distributed job execution across cluster nodes:

```go
// Execute a job remotely
executeService := &ExecuteService{}
job := &types.CJob{
    DeviceId: "device-001",
    HostId: "host-001",
    Arguments: map[string]string{
        "namespace": "kube-system",
        "resource": "pods",
    },
}

// Submit job for execution
response := executeService.Post(object.New(job), vnic)
```

### Protocol-Specific Usage

#### SNMP Collection
```go
snmpCollector := &SNMPv2Collector{}
snmpCollector.Init(connection, resources)
snmpCollector.Connect()
snmpCollector.Exec(job)
snmpCollector.Disconnect()
```

#### SSH Collection
```go
sshCollector := &SshCollector{}
sshCollector.Init(connection, resources)
sshCollector.Connect()
sshCollector.Exec(job)
sshCollector.Disconnect()
```

#### Kubernetes Collection
```go
k8sCollector := &Kubernetes{}
k8sCollector.Init(connection, resources)

// Job with parameter substitution
job := &types.CJob{
    Arguments: map[string]string{
        "namespace": "default",
        "resource": "pods",
    },
}
// Command: "get pods -n $namespace" becomes "get pods -n default"
k8sCollector.Exec(job)
```

## Testing

### Running Tests

```bash
# Run all tests with coverage
go test -tags=unit -v -coverpkg=./collector/... -coverprofile=cover.html ./... --failfast

# View coverage report
go tool cover -html=cover.html
```

### Test Structure

- **Unit Tests**: Located in `tests/` directory
- **Mock Services**: Mock implementations for testing
- **Coverage Reports**: HTML coverage reports generated
- **Test Utilities**: Helper functions and test setup

### Test Files

- `tests/Collector_test.go`: Main collector tests
- `tests/TestInit.go`: Test initialization
- `tests/utils_collector/`: Test utilities and mocks

## Development

### Project Structure

```
l8collector/
├── LICENSE                 # Apache 2.0 license
├── README.md              # This file
└── go/
    ├── collector/
    │   ├── common/        # Common interfaces and constants
    │   ├── devices/       # Device management
    │   ├── protocols/     # Protocol implementations
    │   │   ├── snmp/     # SNMP v2c collector
    │   │   ├── ssh/      # SSH collector
    │   │   ├── k8s/      # Kubernetes collector
    │   │   └── Utils.go  # Protocol utilities
    │   └── service/       # Core services
    │       ├── CollectorService.go    # Main collection service
    │       ├── ExecuteService.go      # Remote execution service
    │       ├── DeviceCollector.go     # Device-level operations
    │       ├── HostCollector.go       # Host-level operations
    │       └── JobsQueue.go           # Job scheduling
    ├── tests/             # Test files
    ├── go.mod             # Go module definition
    ├── go.sum             # Go module checksums
    ├── test.sh            # Test script
    └── vendor/            # Vendored dependencies
```

### Adding New Protocols

To add a new protocol collector:

1. Implement the `ProtocolCollector` interface:
   ```go
   type ProtocolCollector interface {
       Init(*types.Connection, ifs.IResources) error
       Protocol() types.Protocol
       Exec(*types.CJob)
       Connect() error
       Disconnect() error
   }
   ```

2. Create the protocol-specific package under `collector/protocols/`
3. Register the new protocol type in the Layer8 type system
4. Add configuration support for the new protocol
5. Implement comprehensive tests

### Code Style

- Follow Go best practices and conventions
- Use meaningful variable and function names
- Include proper error handling
- Add appropriate logging
- Document public interfaces
- Maintain test coverage
- **String Handling**: Use `github.com/saichler/l8utils/go/utils/strings.String` for all string concatenations instead of native `+` operator
- **Raw Values**: Leverage the strings package's support for raw integer/numeric values instead of manual conversions

### Building and Testing

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run tests
./test.sh

# Build
go build ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Contribution Guidelines

- Ensure all tests pass
- Add tests for new functionality
- Follow the existing code style
- Update documentation as needed
- Include appropriate error handling

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Links

- [Layer8 Ecosystem](https://github.com/saichler/layer8)
- [L8Pollaris](https://github.com/saichler/l8pollaris)
- [L8Services](https://github.com/saichler/l8services)

---

For questions, issues, or contributions, please use the GitHub issue tracker or submit pull requests.