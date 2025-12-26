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

package service

import (
	"math/rand"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

// JobCadence returns the current cadence interval for a job in seconds.
// The cadence system supports multiple intervals that can increase as data
// stabilizes (e.g., poll frequently at start, then slow down).
//
// When SmoothFirstCollection is enabled, the first execution of each cadence
// level is randomized to prevent thundering herd scenarios where many devices
// would poll simultaneously.
//
// Parameters:
//   - job: The collection job containing cadence configuration
//
// Returns:
//   - The cadence interval in seconds for the current level
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
