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
	"errors"
	"time"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/protocols/graphql"
	"github.com/saichler/l8collector/go/collector/protocols/k8s"
	"github.com/saichler/l8collector/go/collector/protocols/rest"
	"github.com/saichler/l8collector/go/collector/protocols/snmp"
	"github.com/saichler/l8collector/go/collector/protocols/ssh"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/maps"
	"github.com/saichler/l8utils/go/utils/strings"
)

// HostCollector manages data collection for a single host within a device target.
// It creates and manages protocol-specific collectors based on the host configuration
// and coordinates the execution of collection jobs through the boot sequence and
// steady-state polling.
//
// The HostCollector:
//   - Creates protocol collectors (SNMP, SSH, REST, GraphQL, Kubernetes)
//   - Manages the boot sequence for device discovery and configuration
//   - Executes scheduled collection jobs via the JobsQueue
//   - Handles job completion and forwards results to the parser service
//   - Tracks device online/offline status
type HostCollector struct {
	service          *CollectorService      // Parent service reference
	target           *l8tpollaris.L8PTarget // Target device configuration
	hostId           string                 // Unique identifier for this host
	collectors       *maps.SyncMap          // Protocol -> ProtocolCollector map
	jobsQueue        *JobsQueue             // Queue of scheduled collection jobs
	running          bool                   // Flag indicating if collection is active
	currentBootStage int                    // Current boot stage index (0-4)
	bootStages       []*BootState           // Boot state tracking for each stage
	pollarisName     string                 // Identified device pollaris profile name
}

// newHostCollector creates a new HostCollector instance for the specified host.
// It initializes the collectors map, jobs queue, and registers the parser service
// link for sending collection results.
//
// Parameters:
//   - target: The L8PTarget containing host configurations
//   - hostId: The unique identifier for this host
//   - service: The parent CollectorService
//
// Returns:
//   - A new HostCollector ready to be started
func newHostCollector(target *l8tpollaris.L8PTarget, hostId string, service *CollectorService) *HostCollector {
	hc := &HostCollector{}
	hc.target = target
	hc.hostId = hostId
	hc.collectors = maps.NewSyncMap()
	hc.service = service
	hc.jobsQueue = NewJobsQueue(target, hostId, service)
	hc.running = true
	hc.bootStages = make([]*BootState, 5)
	return hc
}

func (this *HostCollector) update() error {
	host := this.target.Hosts[this.hostId]
	for _, config := range host.Configs {
		exist := this.collectors.Contains(config.Protocol)
		if !exist {
			col, err := newProtocolCollector(config, this.service.vnic.Resources())
			if err != nil {
				return this.service.vnic.Resources().Logger().Error(err)
			}
			if col != nil {
				this.collectors.Put(config.Protocol, col)
			}
		}
	}

	this.bootStages = make([]*BootState, 5)
	this.bootStages[0] = this.newBootState(0)

	return nil
}

func (this *HostCollector) stop() {
	this.sendDeviceDown()
	this.running = false
	this.collectors.Iterate(func(k, v interface{}) {
		c := v.(common.ProtocolCollector)
		c.Disconnect()
	})
	this.collectors = nil
	this.jobsQueue.Shutdown()
	this.jobsQueue = nil
	this.bootStages = nil
	this.target = nil
	this.service = nil
}

func (this *HostCollector) sendDeviceDown() {
	job := &l8tpollaris.CJob{
		TargetId:     this.target.TargetId,
		HostId:       this.hostId,
		LinksId:      this.target.LinksId,
		JobName:      "deviceStatus",
		PollarisName: "boot02",
		Always:       true,
	}
	staticJobs["deviceStatus"].(*DeviceStatusJob).doDown(job, this)
	pService, pArea := targets.Links.Parser(job.LinksId)
	this.service.vnic.Proximity(pService, pArea, ifs.POST, job)
	this.service.vnic.Resources().Logger().Info("Sending Device Down")
}

// start initializes protocol collectors and begins the collection process.
// It creates collectors for each protocol configured for this host and
// starts the collection goroutine which processes jobs from the queue.
//
// Returns:
//   - Always returns nil (errors are logged but don't prevent startup)
func (this *HostCollector) start() error {
	host := this.target.Hosts[this.hostId]
	for _, config := range host.Configs {
		col, err := newProtocolCollector(config, this.service.vnic.Resources())
		if err != nil {
			this.service.vnic.Resources().Logger().Error(err)
		}
		if col != nil {
			this.collectors.Put(config.Protocol, col)
		}
	}

	this.bootStages[0] = this.newBootState(0)

	go this.collect()

	return nil
}

