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
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

type internalCache struct {
	cache           map[string]interface{}
	UniqueToPrimary map[string]string
	PrimaryToUnique map[string]string
	hasExtraKeys    bool
	addedOrder      []string
	key2order       map[string]int
	deleteCount     int
	stamp           int64
	queries         map[string]*internalQuery
	metadataFunc    map[string]func(interface{}) (bool, string)
	globalMetadata  *l8api.L8MetaData
}

func newInternalCache() *internalCache {
	iq := &internalCache{}
	iq.cache = make(map[string]interface{})
	iq.addedOrder = make([]string, 0)
	iq.key2order = make(map[string]int)
	iq.queries = make(map[string]*internalQuery)
	iq.globalMetadata = newMetadata()
	iq.UniqueToPrimary = make(map[string]string)
	iq.PrimaryToUnique = make(map[string]string)
	return iq
}

func newMetadata() *l8api.L8MetaData {
	metadata := &l8api.L8MetaData{}
	metadata.KeyCount = &l8api.L8Count{}
	metadata.KeyCount.Counts = make(map[string]int32)
	metadata.ValueCount = make(map[string]*l8api.L8Count)
	return metadata
}

func (this *internalCache) removeFromMetadata(key string) (interface{}, bool) {
	old, ok := this.cache[key]
	if ok && this.metadataFunc != nil {
		for count, f := range this.metadataFunc {
			ok1, v := f(old)
			if ok1 {
				this.globalMetadata.KeyCount.Counts[count]--
				if v != "" {
					vCount, ok2 := this.globalMetadata.ValueCount[count]
					if ok2 {
						vCount.Counts[v]--
					}
				}
			}
		}
	}
	return old, ok
}

func (this *internalCache) addToMetadata(value interface{}) {
	addToMetadata(value, this.metadataFunc, this.globalMetadata)
}

func addToMetadata(value interface{}, metadataFunc map[string]func(interface{}) (bool, string), metadata *l8api.L8MetaData) {
	if metadataFunc != nil {
		for count, f := range metadataFunc {
			ok1, v := f(value)
			if ok1 {
				metadata.KeyCount.Counts[count]++
				if v != "" {
					vCount, ok2 := metadata.ValueCount[count]
					if !ok2 {
						vCount = &l8api.L8Count{}
						vCount.Counts = make(map[string]int32)
						metadata.ValueCount[count] = vCount
					}
					vCount.Counts[v]++
				}
			}
		}
	}
}

func (this *internalCache) put(pk, uk string, value interface{}) {
	_, ok := this.removeFromMetadata(pk)
	this.cache[pk] = value
	this.putUnique(pk, uk)
	if !ok {
		this.addedOrder = append(this.addedOrder, pk)
		this.stamp = time.Now().Unix()
		this.key2order[pk] = len(this.addedOrder) - 1
	}
	this.addToMetadata(value)
}

func (this *internalCache) get(pk, uk string) (interface{}, bool) {
	if pk == "" && uk == "" {
		return nil, false
	}
	if pk == "" && uk != "" {
		pk = this.UniqueToPrimary[uk]
	}
	item, ok := this.cache[pk]
	return item, ok
}

func (this *internalCache) delete(pk, uk string) (interface{}, bool) {
	item, ok := this.removeFromMetadata(pk)
	if !ok {
		return item, ok
	}
	delete(this.cache, pk)
	this.deleteUnique(pk, uk)

	// Mark as tombstone in addedOrder
	if idx, exists := this.key2order[pk]; exists {
		this.addedOrder[idx] = ""
		delete(this.key2order, pk)
		this.deleteCount++
	}

	this.stamp = time.Now().Unix()
	this.cleanupOrder()
	return item, ok
}

func (this *internalCache) cleanupOrder() {
	// Trigger cleanup when tombstones exceed threshold:
	// - More than 100 tombstones, OR
	// - Tombstones are more than 25% of the slice
	if this.deleteCount < 100 && (len(this.addedOrder) == 0 || this.deleteCount < len(this.addedOrder)/4) {
		return
	}

	// Create new slice without tombstones
	newOrder := make([]string, 0, len(this.addedOrder)-this.deleteCount)
	newKey2Order := make(map[string]int, len(this.addedOrder)-this.deleteCount)

	for _, pk := range this.addedOrder {
		if pk != "" { // Skip tombstones
			newKey2Order[pk] = len(newOrder)
			newOrder = append(newOrder, pk)
		}
	}

	this.addedOrder = newOrder
	this.key2order = newKey2Order
	this.deleteCount = 0
}

func (this *internalCache) size() int {
	return len(this.cache)
}

func (this *internalCache) fetch(start, blockSize int, q ifs.IQuery) ([]interface{}, *l8api.L8MetaData) {

	noCriteriaOrSort := true

	hash := q.Hash()
	dq, ok := this.queries[hash]
	//This is a new query, so create it
	if !ok {
		dq = newInternalQuery(q)
		this.queries[hash] = dq
	}

	// Update last used time for TTL tracking
	atomic.StoreInt64(&dq.lastUsed, time.Now().Unix())

	//If the query timestamp has changed, it means elements were added/removed
	//so we need re-cresate the sorted set
	if dq.stamp != this.stamp {
		qrt := reflect.ValueOf(q.Criteria())
		noCriteriaOrSort = (!qrt.IsValid() || qrt.IsNil()) && strings.TrimSpace(q.SortBy()) == ""
		//Query does not have criteria so use the added order for keys
		if noCriteriaOrSort {
			dq.prepare(this.cache, this.addedOrder, this.stamp, q.Descending(), this.metadataFunc)
		} else {
			//We need to create a dataset sorted by the sortby and filter by the criteria
			dq.prepare(this.cache, nil, this.stamp, q.Descending(), this.metadataFunc)
		}
	}

	//return just the subset of rows requested
	result := make([]interface{}, 0)
	for i := start; i < len(dq.data); i++ {
		key := dq.data[i]
		value, ok := this.cache[key]
		if ok {
			result = append(result, value)
		}
		if blockSize == 0 {
			continue
		}
		if len(result) >= blockSize {
			break
		}
	}
	if !noCriteriaOrSort {
		return result, dq.metadata
	}
	return result, this.globalMetadata
}

func (this *internalCache) addMetadataFunc(name string, f func(interface{}) (bool, string)) {
	if this.metadataFunc == nil {
		this.metadataFunc = make(map[string]func(interface{}) (bool, string))
	}
	this.metadataFunc[name] = f
	if len(this.cache) > 0 {
		for _, elem := range this.cache {
			ok1, v := f(elem)
			if ok1 {
				this.globalMetadata.KeyCount.Counts[name]++
				if v != "" {
					vCount, ok2 := this.globalMetadata.ValueCount[name]
					if !ok2 {
						vCount = &l8api.L8Count{}
						vCount.Counts = make(map[string]int32)
						this.globalMetadata.ValueCount[name] = vCount
					}
					vCount.Counts[v]++
				}
			}
		}
	}
}
