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

package pollaris

import "github.com/saichler/l8utils/go/utils/strings"

// pollarisKey generates a composite key from the given pollaris attributes.
// The key is constructed by concatenating non-empty attributes with '+'
// separators. The key format is: name+vendor+series+family+software+hardware+version
// This key is used for hierarchical lookups where more specific configurations
// (with more attributes) take precedence over less specific ones.
func pollarisKey(name, vendor, series, family, software, hardware, version string) string {
	buff := strings.New()
	buff.Add(name)
	addToKey(vendor, buff)
	addToKey(series, buff)
	addToKey(family, buff)
	addToKey(software, buff)
	addToKey(hardware, buff)
	addToKey(version, buff)
	return buff.String()
}

// addToKey appends a string to the key buffer with a '+' separator.
// Empty strings are ignored to allow optional key components.
func addToKey(str string, buff *strings.String) {
	if str != "" {
		buff.Add("+")
		buff.Add(str)
	}
}
