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

// Package utils_collector provides test utilities for the L8Collector service.
// It includes mock services and helper functions for creating test configurations.
package utils_collector

import (
	"sync"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
)

// ServiceType is the identifier for the MockParsingService.
const (
	ServiceType = "MockParsingService"
)

// MockParsingService is a test mock that simulates the parser service.
// It receives completed jobs from the CollectorService and tracks how many
// times each job has been executed. This allows tests to verify that jobs
// are being collected and forwarded correctly.
type MockParsingService struct {
	jobsComplete map[string]map[string]int
	mtx          *sync.Mutex
}

// Activate initializes the MockParsingService by registering the CJob type
// and creating the job tracking map. Implements the IService interface.
func (this *MockParsingService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8tpollaris.CJob{})
	this.jobsComplete = make(map[string]map[string]int)
	this.mtx = &sync.Mutex{}
	return nil
}

// DeActivate is a no-op for the mock service. Implements the IService interface.
func (this *MockParsingService) DeActivate() error {
	return nil
}

// Post receives a completed job and increments its execution count.
// This is the primary method for tracking job completions in tests.
func (this *MockParsingService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	job := pb.Element().(*l8tpollaris.CJob)
	jp, ok := this.jobsComplete[job.PollarisName]
	if !ok {
		jp = make(map[string]int)
		this.jobsComplete[job.PollarisName] = jp
	}
	jp[job.JobName]++
	//fmt.Println("Result:", string(job.Result))
	return nil
}

// JobsCounts returns the job execution counts map for test verification.
// The map structure is: pollarisName -> jobName -> executionCount
func (this *MockParsingService) JobsCounts() map[string]map[string]int {
	return this.jobsComplete
}

// Put is a no-op stub for the IService interface.
func (this *MockParsingService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Patch is a no-op stub for the IService interface.
func (this *MockParsingService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Delete is a no-op stub for the IService interface.
func (this *MockParsingService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Get is a no-op stub for the IService interface.
func (this *MockParsingService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// GetCopy is a no-op stub for the IService interface.
func (this *MockParsingService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Failed is a no-op stub for the IService interface.
func (this *MockParsingService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

// TransactionConfig returns nil as the mock service doesn't use transactions.
func (this *MockParsingService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

// WebService returns nil as the mock service doesn't expose a web interface.
func (this *MockParsingService) WebService() ifs.IWebService {
	return nil
}
