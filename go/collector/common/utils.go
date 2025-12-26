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

package common

import (
	"bytes"
	"math/rand"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

// ReplaceArguments performs variable substitution in a command string using
// the job's Arguments map. Variables are identified by a '$' prefix followed
// by the variable name, terminated by a space or end of string.
//
// Example:
//
//	what = "get pods -n $namespace"
//	job.Arguments = {"namespace": "kube-system"}
//	result = "get pods -n kube-system"
//
// If the job has no arguments or a referenced variable is not found,
// the original string is returned unchanged. This function is primarily
// used by the Kubernetes collector for dynamic command templating.
//
// Parameters:
//   - what: The command string containing $variable placeholders
//   - job: The collection job containing the Arguments map
//
// Returns:
//   - The command string with all variables replaced, or the original
//     string if no substitution was needed or possible
func ReplaceArguments(what string, job *l8tpollaris.CJob) string {
	if job.Arguments == nil {
		return what
	}
	buff := bytes.Buffer{}
	arg := bytes.Buffer{}
	open := false
	for _, c := range what {
		if c == '$' {
			open = true
		} else if c == ' ' && open {
			open = false
			v, ok := job.Arguments[arg.String()]
			if !ok {
				return what
			}
			buff.WriteString(v)
			buff.WriteString(" ")
			arg.Reset()
		} else if open {
			arg.WriteRune(c)
		} else {
			buff.WriteRune(c)
		}
	}

	if open {
		v, ok := job.Arguments[arg.String()]
		if !ok {
			return what
		}
		buff.WriteString(v)
	}
	return buff.String()
}

// RandomSecondWithin15Minutes returns a random integer between 0 and 899,
// representing a random second within a 15-minute window (900 seconds).
// Used for smoothing initial collection times to prevent thundering herd
// scenarios when SmoothFirstCollection is enabled.
func RandomSecondWithin15Minutes() int {
	return rand.Intn(900)
}

// RandomSecondWithin3Minutes returns a random integer between 0 and 179,
// representing a random second within a 3-minute window (180 seconds).
// Used for shorter randomization intervals in collection scheduling.
func RandomSecondWithin3Minutes() int {
	return rand.Intn(180)
}
