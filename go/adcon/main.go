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
	"fmt"
	"github.com/saichler/l8bus/go/overlay/vnic"
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/service"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	common2 "github.com/saichler/probler/go/prob/common"
	"os"
)

// k8sPrimeObjectLinkIDs is the list of K8s/Istio prime objects we collect.
// One target per LinkID is posted to the collector so that JobsQueue.target.LinksId
// is per-PrimeObject and CJobs route to the matching parser/inventory cache.
var k8sPrimeObjectLinkIDs = []string{
	common2.K8sClust_Links_ID,
	common2.K8sPod_Links_ID,
	common2.K8sDeploy_Links_ID,
	common2.K8sSts_Links_ID,
	common2.K8sDs_Links_ID,
	common2.K8sRs_Links_ID,
	common2.K8sJob_Links_ID,
	common2.K8sCj_Links_ID,
	common2.K8sHpa_Links_ID,
	common2.K8sSvc_Links_ID,
	common2.K8sIng_Links_ID,
	common2.K8sNetPol_Links_ID,
	common2.K8sEp_Links_ID,
	common2.K8sEpSl_Links_ID,
	common2.K8sIngCl_Links_ID,
	common2.K8sPv_Links_ID,
	common2.K8sPvc_Links_ID,
	common2.K8sScl_Links_ID,
	common2.K8sCm_Links_ID,
	common2.K8sSec_Links_ID,
	common2.K8sRq_Links_ID,
	common2.K8sLr_Links_ID,
	common2.K8sPdb_Links_ID,
	common2.K8sSa_Links_ID,
	common2.K8sRole_Links_ID,
	common2.K8sCr_Links_ID,
	common2.K8sRb_Links_ID,
	common2.K8sCrb_Links_ID,
	common2.K8sNode_Links_ID,
	common2.K8sNs_Links_ID,
	common2.K8sVCl_Links_ID,
	common2.IstioVs_Links_ID,
	common2.IstioDr_Links_ID,
	common2.IstioGw_Links_ID,
	common2.IstioSe_Links_ID,
	common2.IstioPa_Links_ID,
	common2.IstioAp_Links_ID,
	common2.IstioSc_Links_ID,
	common2.IstioEf_Links_ID,
	common2.K8sCrd_Links_ID,
	common2.K8sEvt_Links_ID,
}

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
	res.Logger().SetLogLevel(ifs.Info_Level)

	clusterName := os.Getenv("ClusterName")
	coll, _ := nic.Resources().Services().ServiceHandler(common2.AdControl_Service_Name, common2.AdControl_Service_Area)
	fmt.Println("Posting", len(k8sPrimeObjectLinkIDs), "K8s targets to the collector!")
	for _, linkID := range k8sPrimeObjectLinkIDs {
		t := newK8sTarget(clusterName, linkID)
		t.State = l8tpollaris.L8PTargetState_Up
		coll.Post(object.New(nil, t), nic)
	}
	fmt.Println("Posted!")
	common2.WaitForSignal(res)
}

// newK8sTarget builds one collection target for a single K8s prime object type.
// The target's LinksId routes every CJob produced by its JobsQueue to the
// per-type parser registered in prob/common/Links_k8s.go.
func newK8sTarget(clusterName, linkID string) *l8tpollaris.L8PTarget {
	t := &l8tpollaris.L8PTarget{}
	t.TargetId = clusterName + "/" + linkID
	t.LinksId = linkID
	t.InventoryType = l8tpollaris.L8PTargetType_K8s_Cluster

	host := &l8tpollaris.L8PHost{}
	host.HostId = clusterName

	k8sConfig := &l8tpollaris.L8PHostProtocol{}
	k8sConfig.Protocol = l8tpollaris.L8PProtocol_L8PKubernetesAPI

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	host.Configs[int32(k8sConfig.Protocol)] = k8sConfig

	t.Hosts = make(map[string]*l8tpollaris.L8PHost)
	t.Hosts[t.TargetId] = host
	return t
}
