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

package cache

import (
	"encoding/binary"
	"net"
	"reflect"
	"sort"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

type internalQuery struct {
	query    ifs.IQuery
	data     []string
	stamp    int64
	hash     string
	metadata *l8api.L8MetaData
	lastUsed int64
}

func newInternalQuery(query ifs.IQuery) *internalQuery {
	iq := &internalQuery{query: query}
	iq.hash = query.Hash()
	iq.metadata = newMetadata()
	iq.lastUsed = time.Now().Unix()
	return iq
}

func (this *internalQuery) prepare(cache map[string]interface{}, addedOrder []string, stamp int64, descending bool, metadataFunc map[string]func(interface{}) (bool, string)) {
	this.stamp = stamp

	data := make([]string, 0)

	//if added order is not nil, there is no criteria the query
	//so just use the added order as data
	if addedOrder != nil {
		data = addedOrder
	} else {
		this.metadata = newMetadata()
		for k, v := range cache {
			if this.query.Match(v) {
				data = append(data, k)
				addToMetadata(v, metadataFunc, this.metadata)
			}
		}
	}

	//sort the data
	sort.Slice(data, func(i, j int) bool {
		//if the added order is nil and the query have a sort by
		if addedOrder == nil && this.query.SortBy() != "" {
			v1 := this.query.SortByValue(cache[data[i]])
			v2 := this.query.SortByValue(cache[data[j]])
			if v1 != nil && v2 != nil {
				reuslt := lessThan(v1, v2)
				if descending {
					return !reuslt
				} else {
					return reuslt
				}
			}
		}
		//We just sort according to the key
		return lessThan(data[i], data[j])
	})
	this.data = data
}

func lessThan(a interface{}, b interface{}) bool {
	switch v1 := a.(type) {
	case int:
		if v2, ok := b.(int); ok {
			return v1 < v2
		}
	case int64:
		if v2, ok := b.(int64); ok {
			return v1 < v2
		}
	case int32:
		if v2, ok := b.(int32); ok {
			return v1 < v2
		}
	case float64:
		if v2, ok := b.(float64); ok {
			return v1 < v2
		}
	case float32:
		if v2, ok := b.(float32); ok {
			return v1 < v2
		}
	case string:
		if v2, ok := b.(string); ok {
			// Check if both strings are IPv4 addresses
			ip1 := net.ParseIP(v1)
			ip2 := net.ParseIP(v2)
			if ip1 != nil && ip2 != nil {
				// Check if they are IPv4 (not IPv6)
				ip1v4 := ip1.To4()
				ip2v4 := ip2.To4()
				if ip1v4 != nil && ip2v4 != nil {
					// Convert IPv4 to uint32 for comparison
					num1 := binary.BigEndian.Uint32(ip1v4)
					num2 := binary.BigEndian.Uint32(ip2v4)
					return num1 < num2
				}
			}
			// If not both IPv4, compare as regular strings
			return v1 < v2
		}
	case uint:
		if v2, ok := b.(uint); ok {
			return v1 < v2
		}
	case uint64:
		if v2, ok := b.(uint64); ok {
			return v1 < v2
		}
	case uint32:
		if v2, ok := b.(uint32); ok {
			return v1 < v2
		}
	}

	// Handle custom types (like protobuf enums) using reflection
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	// Both must have the same underlying kind
	if va.Kind() != vb.Kind() {
		return false
	}

	switch va.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return va.Int() < vb.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return va.Uint() < vb.Uint()
	case reflect.Float32, reflect.Float64:
		return va.Float() < vb.Float()
	case reflect.String:
		return va.String() < vb.String()
	}

	return false
}
