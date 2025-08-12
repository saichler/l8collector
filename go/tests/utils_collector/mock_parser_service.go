package utils_collector

import (
	"fmt"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8types/go/ifs"
	"sync"
)

const (
	ServiceType = "MockParsingService"
)

type MockParsingService struct {
	jobsComplete map[string]map[string]int
	mtx          *sync.Mutex
}

func (this *MockParsingService) Activate(serviceName string, serviceArea byte,
	r ifs.IResources, l ifs.IServiceCacheListener, args ...interface{}) error {
	r.Registry().Register(&types.CJob{})
	this.jobsComplete = make(map[string]map[string]int)
	this.mtx = &sync.Mutex{}
	return nil
}

func (this *MockParsingService) DeActivate() error {
	return nil
}

func (this *MockParsingService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	job := pb.Element().(*types.CJob)
	jp, ok := this.jobsComplete[job.PollarisName]
	if !ok {
		jp = make(map[string]int)
		this.jobsComplete[job.PollarisName] = jp
	}
	jp[job.JobName]++
	fmt.Println("Result:", string(job.Result))
	return nil
}

func (this *MockParsingService) JobsCounts() map[string]map[string]int {
	return this.jobsComplete
}

func (this *MockParsingService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *MockParsingService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *MockParsingService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *MockParsingService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *MockParsingService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *MockParsingService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}
func (this *MockParsingService) TransactionMethod() ifs.ITransactionMethod {
	return nil
}
func (this *MockParsingService) WebService() ifs.IWebService {
	return nil
}
