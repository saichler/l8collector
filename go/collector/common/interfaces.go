/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package common provides shared interfaces, constants, and utility functions
// used across the L8Collector service. It defines the core ProtocolCollector
// interface that all protocol implementations must satisfy, as well as boot
// stage constants for the collector initialization sequence.
package common

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
)

// Boot stage constants define the sequential stages of the collector boot process.
// Each stage represents a phase in the initialization and discovery sequence:
//   - BOOT_STAGE_00: Initial discovery and system identification
//   - BOOT_STAGE_01: Basic connectivity validation
//   - BOOT_STAGE_02: Device capability discovery
//   - BOOT_STAGE_03: Extended MIB and feature discovery
//   - BOOT_STAGE_04: Final configuration and steady-state transition
const (
	BOOT_STAGE_00 = "Boot_Stage_00"
	BOOT_STAGE_01 = "Boot_Stage_01"
	BOOT_STAGE_02 = "Boot_Stage_02"
	BOOT_STAGE_03 = "Boot_Stage_03"
	BOOT_STAGE_04 = "Boot_Stage_04"
)

// BootStages is an ordered slice of all boot stage identifiers, used for
// iterating through the boot sequence in the correct order.
var BootStages = []string{BOOT_STAGE_00, BOOT_STAGE_01, BOOT_STAGE_02, BOOT_STAGE_03, BOOT_STAGE_04}

// ProtocolCollector defines the interface that all protocol-specific collectors
// must implement. This interface enables the collector service to interact with
// different protocols (SNMP, SSH, Kubernetes, REST, GraphQL) in a uniform manner.
//
// Implementations of this interface handle:
//   - Connection lifecycle management (Connect, Disconnect)
//   - Job execution for data collection (Exec)
//   - Protocol identification (Protocol)
//   - Connection state tracking (Online)
type ProtocolCollector interface {
	// Init initializes the protocol collector with host configuration and resources.
	// This method should set up any protocol-specific settings but should not
	// establish the actual connection. Returns an error if initialization fails.
	Init(*l8tpollaris.L8PHostProtocol, ifs.IResources) error

	// Protocol returns the protocol type identifier for this collector.
	// Used to match jobs to the appropriate collector implementation.
	Protocol() l8tpollaris.L8PProtocol

	// Exec executes a collection job using this protocol. The job contains
	// all necessary information including the pollaris name, job name, and
	// any arguments. Results are stored in the job's Result field, and errors
	// are recorded in the job's Error and ErrorCount fields.
	Exec(job *l8tpollaris.CJob)

	// Connect establishes the connection to the target device using this protocol.
	// Should be called before Exec. Returns an error if connection fails.
	Connect() error

	// Disconnect closes the connection to the target device and releases
	// any associated resources. Should be called when the collector is
	// no longer needed or during shutdown.
	Disconnect() error

	// Online returns true if the collector has successfully connected and
	// completed at least one successful poll. Used for device status reporting.
	Online() bool
}

// SmoothFirstCollection when set to true, enables randomized initial collection
// timing to prevent thundering herd scenarios when many devices start collecting
// simultaneously. When enabled, the first collection for each job will be
// delayed by a random interval within the job's cadence period.
var SmoothFirstCollection = false
