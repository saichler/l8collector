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

package rules

// Constants defining standard parameter and workspace key names used across parsing rules.
const (
	// Input is the workspace key for the input data to be parsed.
	Input = "input"
	// Output is the workspace key for storing the result of rule execution.
	Output = "output"
	// What is the parameter name for the substring to search for (used by Contains rule).
	What = "what"
	// From is the parameter name specifying the source field in input data.
	From = "from"
	// PropertyId is the workspace key for the target property path.
	PropertyId = "propertyid"
	// Start is the parameter name for the starting position in data.
	Start = "start"
	// Columns is the parameter name for the number of columns in table parsing.
	Columns = "columns"
	// KeyColumn is the parameter name for specifying which columns form the key.
	KeyColumn = "key_column"
	// JobEnded is the workspace key for the job's end timestamp (unix seconds).
	JobEnded = "job_ended"
)
