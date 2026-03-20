# L8Collector

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.26.1-blue.svg)](https://golang.org/dl/)

**© 2025-2026 Sharon Aicler (saichler@gmail.com)**

Part of the Layer 8 Ecosystem - Licensed under the Apache License, Version 2.0.

---

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

L8Collector is designed as a microservice within the Layer8 ecosystem that enables automated data collection from network infrastructure. It implements a pluggable architecture that supports multiple collection protocols and can be easily extended to support additional protocols.

The collector operates on a target-centric model where each target device can have multiple hosts, and each host can be polled using different protocols based on the device configuration. Target management is handled by the Pollaris framework (`l8pollaris`), which provides centralized target lifecycle management and configuration.

## Features

- **Multi-Protocol Support**: SNMP v2c, SSH, Kubernetes, REST/RESTCONF, and GraphQL data collection
- **Concurrent Collection**: Parallel data collection with goroutine-per-host concurrency model
- **Service-Oriented Architecture**: Built as a microservice with Layer8 framework and SLA support
- **Target Management**: Dynamic target device configuration via Pollaris TargetCenter
- **5-Stage Boot Sequence**: Progressive device discovery and capability detection
- **Job Queuing**: Cadence-based job scheduling with round-robin execution
- **Remote Job Execution**: ExecuteService for distributed job processing across cluster nodes
- **Result Aggregation**: Batched result forwarding to parser service via Aggregator
- **Parameter Substitution**: Dynamic argument replacement in Kubernetes commands
- **Smooth First Collection**: Optional randomized initial collection timing to prevent thundering herd
- **Error Handling**: Robust error handling with SNMP net-snmp fallback mechanism
- **Testing Framework**: Integration tests using Layer8 topology with opensim simulator

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│         Layer8 Virtual Network (IVNic)                  │
└─────────────────────────────┬───────────────────────────┘
                              │
        ┌─────────────────────┴─────────────────────────┐
        │                                               │
   ┌────▼──────────────────┐    ┌───────────────────────▼──┐
   │  CollectorService     │    │  ExecuteService          │
   │  (Main orchestrator)  │◄───┤  (Remote job execution)  │
   │                       │    │                          │
   │ - Manages targets     │    │ - Handles CJob POST      │
   │ - Creates HostCollect │    │ - Local or distributed   │
   │ - Thread-safe map     │    │ - Fallback to other nodes│
   └────┬──────────────────┘    └──────────────────────────┘
        │
        │ (One per host)
        │
   ┌────▼──────────────────────────────────┐
   │  HostCollector                        │
   │  (Per-host orchestrator)              │
   │                                       │
   │  - 5-stage boot sequence              │
   │  - Creates protocol collectors        │
   │  - Manages JobsQueue                  │
   │  - Forwards results via Aggregator    │
   └────┬──────────────────────────────────┘
        │
        │ (One per protocol per host)
        │
   ┌────┴──────────────────────────────────────────────┐
   │                                                    │
   │  SNMPv2   SSH    K8s    REST    GraphQL            │
   │  (Protocol Collectors - ProtocolCollector iface)   │
   └────────────────────────────────────────────────────┘
```

### Core Components

- **CollectorService**: Main service orchestrating the collection process with SLA support. Receives `L8PTarget` messages to start/stop polling devices.
- **ExecuteService**: Service for remote job execution and distribution across cluster nodes. Falls back to routing jobs to other collector instances when the local host collector isn't found.
- **HostCollector**: Manages collection from individual hosts. Creates protocol collectors, runs the boot sequence, and executes scheduled jobs via JobsQueue.
- **JobsQueue**: Maintains an ordered list of scheduled jobs with cadence-based execution. Jobs execute round-robin — completed jobs move to the end of the queue.
- **JobCadence**: Manages time-based job execution intervals (minimum 3-second cadence).
- **BootSequence**: 5-stage progressive device discovery process (stages 00-04).
- **Aggregator**: Batches collection results before forwarding to the parser service.

### Protocol Collectors

All protocol collectors implement the `ProtocolCollector` interface:

```go
type ProtocolCollector interface {
    Init(*L8PHostProtocol, IResources) error
    Protocol() L8PProtocol
    Exec(job *CJob)
    Connect() error
    Disconnect() error
    Online() bool
}
```

- **SNMPv2Collector**: SNMP v2c data collection with net-snmp fallback
- **SshCollector**: SSH-based command execution and data collection
- **Kubernetes**: kubectl-based cluster data collection with parameter substitution
- **RestCollector**: REST/RESTCONF API data collection with authentication support
- **GraphQlCollector**: GraphQL API data collection with flexible querying

### Data Flow

1. **Target Arrival**: `L8PTarget` message arrives via POST → CollectorService creates HostCollectors
2. **Boot Sequence**: Each HostCollector runs 5 stages of progressive device discovery
3. **Steady-State Polling**: JobsQueue schedules and executes jobs based on cadence intervals
4. **Result Aggregation**: Results are batched by the Aggregator and forwarded to the parser service
5. **Remote Execution**: ExecuteService handles on-demand jobs, routing to the correct collector instance

## Scalability & Performance

### Device Capacity

- **Conservative estimate**: 1,000-2,000 devices per collector instance
- **Optimistic estimate**: 5,000-10,000 devices per collector instance
- Scales horizontally using Layer8 service mesh for larger deployments

### Concurrency Model

- **1:1 Host-to-Goroutine mapping**: Each host runs in a dedicated goroutine
- **Per-protocol collectors**: SNMP, SSH, K8s, REST, GraphQL collectors created per host
- **Thread-safe collections**: Uses `SyncMap` for device management
- **Non-blocking execution**: Jobs run sequentially per host, hosts run in parallel

### Resource Consumption

Per device resource usage:
- **Memory**: ~1-2KB overhead + job queue storage
- **Goroutines**: 1+ per host (typically 1 host per device)
- **Network connections**: 1 per protocol per host
- **CPU**: Based on polling cadence (minimum 3-second intervals)

### Performance Optimizations

- **SNMP Net-SNMP Fallback**: Automatic fallback to net-snmp when WapSNMP timeouts occur
- **Configurable Timeouts**: Contextual cancellation for SNMP operations
- **Job Queue Optimization**: Round-robin scheduling with configurable cadences
- **Connection Reuse**: Protocol connections reused across multiple jobs
- **Smooth First Collection**: Optional randomized initial timing prevents thundering herd
- **Result Aggregation**: Batched forwarding reduces network overhead

### Horizontal Scaling

- Multiple collector instances handling different device sets
- Layer8 service mesh for distributed coordination
- Load balancing across available collector nodes
- Automatic failover and job redistribution via ExecuteService

## Supported Protocols

### SNMP v2c
- Community-based authentication
- Configurable timeout and retry settings
- OID-based data collection with OID normalization
- Support for SNMP walks and gets
- Net-SNMP fallback for timeout resilience

### SSH
- Username/password authentication
- Command execution with prompt detection
- Session management
- Configurable prompts and timeouts

### Kubernetes
- kubectl-based data collection
- Context-aware configuration
- Base64-encoded kubeconfig support
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
- **Go 1.26.1+**: Programming language runtime
- **github.com/cdevr/WapSNMP**: SNMP protocol implementation
- **golang.org/x/crypto**: SSH client implementation
- **github.com/google/uuid**: UUID generation
- **google.golang.org/protobuf**: Protocol Buffers serialization

### Layer8 Ecosystem Dependencies
- **l8pollaris**: Polling framework, target management, and data modeling (L8PTarget, CJob, etc.)
- **l8bus**: Layer8 messaging bus for inter-service communication
- **l8types**: Common type definitions and interfaces (IService, IVNic, IResources)
- **l8utils**: Utility libraries (SyncMap, Aggregator, optimized string handling)
- **l8srlz**: Serialization framework
- **l8parser**: Data parsing framework (collector forwards results here)
- **l8web**: Web client libraries for REST and GraphQL support
- **l8test**: Test framework with virtual network topology
- **l8services**: Service management framework (indirect)
- **probler**: Problem/error handling
- **podys**: System utilities

## Installation

### Prerequisites
- Go 1.26.1 or later
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
- Fetch and vendor dependencies
- Run unit tests with coverage
- Generate HTML coverage reports

## Configuration

### Target Configuration

Targets are configured using the Pollaris model (`L8PTarget`) with hosts and protocol-specific connections:

```go
target := &l8tpollaris.L8PTarget{
    TargetId: "device-001",
    State:    l8tpollaris.L8PTargetState_Up,
    Hosts: []*l8tpollaris.L8PHost{
        {
            HostId: "host-001",
            Protocols: []*l8tpollaris.L8PHostProtocol{
                // Protocol-specific connection configuration
            },
        },
    },
}
```

### Environment Configuration

The service integrates with the Layer8 ecosystem and uses standard Layer8 configuration:

- Service registration and discovery via SLA
- Resource management through IResources
- Logging configuration (configurable levels)
- Virtual network interface management (IVNic)

## Usage

### Service Activation

L8Collector runs as a service within the Layer8 ecosystem:

```go
// Activate collector service with Links configuration
service.Activate(linksID, vnic)
```

This internally creates a `CollectorService` with an SLA and registers the `ExecuteService` for remote job handling.

### Starting/Stopping Collection

Send `L8PTarget` messages to the collector service:

```go
// Start polling a device
target := &l8tpollaris.L8PTarget{
    TargetId: "device-001",
    State:    l8tpollaris.L8PTargetState_Up,
    Hosts:    hosts,
}
collectorService.Post(object.New(target), vnic)

// Stop polling
target.State = l8tpollaris.L8PTargetState_Down
collectorService.Post(object.New(target), vnic)
```

### Remote Job Execution

The ExecuteService handles on-demand job execution:

```go
job := &l8tpollaris.CJob{
    DeviceId: "device-001",
    HostId:   "host-001",
    Arguments: map[string]string{
        "namespace": "kube-system",
    },
}
executeService.Post(object.New(job), vnic)
```

### Protocol-Specific Usage

#### SNMP Collection
```go
snmpCollector := &snmp.SNMPv2Collector{}
snmpCollector.Init(hostProtocol, resources)
snmpCollector.Connect()
snmpCollector.Exec(job)
snmpCollector.Disconnect()
```

#### SSH Collection
```go
sshCollector := &ssh.SshCollector{}
sshCollector.Init(hostProtocol, resources)
sshCollector.Connect()
sshCollector.Exec(job)
sshCollector.Disconnect()
```

#### Kubernetes Collection
```go
k8sCollector := &k8s.Kubernetes{}
k8sCollector.Init(hostProtocol, resources)
// Commands use $variable substitution from job.Arguments
k8sCollector.Exec(job)
```

#### REST/RESTCONF Collection
```go
restCollector := &rest.RestCollector{}
restCollector.Init(hostProtocol, resources)
restCollector.Connect()
restCollector.Exec(job) // Poll format: "METHOD::endpoint::body"
restCollector.Disconnect()
```

#### GraphQL Collection
```go
graphQlCollector := &graphql.GraphQlCollector{}
graphQlCollector.Init(hostProtocol, resources)
graphQlCollector.Connect()
graphQlCollector.Exec(job) // Poll.What contains the GraphQL query
graphQlCollector.Disconnect()
```

## Testing

### Running Tests

```bash
cd go

# Run all tests with coverage
go test -tags=unit -v -coverpkg=./collector/... -coverprofile=cover.html ./... --failfast

# View coverage report
go tool cover -html=cover.html
```

### Test Structure

Tests are located in `go/tests/` and use the Layer8 test framework (`l8test`) with a 4-node virtual network topology:

- `Collector_test.go` — Main collector integration tests (SNMP, SSH, K8s)
- `CollectorRest_test.go` — REST collector protocol tests
- `CollectorGraphQl_test.go` — GraphQL collector protocol tests
- `EntityMib_test.go` — MIB entity testing
- `TestInit.go` — Test environment initialization (4-node topology, opensim simulator)
- `activate.go` — Service activation helpers
- `utils_collector/utils.go` — Test utility functions
- `utils_collector/mock_parser_service.go` — Mock parser service for validating data flow

## Development

### Project Structure

```
l8collector/
├── LICENSE                 # Apache 2.0 license
├── README.md               # This file
├── Layer8Logo.gif          # Layer8 logo
└── go/
    ├── collector/
    │   ├── common/         # ProtocolCollector interface, boot stage constants, utils
    │   │   ├── interfaces.go
    │   │   └── utils.go
    │   ├── protocols/      # Protocol implementations
    │   │   ├── snmp/       # SNMP v2c + net-snmp fallback
    │   │   │   ├── SNMPv2.go
    │   │   │   ├── SNMPv2Walk.go
    │   │   │   └── NetSNMPv2.go
    │   │   ├── ssh/        # SSH collector
    │   │   │   └── Ssh.go
    │   │   ├── k8s/        # Kubernetes collector
    │   │   │   └── Kubernetes.go
    │   │   ├── rest/       # REST/RESTCONF collector
    │   │   │   └── RestCollector.go
    │   │   ├── graphql/    # GraphQL collector
    │   │   │   └── GraphSqlCollector.go
    │   │   └── Utils.go    # Shared protocol utilities
    │   └── service/        # Core services
    │       ├── CollectorService.go  # Main collection service with SLA
    │       ├── ExecuteService.go    # Remote execution service
    │       ├── HostCollector.go     # Host-level operations
    │       ├── BootSequence.go      # 5-stage boot process
    │       ├── JobsQueue.go         # Job scheduling
    │       ├── JobCadence.go        # Cadence management
    │       ├── StaticJobs.go        # Static job definitions
    │       └── hash.go              # Collector key generation
    ├── tests/              # Integration tests
    │   ├── Collector_test.go
    │   ├── CollectorRest_test.go
    │   ├── CollectorGraphQl_test.go
    │   ├── EntityMib_test.go
    │   ├── TestInit.go
    │   ├── activate.go
    │   └── utils_collector/
    │       ├── utils.go
    │       └── mock_parser_service.go
    ├── go.mod              # Go module definition
    ├── go.sum              # Go module checksums
    ├── test.sh             # Test script
    └── vendor/             # Vendored dependencies
```

### Adding New Protocols

To add a new protocol collector:

1. Implement the `ProtocolCollector` interface in a new package under `collector/protocols/`
2. Register the new protocol type in the Layer8 type system
3. Add the protocol case to `HostCollector.createProtocolCollectors()`
4. Add configuration support for the new protocol
5. Implement integration tests in `tests/`

### Code Style

- Follow Go best practices and conventions
- Use `github.com/saichler/l8utils/go/utils/strings.String` for string concatenations
- Leverage the strings package's native support for raw integer/numeric values
- Include proper error handling and logging
- Maintain test coverage

### Building and Testing

```bash
cd go

# Format code
go fmt ./...

# Vet code
go vet ./...

# Run tests
./test.sh

# Build (verify compilation)
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

```
© 2025-2026 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## Links

- [Layer8 Ecosystem](https://github.com/saichler/layer8)
- [L8Pollaris](https://github.com/saichler/l8pollaris)
- [L8Services](https://github.com/saichler/l8services)
