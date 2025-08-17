# L8Collector

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.23.8-blue.svg)](https://golang.org/dl/)

L8Collector is a multi-protocol network data collection service built on the Layer8 ecosystem and Pollaris model. It provides a unified framework for collecting data from various network devices and systems using different protocols including SNMP, SSH, and Kubernetes.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
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
- **Error Handling**: Robust error handling and connection management
- **Testing Framework**: Comprehensive unit testing with coverage reporting

## Architecture

The L8Collector follows a modular architecture with the following key components:

### Core Components

- **CollectorService**: Main service orchestrating the collection process
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
- **l8utils**: Utility libraries
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