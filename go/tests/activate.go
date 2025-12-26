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

package tests

import (
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/probler/go/prob/common"
)

// ActivateTargets registers and activates the targets service on the given VNic.
// This is a helper function used by tests to set up the target management
// service with the default database credentials and database name.
//
// Parameters:
//   - vnic: The virtual network interface to activate the service on
func ActivateTargets(vnic ifs.IVNic) {
	targets.Activate(common.DB_CREDS, common.DB_NAME, vnic)
}
