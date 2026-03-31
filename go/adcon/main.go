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
	"github.com/saichler/l8collector/go/collector/protocols/k8sclient"
	"github.com/saichler/l8collector/go/collector/service"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8web/go/web/server"
	common2 "github.com/saichler/probler/go/prob/common"
	"github.com/saichler/probler/go/prob/common/creates"
	"os"
	"strconv"
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

	if err := startAdmissionServer(res); err != nil {
		res.Logger().Error("Failed to start admission server: ", err.Error())
		return
	}

	//no need to activate with links id k8s as they are the same area for collection
	service.Activate(common2.K8sC_Links_ID, nic)
	res.Logger().SetLogLevel(ifs.Error_Level)
	//Here send a message to self to start collecting
	cl := creates.CreateCluster2(os.Getenv("ClusterName"))
	coll, _ := nic.Resources().Services().ServiceHandler(common2.AdControl_Service_Name, common2.AdControl_Service_Area)
	coll.Post(object.New(nil, cl), nic)
	common2.WaitForSignal(res)
}

func startAdmissionServer(resources ifs.IResources) error {
	port, err := envInt("ADMISSION_PORT", 8443)
	if err != nil {
		return err
	}

	collector := &k8sclient.ClientGoCollector{}
	err = collector.Init(&l8tpollaris.L8PHostProtocol{
		Protocol: l8tpollaris.L8PProtocol_L8PKubernetesAPI,
	}, resources)
	if err != nil {
		return err
	}
	err = collector.Connect()
	if err != nil {
		return err
	}

	serverConfig := &server.RestServerConfig{
		Host:           envString("ADMISSION_HOST", "0.0.0.0"),
		Port:           port,
		Authentication: false,
		Prefix:         "",
		CertName:       envString("ADMISSION_CERT_NAME", "/data/admission"),
	}
	svr, err := server.NewRestServer(serverConfig)
	if err != nil {
		return err
	}
	err = collector.RegisterAdmissionHandler(svr.(*server.RestServer), envString("ADMISSION_PATH", k8sclient.DefaultAdmissionPath))
	if err != nil {
		return err
	}
	go func() {
		if startErr := svr.Start(); startErr != nil {
			resources.Logger().Error("Admission web server stopped: ", startErr.Error())
		}
	}()
	return nil
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}
