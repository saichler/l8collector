# L8Collector

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.24.0-blue.svg)](https://golang.org/dl/)

L8Collector is a multi-protocol network data collection service built on the Layer8 ecosystem and Pollaris model. It provides a unified framework for collecting data from various network devices and systems using different protocols including SNMP, SSH, Kubernetes, REST/RESTCONF, and GraphQL.

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

- **Multi-Protocol Support**: SNMP v2c, SSH, Kubernetes, REST/RESTCONF, and GraphQL data collection
- **Concurrent Collection**: Parallel data collection from multiple devices and hosts
- **Service-Oriented Architecture**: Built as a microservice with Layer8 framework and SLA support
- **Target Management**: Dynamic target device configuration with TargetService and TargetCenter
- **Service Level Agreements**: Integrated SLA support for better service lifecycle management
- **Job Queuing**: Efficient job scheduling and execution with cadence management
- **Remote Job Execution**: ExecuteService for distributed job processing across cluster nodes with SLA
- **Parameter Substitution**: Dynamic argument replacement in Kubernetes commands
- **Error Handling**: Robust error handling and connection management
- **String Handling Optimization**: Unified string concatenation using l8utils/strings package
- **Testing Framework**: Comprehensive unit testing with coverage reporting including GraphQL and REST tests
- **Web Interface**: Interactive web interface for monitoring and management

## Architecture

The L8Collector follows a modular architecture with the following key components:

### Core Components

- **CollectorService**: Main service orchestrating the collection process with Service Level Agreement (SLA) support
- **ExecuteService**: Service for remote job execution and distribution across cluster nodes with SLA integration
- **TargetService**: New centralized target device management service with dynamic configuration updates
- **TargetCenter**: Target configuration center for managing device targets and their lifecycle
- **HostCollector**: Manages collection from individual hosts
- **DeviceCollector**: Handles device-level collection logic
- **JobsQueue**: Manages collection job scheduling and execution
- **DeviceCenter**: Central management for device configurations

### Protocol Collectors

- **SNMPv2Collector**: SNMP version 2c data collection
- **SshCollector**: SSH-based command execution and data collection
- **Kubernetes**: Kubernetes cluster data collection via kubectl
- **RestCollector**: REST/RESTCONF API data collection with authentication support
- **GraphQlCollector**: GraphQL API data collection with flexible querying

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

- **Enhanced SNMP Timeout Protection**: Configurable timeout with net-snmp fallback mechanism for robust OID collection
- **Net-SNMP Fallback**: Automatic fallback to net-snmp when WapSNMP timeouts occur
- **Job Queue Optimization**: Priority-based scheduling with configurable cadences
- **Connection Pooling**: Efficient reuse of protocol connections
- **Resource Limits**: Configurable limits prevent resource exhaustion
- **Improved Error Recovery**: Enhanced timeout handling and connection resilience

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

#### Updated Components (Latest)
- **SNMP Collector**: Enhanced timeout handling with net-snmp fallback, improved error recovery and connection resilience
- **HostCollector**: Optimized error message construction and protocol validation using l8utils/strings
- **SSH Collector**: Enhanced connection error handling and logging with efficient string building
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

### REST/RESTCONF
- HTTP/HTTPS-based API data collection
- Multiple authentication methods (token-based, basic auth)
- Support for GET, POST, PUT, PATCH, DELETE methods
- Flexible body and response type handling
- Certificate-based secure connections
- Configurable HTTP prefixes and endpoints

### GraphQL
- GraphQL query execution
- API key and token-based authentication
- Flexible query structure support
- Typed response handling with protobuf integration
- HTTPS with certificate support

## Dependencies

### Core Dependencies
- **Go 1.24.0+**: Programming language runtime
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
- **l8web**: Web client libraries for REST and GraphQL support

## Installation

### Prerequisites
- Go 1.24.0 or later
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

L8Collector is designed to run as a service within the Layer8 ecosystem with Service Level Agreement (SLA) support:

```go
// Service activation with SLA
collectorService := &CollectorService{}
sla := ifs.NewServiceLevelAgreement(collectorService, "collector", serviceArea, false, nil)
err := collectorService.Activate(sla, vnic)

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

The ExecuteService enables distributed job execution across cluster nodes with SLA support:

```go
// Execute a job remotely with SLA
executeService := &ExecuteService{}
slaExec := ifs.NewServiceLevelAgreement(executeService, "exec", serviceArea, false, nil)
slaExec.SetArgs(collectorService)
vnic.Resources().Services().Activate(slaExec, vnic)

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

#### REST/RESTCONF Collection
```go
restCollector := &RestCollector{}
restCollector.Init(hostProtocol, resources)
restCollector.Connect()

// Execute REST job with method, endpoint, and body
job := &types.CJob{
    PollarisName: "devices",
    JobName: "get-device",
}
// Poll format: "METHOD::endpoint::body"
// Example: "GET::/api/devices::{"query":"filter"}"
restCollector.Exec(job)
restCollector.Disconnect()
```

#### GraphQL Collection
```go
graphQlCollector := &GraphQlCollector{}
graphQlCollector.Init(hostProtocol, resources)
graphQlCollector.Connect()

// Execute GraphQL query
job := &types.CJob{
    PollarisName: "devices",
    JobName: "query-devices",
}
// Poll.What contains the GraphQL query string
graphQlCollector.Exec(job)
graphQlCollector.Disconnect()
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
- `tests/CollectorRest_test.go`: REST collector tests
- `tests/TestInit.go`: Test initialization
- `tests/utils_collector/`: Test utilities and mocks

## Development

### Project Structure

```
l8collector/
├── LICENSE                 # Apache 2.0 license
├── README.md              # This file
├── Layer8Logo.gif         # Layer8 logo
├── web.html               # Web interface
└── go/
    ├── collector/
    │   ├── common/        # Common interfaces and constants
    │   ├── targets/       # Target device management
    │   │   ├── TargetService.go       # Target management service
    │   │   └── TargetCenter.go        # Target configuration center
    │   ├── protocols/     # Protocol implementations
    │   │   ├── snmp/     # SNMP v2c collector
    │   │   ├── ssh/      # SSH collector
    │   │   ├── k8s/      # Kubernetes collector
    │   │   ├── rest/     # REST/RESTCONF collector
    │   │   ├── graphql/  # GraphQL collector
    │   │   └── Utils.go  # Protocol utilities
    │   └── service/       # Core services
    │       ├── CollectorService.go    # Main collection service with SLA
    │       ├── ExecuteService.go      # Remote execution service with SLA
    │       ├── HostCollector.go       # Host-level operations
    │       ├── JobsQueue.go           # Job scheduling
    │       ├── JobCadence.go          # Job cadence management
    │       ├── StaticJobs.go          # Static job configurations
    │       └── BootSequence.go        # Service boot sequence
    ├── tests/             # Test files
    │   ├── CollectorGraphQl_test.go  # GraphQL collector tests
    │   ├── CollectorRest_test.go     # REST collector tests
    │   └── Collector_test.go         # Main collector tests
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

## Latest Updates

### Recent Improvements (2024-2025)

#### Service Level Agreement (SLA) Integration (October 2025)
- **SLA Support**: Added Service Level Agreement support to core services
  - CollectorService now activates with SLA for better service management
  - ExecuteService integrated with SLA for remote job execution
  - Improved service lifecycle management and monitoring
- **Target Management Refactoring**:
  - Introduced TargetService for centralized target device management
  - Added TargetCenter for dynamic device configuration updates
  - Replaced DeviceService with more flexible target-based architecture
  - Support for L8PTarget and L8PTargetList types
- **Enhanced Testing**: Updated test suite with SLA integration tests

### Previous Improvements (2024-2025)

#### New Protocol Support
- **REST/RESTCONF Collector**: Full-featured REST API data collection with support for all HTTP methods (GET, POST, PUT, PATCH, DELETE)
  - Token-based and basic authentication
  - Flexible request/response handling with protobuf integration
  - Certificate-based secure connections
  - Configurable endpoints and HTTP prefixes
- **GraphQL Collector**: GraphQL query execution support
  - API key and token-based authentication
  - Flexible query structure
  - Typed response handling with protobuf
  - HTTPS with certificate support
- **Protocol Utilities**: Common utility functions for protocol implementations (SetValue, Keys for CTable/CMap operations)

#### Enhanced SNMP Resilience
- **Net-SNMP Fallback**: Automatic fallback to net-snmp when WapSNMP timeouts occur for improved OID collection reliability
- **Configurable Timeouts**: Enhanced timeout handling with contextual cancellation for better resource management
- **Error Recovery**: Improved connection resilience and error reporting for SNMP operations

#### Performance & Stability
- **Go 1.24.0**: Updated to latest Go version for improved performance and security
- **Enhanced Logging**: Better error context and debugging information across all collectors
- **Memory Optimization**: Continued string handling improvements reducing garbage collection pressure

#### Dependencies Updates
- Updated Layer8 ecosystem dependencies with latest security patches and performance improvements
- Enhanced WapSNMP integration with fallback mechanisms
- Added l8web package for REST and GraphQL client support

---

For questions, issues, or contributions, please use the GitHub issue tracker or submit pull requests.