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

package service

import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
)

// BootState tracks the progress of a boot stage during device discovery.
// Each boot stage has a set of jobs that must complete before advancing
// to the next stage. The BootState tracks which jobs have completed.
//
// The boot sequence follows these stages:
//   - Stage 0: Initial discovery (system MIB collection)
//   - Stage 1: Basic connectivity validation
//   - Stage 2: Device capability discovery
//   - Stage 3: Extended MIB and feature discovery
//   - Stage 4: Final configuration and steady-state transition
type BootState struct {
	jobNames map[string]bool // Map of job names to completion status
	stage    int             // Current boot stage index
}

// newBootState creates a new BootState for the specified boot stage.
// It queries the pollaris registry for all polls in the stage's group
// and creates jobs for those polls that have matching protocol collectors.
//
// Parameters:
//   - stage: The boot stage index (0-4)
//
// Returns:
//   - A new BootState initialized with the stage's jobs
func (this *HostCollector) newBootState(stage int) *BootState {
	bs := &BootState{}
	bs.stage = stage
	bs.jobNames = make(map[string]bool)
	pollList, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), common.BootStages[stage],
		"", "", "", "", "", "")
	if err != nil {
		this.service.vnic.Resources().Logger().Error("Boot stage ", stage, " does not exist,skipping")
		return bs
	}
	for _, pollrs := range pollList {
		hasProtocol := false
		for _, poll := range pollrs.Polling {
			_, ok := this.collectors.Get(poll.Protocol)
			if ok {
				bs.jobNames[poll.Name] = false
				hasProtocol = true
			}
		}
		if hasProtocol {
			err = this.jobsQueue.InsertJob(pollrs.Name, "", "", "", "", "", "", 0, 0)
			if err != nil {
				this.service.vnic.Resources().Logger().Error("Error adding pollaris to boot: ", err)
			}
		}
	}
	return bs
}

// isComplete checks if all jobs in this boot stage have completed.
// Returns true when every job in jobNames has been marked as complete.
func (this *BootState) isComplete() bool {
	for _, complete := range this.jobNames {
		if !complete {
			return false
		}
	}
	return true
}

// doStaticJob checks if the job is a static job and executes it if so.
// Static jobs are special built-in operations like device detail discovery.
// If the job is found in the static jobs registry, it is executed and
// marked as complete.
//
// Parameters:
//   - job: The collection job to check and potentially execute
//   - hostColletor: The host collector context for execution
//
// Returns:
//   - true if the job was a static job and was handled
//   - false if the job is not a static job
func (this *BootState) doStaticJob(job *l8tpollaris.CJob, hostColletor *HostCollector) bool {
	sjob, ok := staticJobs[job.JobName]
	if ok {
		sjob.do(job, hostColletor)
		_, ok = this.jobNames[job.JobName]
		if ok {
			this.jobNames[job.JobName] = true
		}
		return true
	}
	return false
}

// jobComplete marks a job as completed in this boot stage's tracking map.
// If the job is part of this stage, its completion status is set to true.
//
// Parameters:
//   - job: The completed collection job
func (this *BootState) jobComplete(job *l8tpollaris.CJob) {
	_, ok := this.jobNames[job.JobName]
	if ok {
		this.jobNames[job.JobName] = true
	}
}

// bootDetailDevice processes the system MIB response to identify the device
// type and load appropriate polling configuration. It extracts the sysObjectID
// OID from the SNMP response and looks up the corresponding pollaris profile.
//
// This method is called during boot stage 0 after the initial system MIB
// collection completes. It sets the pollarisName for the host collector,
// which determines the device-specific polls to execute.
//
// Parameters:
//   - job: The completed system MIB job containing the SNMP walk results
func (this *HostCollector) bootDetailDevice(job *l8tpollaris.CJob) {
	if this.pollarisName != "" {
		return
	}
	if job.Result == nil || len(job.Result) < 3 {
		this.service.vnic.Resources().Logger().Error("HostCollector.loadPolls: ", job.TargetId, " has sysmib empty Result")
		return
	}
	enc := object.NewDecode(job.Result, 0, this.service.vnic.Resources().Registry())
	data, err := enc.Get()
	if err != nil {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: ", job.TargetId, " has sysmib error ", err.Error())
		return
	}
	cmap, ok := data.(*l8tpollaris.CMap)
	if !ok {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: ", job.TargetId, " systemMib not A CMap")
		return
	}
	strData, ok := cmap.Data[".1.3.6.1.2.1.1.2.0"]
	if !ok {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: ", job.TargetId, " sysmib does not contain sysoid")
		return
	}

	enc = object.NewDecode(strData, 0, this.service.vnic.Resources().Registry())
	byteInterface, _ := enc.Get()
	sysoid, _ := byteInterface.(string)
	this.service.vnic.Resources().Logger().Debug("HostCollector, loadPolls: ", job.TargetId, " discovered sysoid =", sysoid)
	if sysoid == "" {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: ", job.TargetId, " - sysoid is blank!")
		/* when there is DebugEnabled
		for k, v := range cmap.Data {
			enc = object.NewDecode(v, 0, this.service.vnic.Resources().Registry())
			val, _ := enc.Get()
			this.service.vnic.Resources().Logger().Debug("Key =", k, " value=", val)
		}*/
		return
	}

	plrs := boot.GetPollarisByOid(sysoid)
	plc := pollaris.Pollaris(this.service.vnic.Resources())
	plc.Post(plrs, false)
	if plrs != nil {
		if plrs.Name != "boot03" {
			this.pollarisName = plrs.Name
			this.insertCustomJobs(plrs.Name)
		}
	}
}

// insertCustomJobs adds custom polling jobs for a specific pollaris profile.
// This is called after device identification to schedule device-specific
// data collection jobs based on the identified device type.
//
// Parameters:
//   - pollarisName: The name of the pollaris profile to load jobs from
func (this *HostCollector) insertCustomJobs(pollarisName string) {
	this.jobsQueue.InsertJob(pollarisName, "", "", "", "", "", "", 0, 0)
}
