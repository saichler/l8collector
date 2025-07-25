package service

import (
	"bytes"
	"errors"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"sync"
	"time"
)

type JobsQueue struct {
	deviceId string
	hostId   string
	jobs     []*types.Job
	jobsMap  map[string]*types.Job
	mtx      *sync.Mutex
	iService *types.DeviceServiceInfo
	pService *types.DeviceServiceInfo
	shutdown bool
	service  *CollectorService
}

func (this *JobsQueue) Shutdown() {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.shutdown = true
	this.jobs = nil
	this.jobsMap = nil
	this.service = nil
	this.iService = nil
	this.pService = nil
	this.hostId = ""
	this.deviceId = ""
}

func NewJobsQueue(deviceId, hostId string, service *CollectorService,
	iService *types.DeviceServiceInfo, pService *types.DeviceServiceInfo) *JobsQueue {
	jq := &JobsQueue{}
	jq.service = service
	jq.mtx = &sync.Mutex{}
	jq.jobs = make([]*types.Job, 0)
	jq.jobsMap = make(map[string]*types.Job)
	jq.deviceId = deviceId
	jq.hostId = hostId
	jq.iService = iService
	jq.pService = pService
	return jq
}

func (this *JobsQueue) newJobsForKey(name, vendor, series, family, software, hardware, version string, cadence, timeout int64) map[string]*types.Job {
	p, err := pollaris.PollarisByKey(this.service.vnic.Resources(), name, vendor, series, family, software, hardware, version)
	if err != nil {
		return nil
	}
	jobs := make(map[string]*types.Job)
	for jobName, poll := range p.Polling {
		job := &types.Job{}
		job.JobName = jobName
		job.PollarisName = p.Name
		job.Cadence = cadence
		job.Timeout = timeout
		job.DeviceId = this.deviceId
		job.HostId = this.hostId
		job.IService = this.iService
		job.PService = this.pService

		if job.Cadence == 0 {
			job.Cadence = poll.Cadence
		}
		if job.Timeout == 0 {
			job.Timeout = poll.Timeout
		}
		jobs[jobName] = job
	}
	return jobs
}

func (this *JobsQueue) newJobsForGroup(groupName, vendor, series, family, software, hardware, version string) []*types.Job {
	polarises, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), groupName, vendor, series, family, software, hardware, version)
	if err != nil {
		return nil
	}
	jobs := make([]*types.Job, 0)
	for _, p := range polarises {
		for jobName, poll := range p.Polling {
			job := &types.Job{}
			job.DeviceId = this.deviceId
			job.HostId = this.hostId
			job.JobName = jobName
			job.PollarisName = p.Name
			job.Cadence = poll.Cadence
			job.Timeout = poll.Timeout
			job.IService = this.iService
			job.PService = this.pService
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (this *JobsQueue) InsertJob(polarisName, vendor, series, family, software, hardware, version string, cadence, timeout int64) error {
	if this == nil {
		return errors.New("Job Queue is already shutdown")
	}
	jobs := this.newJobsForKey(polarisName, vendor, series, family, software, hardware, version, cadence, timeout)
	if jobs == nil {
		return errors.New("cannot find pollaris to create jobs")
	}
	this.mtx.Lock()
	defer this.mtx.Unlock()
	if this.shutdown {
		return errors.New("Job Queue is already shutdown")
	}
	for _, job := range jobs {
		jobKey := JobKey(job.PollarisName, job.JobName)
		old, ok := this.jobsMap[jobKey]
		if !ok {
			this.jobsMap[jobKey] = job
			this.jobs = append(this.jobs, job)
		} else {
			old.Started = 0
			old.Ended = 0
		}
	}
	return nil
}

func (this *JobsQueue) Pop() (*types.Job, int64) {
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
	var job *types.Job
	index := -1
	now := time.Now().Unix()
	waitTimeTillNext := int64(999999)
	for i, j := range this.jobs {
		timeSinceExecuted := now - j.Ended
		if timeSinceExecuted >= j.Cadence {
			job = j
			index = i
			break
		} else {
			timeTillNextExecution := j.Cadence - timeSinceExecuted
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
		swap := make([]*types.Job, 0)
		job := this.jobs[index]
		swap = append(swap, this.jobs[0:index]...)
		swap = append(swap, this.jobs[index+1:]...)
		swap = append(swap, job)
		for i, j := range swap {
			this.jobs[i] = j
		}
	}
}

func MarkStart(job *types.Job) {
	job.Started = time.Now().Unix()
	job.Ended = 0
	job.Result = nil
	job.Error = ""
}

func MarkEnded(job *types.Job) {
	job.Ended = time.Now().Unix()
}

func JobKey(polarisName, jobName string) string {
	buff := bytes.Buffer{}
	buff.WriteString(polarisName)
	buff.WriteString(jobName)
	return buff.String()
}
