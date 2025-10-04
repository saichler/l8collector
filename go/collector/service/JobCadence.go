package service

import (
	"math/rand"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

func JobCadence(job *l8tpollaris.CJob) int64 {
	if common.SmoothFirstCollection && job.Cadence.Startups == nil {
		job.Cadence.Startups = make([]int64, len(job.Cadence.Cadences))
		for i := 0; i < len(job.Cadence.Startups); i++ {
			job.Cadence.Startups[i] = -1
		}
	}

	if common.SmoothFirstCollection && job.Cadence.Startups[job.Cadence.Current] == -1 {
		job.Cadence.Startups[job.Cadence.Current] = rand.Int63n(job.Cadence.Cadences[job.Cadence.Current])
		return job.Cadence.Startups[job.Cadence.Current] + job.Cadence.Cadences[job.Cadence.Current]
	} else {
		return job.Cadence.Cadences[job.Cadence.Current]
	}

}
