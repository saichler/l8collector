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

package registry

import (
	"reflect"

	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/maps"
)

type TypesMap struct {
	impl *maps.SyncMap
}

func NewTypesMap() *TypesMap {
	s2t := &TypesMap{}
	s2t.impl = maps.NewSyncMap()
	return s2t
}

func (m *TypesMap) Put(key string, value reflect.Type) (bool, error) {
	info, err := NewInfo(value)
	if err != nil {
		return false, err
	}
	return m.impl.Put(key, info), err
}

func (m *TypesMap) Get(key string) (*Info, bool) {
	value, ok := m.impl.Get(key)
	if value != nil {
		info := value.(*Info)
		return info, ok
	}
	return nil, ok
}

func (m *TypesMap) Del(key string) bool {
	_, ok := m.impl.Delete(key)
	return ok
}

func (m *TypesMap) Contains(key string) bool {
	return m.impl.Contains(key)
}

func (m *TypesMap) TypeList() *l8api.L8TypeList {
	typeList := &l8api.L8TypeList{}
	typeList.List = make([]string, 0)
	m.impl.Iterate(func(k, v interface{}) {
		typeList.List = append(typeList.List, k.(string))
	})
	return typeList
}