func (this *HostCollector) collect() {
	// Capture references before they may be cleared by stop()
	resources := this.service.vnic.Resources()
	targetId := this.target.TargetId
	hostId := this.hostId

	pc := pollaris.Pollaris(resources)
	var job *l8tpollaris.CJob
	var waitTime int64
	for this.running {

		job, waitTime = this.jobsQueue.Pop()
		if job != nil {
			resources.Logger().Debug("Poped job ", job.PollarisName, ":", job.JobName)
		} else {
			resources.Logger().Debug("No Job, waitTime ", waitTime)
		}

		if job != nil {
			poll := pc.Poll(job.PollarisName, job.JobName)
			if poll == nil {
				resources.Logger().Error(strings.New("cannot find poll ", job.PollarisName, " - ", job.JobName, " for device id ").String(), targetId)
				continue
			}
			MarkStart(job)

			// Static jobs (ipAddress, deviceStatus) are handled locally
			// regardless of boot stage - they never go to protocol collectors
			if sjob, ok := staticJobs[job.JobName]; ok {
				sjob.do(job, this)
				MarkEnded(job)
				this.jobComplete(job)
				if this.currentBootStage < len(this.bootStages) {
					this.bootStages[this.currentBootStage].jobComplete(job)
					for this.bootStages[this.currentBootStage].isComplete() {
						this.currentBootStage++
						if this.currentBootStage >= len(this.bootStages) {
							break
						}
						this.bootStages[this.currentBootStage] = this.newBootState(this.currentBootStage)
					}
				}
				continue
			}

			c, ok := this.collectors.Get(poll.Protocol)
			if !ok {
				MarkEnded(job)
				this.jobsQueue.DisableJob(job)
				continue
			}

			c.(common.ProtocolCollector).Exec(job)
			MarkEnded(job)
			if this.running {
				this.jobComplete(job)
				if this.currentBootStage < len(this.bootStages) {
					this.bootStages[this.currentBootStage].jobComplete(job)
					for this.bootStages[this.currentBootStage].isComplete() {
						this.currentBootStage++
						if this.currentBootStage >= len(this.bootStages) {
							break
						}
						this.bootStages[this.currentBootStage] = this.newBootState(this.currentBootStage)
					}
				}
			}

			if job.ErrorCount >= 5 {
				resources.Logger().Error("Job ", job.TargetId, " - ", job.PollarisName, " - ",
					job.JobName, " has failed ", job.ErrorCount, " in a row.")
			}
		} else {
			resources.Logger().Debug("No more jobs, next job in ", waitTime, " seconds.")
			time.Sleep(time.Second * time.Duration(waitTime))
		}
	}
	resources.Logger().Debug("Host collection for device ", targetId, " host ", hostId, " has ended.")
}

func (this *HostCollector) execJob(job *l8tpollaris.CJob) bool {
	pc := pollaris.Pollaris(this.service.vnic.Resources())
	poll := pc.Poll(job.PollarisName, job.JobName)
	if poll == nil {
		panic(this.target.TargetId + ": cannot find poll " + job.PollarisName + "/" + job.JobName)
		this.service.vnic.Resources().Logger().Error("cannot find poll for device id ", this.target.TargetId)
		return false
	}
	MarkStart(job)
	c, ok := this.collectors.Get(poll.Protocol)
	if !ok {
		MarkEnded(job)
		return false
	}
	c.(common.ProtocolCollector).Exec(job)
	MarkEnded(job)
	return true
}

func newProtocolCollector(config *l8tpollaris.L8PHostProtocol, resource ifs.IResources) (common.ProtocolCollector, error) {
	var protocolCollector common.ProtocolCollector
	if config.Protocol == l8tpollaris.L8PProtocol_L8PGraphQL {
		protocolCollector = &graphql.GraphQlCollector{}
	} else if config.Protocol == l8tpollaris.L8PProtocol_L8PRESTCONF {
		protocolCollector = &rest.RestCollector{}
	} else if config.Protocol == l8tpollaris.L8PProtocol_L8PSSH {
		protocolCollector = &ssh.SshCollector{}
	} else if config.Protocol == l8tpollaris.L8PProtocol_L8PPSNMPV2 {
		protocolCollector = &snmp.SNMPv2Collector{}
	} else if config.Protocol == l8tpollaris.L8PProtocol_L8PKubectl {
		protocolCollector = &k8s.Kubernetes{}
	} else {
		return nil, errors.New(strings.New("Unknown Protocol ", config.Protocol.String()).String())
	}
	err := protocolCollector.Init(config, resource)
	return protocolCollector, err
}

func (this *HostCollector) jobComplete(job *l8tpollaris.CJob) {
	if job.Error != "" {
		this.service.vnic.Resources().Logger().Error("Job ", job.TargetId, " - ", job.PollarisName,
			" - ", job.JobName, " has an error:", job.Error)
		job.Cadence.Current = 0
		return
	}

	if !jobHasChange(job) {
		this.service.vnic.Resources().Logger().Debug("Job", job.JobName, " has no change")
		return
	}

	pService, pArea := targets.Links.Parser(job.LinksId)
	this.service.agg.AddElement(job, ifs.Proximity, "", pService, pArea, ifs.POST)
	//err := this.service.vnic.Proximity(pService, pArea, ifs.POST, job)
	//if err != nil {
	//	this.service.vnic.Resources().Logger().Error("HostCollector:", err.Error())
	//}
	if job.JobName == "systemMib" {
		this.service.vnic.Resources().Logger().Debug("SystemMib for ", job.TargetId, " was received")
		this.bootDetailDevice(job)
	}
}

func jobHasChange(job *l8tpollaris.CJob) bool {
	if job.Always {
		return true
	}
	if job.Result != nil && job.Cadence.Current < int32(len(job.Cadence.Cadences)-1) {
		job.Cadence.Current++
	}
	if job.LastResult == nil && job.Result == nil {
		return false
	} else if job.LastResult == nil && job.Result != nil {
		return true
	} else if job.Result == nil {
		return true
	}
	if len(job.Result) != len(job.LastResult) {
		return true
	}
	for i := 0; i < len(job.Result); i++ {
		if job.Result[i] != job.LastResult[i] {
			return true
		}
	}
	return false
}
