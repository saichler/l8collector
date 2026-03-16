// © 2025 Sharon Aicler (saichler@gmail.com)
//
// Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vnic

import (
	"github.com/saichler/l8types/go/ifs"
)

// VnicAPI provides a high-level API for service communication through a VNic.
// It supports standard REST-like operations (Get, Post, Put, Patch, Delete).
type VnicAPI struct {
	serviceName string
	serviceArea byte
	vnic        *VirtualNetworkInterface
	leader      bool
	all         bool
}

func (v VnicAPI) Post(i interface{}) ifs.IElements {

	//TODO implement me
	panic("implement me")
}

func (v VnicAPI) Put(i interface{}) ifs.IElements {
	//TODO implement me
	panic("implement me")
}

func (v VnicAPI) Patch(i interface{}) ifs.IElements {
	//TODO implement me
	panic("implement me")
}

func (v VnicAPI) Delete(i interface{}) ifs.IElements {
	//TODO implement me
	panic("implement me")
}

func (v VnicAPI) Get(s string) ifs.IElements {
	//TODO implement me
	panic("implement me")
}

func newAPI(serviceName string, serviceArea byte, leader, all bool) ifs.ServiceAPI {
	api := &VnicAPI{}
	api.serviceName = serviceName
	api.serviceArea = serviceArea
	api.leader = leader
	api.all = all
	return api
}
