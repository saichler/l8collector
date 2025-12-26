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
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

// JobsQueue manages the scheduling and execution of collection jobs for a host.
// It maintains a list of jobs sorted by next execution time and provides
// thread-safe access to the job queue.
//
// The JobsQueue:
//   - Schedules jobs based on their configured cadence intervals
//   - Tracks job completion times for next execution calculation
//   - Supports dynamic job insertion during boot sequence
//   - Provides round-robin execution by moving executed jobs to the end
type JobsQueue struct {
	target   *l8tpollaris.L8PTarget         // Target device configuration
	hostId   string                         // Host identifier for this queue
	jobs     []*l8tpollaris.CJob            // Ordered list of scheduled jobs
	jobsMap  map[string]*l8tpollaris.CJob   // Map for quick job lookup by key
	mtx      *sync.Mutex                    // Mutex for thread-safe queue access
	shutdown bool                           // Flag indicating queue shutdown
	service  *CollectorService              // Parent service reference
}

// Shutdown gracefully stops the jobs queue and releases all resources.
// After shutdown, the queue cannot be used and all operations return errors.
func (this *JobsQueue) Shutdown() {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.shutdown = true
	this.jobs = nil
	this.jobsMap = nil
	this.service = nil
	this.hostId = ""
	this.target = nil
}

// NewJobsQueue creates a new JobsQueue for the specified target and host.
// The queue is initialized with empty job lists and is ready to accept jobs.
//
// Parameters:
//   - target: The L8PTarget containing device configuration
//   - hostId: The unique identifier for this host
//   - service: The parent CollectorService
//
// Returns:
//   - A new JobsQueue ready to accept and schedule jobs
func NewJobsQueue(target *l8tpollaris.L8PTarget, hostId string, service *CollectorService) *JobsQueue {
	jq := &JobsQueue{}
	jq.service = service
	jq.mtx = &sync.Mutex{}
	jq.jobs = make([]*l8tpollaris.CJob, 0)
	jq.jobsMap = make(map[string]*l8tpollaris.CJob)
	jq.target = target
	jq.hostId = hostId
	return jq
}

func (this *JobsQueue) newJobsForKey(name, vendor, series, family, software, hardware, version string) map[string]*l8tpollaris.CJob {
	p, err := pollaris.PollarisByKey(this.service.vnic.Resources(), name, vendor, series, family, software, hardware, version)
	if err != nil {
		return nil
	}
	jobs := make(map[string]*l8tpollaris.CJob)
	for jobName, poll := range p.Polling {
		job := &l8tpollaris.CJob{}
		job.JobName = jobName
		job.PollarisName = p.Name
		job.Cadence = poll.Cadence
		job.Timeout = poll.Timeout
		job.TargetId = this.target.TargetId
		job.HostId = this.hostId
		job.LinksId = this.target.LinksId
		job.Always = poll.Always
		if job.Timeout == 0 {
			job.Timeout = poll.Timeout
		}
		jobs[jobName] = job
	}
	return jobs
}

func (this *JobsQueue) newJobsForGroup(groupName, vendor, series, family, software, hardware, version string) []*l8tpollaris.CJob {
	polarises, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), groupName, vendor, series, family, software, hardware, version)
	if err != nil {
		return nil
	}
	jobs := make([]*l8tpollaris.CJob, 0)
	for _, p := range polarises {
		for jobName, poll := range p.Polling {

			if !poll.Cadence.Enabled {
				continue
			}

			job := &l8tpollaris.CJob{}
			job.TargetId = this.target.TargetId
			job.HostId = this.hostId
			job.JobName = jobName
			job.PollarisName = p.Name
			job.Cadence = poll.Cadence
			job.Timeout = poll.Timeout
			job.LinksId = this.target.LinksId
			job.Always = poll.Always
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (this *JobsQueue) InsertJob(polarisName, vendor, series, family, software, hardware, version string, cadence, timeout int64) error {
	if this == nil {
		return errors.New("Job Queue is already shutdown")
	}
	jobs := this.newJobsForKey(polarisName, vendor, series, family, software, hardware, version)
	if jobs == nil {
		return errors.New("cannot find pollaris to create jobs")
	}
	this.mtx.Lock()
	defer this.mtx.Unlock()
	if this.shutdown {
		return errors.New("Job Queue is already shutdown")
	}
	for _, job := range jobs {
		if !job.Cadence.Enabled {
			continue
		}
		jobKey := JobKey(job.PollarisName, job.JobName)
		old, ok := this.jobsMap[jobKey]
		if !ok {
			this.jobsMap[jobKey] = job
			this.jobs = append(this.jobs, job)
			this.service.vnic.Resources().Logger().Info("Added job ", job.PollarisName, " - ", job.JobName)
		} else {
			old.Started = 0
			old.Ended = 0
		}
	}
	return nil
}

func (this *JobsQueue) DisableJob(job *l8tpollaris.CJob) {
	job.Cadence.Enabled = false
}

// Pop returns the next job that is ready for execution based on its cadence.
// If no job is ready, it returns the time until the next job should execute.
//
// Returns:
//   - job: The next job to execute, or nil if no jobs are ready
//   - waitTime: Seconds until the next job should execute (if job is nil)
func (this *JobsQueue) Pop() (*l8tpollaris.CJob, int64) {
	if this == nil {
		return nil, -1
	}
	this.mtx.Lock()
	defer this.mtx.Unlock()
	if len(this.jobs) == 0 {
		this.service.vnic.Resources().Logger().Error("Jobs Queue is empty")
	}
	if this.shutdown {
		return nil, -1
	}
	var job *l8tpollaris.CJob
	index := -1
	now := time.Now().Unix()
	waitTimeTillNext := int64(999999)
	for i, j := range this.jobs {
		if !j.Cadence.Enabled {
			continue
		}
		timeSinceExecuted := now - j.Ended
		jobCadence := JobCadence(j)

		if timeSinceExecuted >= jobCadence {
			job = j
			index = i
			break
		} else {
			timeTillNextExecution := jobCadence - timeSinceExecuted
			if timeTillNextExecution < waitTimeTillNext {
				waitTimeTillNext = timeTillNextExecution
			}
		}
	}
	this.moveToLast(index)
	return job, waitTimeTillNext
}

func (this *JobsQueue) moveToLast(index int) {
	if index != -1 {
		swap := make([]*l8tpollaris.CJob, 0)
		job := this.jobs[index]
		swap = append(swap, this.jobs[0:index]...)
		swap = append(swap, this.jobs[index+1:]...)
		swap = append(swap, job)
		for i, j := range swap {
			this.jobs[i] = j
		}
	}
}

// MarkStart prepares a job for execution by saving the previous result
// and resetting execution state. Should be called before Exec.
func MarkStart(job *l8tpollaris.CJob) {
	if job.ErrorCount == 0 {
		job.LastResult = job.Result
	}
	job.Started = time.Now().Unix()
	job.Ended = 0
	job.Result = nil
	job.Error = ""
}

// MarkEnded records the job completion time. Should be called after Exec.
func MarkEnded(job *l8tpollaris.CJob) {
	job.Ended = time.Now().Unix()
}

// JobKey generates a unique key for a job by combining pollaris and job names.
// Used for storing and looking up jobs in the jobsMap.
func JobKey(polarisName, jobName string) string {
	buff := bytes.Buffer{}
	buff.WriteString(polarisName)
	buff.WriteString(jobName)
	return buff.String()
}
