package service

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

type JobsQueue struct {
	target   *l8tpollaris.L8PTarget
	hostId   string
	jobs     []*l8tpollaris.CJob
	jobsMap  map[string]*l8tpollaris.CJob
	mtx      *sync.Mutex
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
	this.hostId = ""
	this.target = nil
}

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

func MarkStart(job *l8tpollaris.CJob) {
	if job.ErrorCount == 0 {
		job.LastResult = job.Result
	}
	job.Started = time.Now().Unix()
	job.Ended = 0
	job.Result = nil
	job.Error = ""
}

func MarkEnded(job *l8tpollaris.CJob) {
	job.Ended = time.Now().Unix()
}

func JobKey(polarisName, jobName string) string {
	buff := bytes.Buffer{}
	buff.WriteString(polarisName)
	buff.WriteString(jobName)
	return buff.String()
}
