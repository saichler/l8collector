/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"github.com/saichler/l8bus/go/overlay/vnic"
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/service"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	common2 "github.com/saichler/probler/go/prob/common"
	"github.com/saichler/probler/go/prob/common/creates"
	"os"
)

func main() {
	common.SmoothFirstCollection = true
	res := common2.CreateResources("admission")
	ifs.SetNetworkMode(ifs.NETWORK_K8s)
	nic := vnic.NewVirtualNetworkInterface(res, nil)
	nic.Start()
	nic.WaitForConnection()

	//Activate pollaris
	pollaris.Activate(nic)

	//no need to activate with links id k8s as they are the same area for collection
	service.Activate(common2.K8sC_Links_ID, nic)
	res.Logger().SetLogLevel(ifs.Error_Level)

	//Here send a message to self to start collecting
	cl := creates.CreateCluster2(os.Getenv("ClusterName"))
	coll, _ := nic.Resources().Services().ServiceHandler(common2.AdControl_Service_Name, common2.AdControl_Service_Area)
	coll.Post(object.New(nil, cl), nic)
	common2.WaitForSignal(res)
}
