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

// Package k8s provides a Kubernetes protocol collector implementation for
// the L8Collector service. It enables data collection from Kubernetes clusters
// using kubectl commands with kubeconfig-based authentication.
package k8s

import (
	"encoding/base64"
	"os"
	"os/exec"

	"github.com/google/uuid"
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"
)

// Kubernetes implements the ProtocolCollector interface for Kubernetes clusters.
// It executes kubectl commands against configured clusters and collects the
// output as serialized string data.
//
// Features:
//   - Base64-encoded kubeconfig support for secure credential storage
//   - Context-aware cluster configuration
//   - Dynamic parameter substitution using $variable syntax
//   - Automatic temporary file cleanup for script execution
//   - Support for any kubectl command through pollaris configuration
//
// The kubeconfig is decoded from base64 and written to a temporary file
// during initialization. The file is automatically cleaned up on Disconnect.
type Kubernetes struct {
	resources  ifs.IResources                // Layer8 resources for logging and security
	config     *l8tpollaris.L8PHostProtocol  // Host configuration with credential reference
	kubeConfig string                        // Path to the temporary kubeconfig file
	context    string                        // Kubernetes context name to use
	connected  bool                          // Connection state flag
}

// Init initializes the Kubernetes collector with the provided host configuration.
// It retrieves the kubeconfig from the security service (stored as base64-encoded
// data), decodes it, and writes it to a temporary file for kubectl to use.
//
// The credential is expected to contain:
//   - context: The Kubernetes context name (returned as username)
//   - kubeconfig: Base64-encoded kubeconfig file contents (returned as password)
//
// Parameters:
//   - config: Host protocol configuration containing the credential ID
//   - resources: Layer8 resources for accessing security credentials and logging
//
// Returns:
//   - error if credential retrieval, decoding, or file writing fails
func (this *Kubernetes) Init(config *l8tpollaris.L8PHostProtocol, resources ifs.IResources) error {
	this.resources = resources
	this.config = config
	_, context, kubeconfig, _, err := this.resources.Security().Credential(this.config.CredId, "kubeconfig", this.resources)
	if err != nil {
		panic(err)
	}
	this.context = context
	this.kubeConfig = ".kubeadm-" + context
	data, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		return err
	}
	err = os.WriteFile(this.kubeConfig, data, 0644)
	return err
}

// Protocol returns the protocol type identifier for Kubernetes.
// This is used by the collector service to route jobs to the correct collector.
func (this *Kubernetes) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PKubectl
}

// Exec executes a kubectl command job against the configured Kubernetes cluster.
// The command is obtained from the pollaris configuration using the job's
// PollarisName and JobName. Variable substitution is performed on the command
// using the job's Arguments map (e.g., "$namespace" is replaced with the value
// from job.Arguments["namespace"]).
//
// The execution process:
//  1. Retrieves the poll configuration for the command template
//  2. Performs variable substitution on the command
//  3. Generates a temporary shell script with the kubectl command
//  4. Executes the script using bash
//  5. Captures the output and stores it in the job's Result field
//  6. Cleans up the temporary script file
//
// Parameters:
//   - job: The collection job containing pollaris reference, arguments, and result storage
func (this *Kubernetes) Exec(job *l8tpollaris.CJob) {
	this.resources.Logger().Debug("K8s Job ", job.PollarisName, ":", job.JobName, " started")
	defer this.resources.Logger().Debug("K8s Job ", job.PollarisName, ":", job.JobName, " ended")

	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		this.resources.Logger().Error(strings.New("K8s:", err.Error()).String())
		return
	}

	script := strings.New("kubectl --kubeconfig=")
	script.Add(this.kubeConfig)
	script.Add(" --context=")
	script.Add(this.context)
	script.Add(" ")
	script.Add(common.ReplaceArguments(poll.What, job))
	script.Add("\n")

	id := uuid.New().String()
	in := strings.New("./", id, ".sh").String()
	defer os.Remove(in)
	os.WriteFile(in, script.Bytes(), 0777)
	c := exec.Command("bash", "-c", in, "2>&1")
	o, e := c.Output()
	if e != nil {
		job.Error = e.Error()
		job.ErrorCount++
	} else {
		job.ErrorCount = 0
	}
	obj := object.NewEncode()
	obj.Add(string(o))
	job.Result = obj.Data()
}

// Connect is a no-op for the Kubernetes collector.
// Kubernetes connections are established on-demand during Exec via kubectl.
//
// Returns:
//   - Always returns nil
func (this *Kubernetes) Connect() error {
	return nil
}

// Disconnect cleans up the Kubernetes collector resources.
// It removes the temporary kubeconfig file created during Init and
// resets all internal state. After calling Disconnect, the collector
// must be re-initialized before use.
//
// Returns:
//   - Always returns nil (cleanup is best-effort)
func (this *Kubernetes) Disconnect() error {
	// Delete the kubeconfig file created in Init()
	if this.kubeConfig != "" {
		os.Remove(this.kubeConfig)
		this.kubeConfig = ""
	}
	this.resources = nil
	this.config = nil
	this.context = ""
	this.connected = false
	return nil
}

// Online returns the connection status of the Kubernetes collector.
// Returns true if the collector has been initialized and is ready to execute commands.
func (this *Kubernetes) Online() bool {
	return this.connected
}
