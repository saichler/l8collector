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

// Package pollaris provides the core polling configuration management service
// for the Layer 8 ecosystem. It manages L8Pollaris configurations that define
// how data should be collected from various targets using different protocols
// (SNMP, SSH, RESTCONF, NETCONF, gRPC, Kubernetes, GraphQL).
package pollaris

import (
	"errors"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8services/go/services/dcache"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"

	"sync"
)

// PollarisCenter is the central management hub for polling configurations.
// It maintains a distributed cache of L8Pollaris objects and provides
// efficient lookup mechanisms by name, key, and group. The center supports
// hierarchical key-based lookups that fall back to less specific matches
// when exact matches are not found.
type PollarisCenter struct {
	// name2Poll is the distributed cache storing L8Pollaris objects indexed by name
	name2Poll ifs.IDistributedCache
	// key2Name maps composite keys (name+vendor+series+...) to pollaris names
	key2Name map[string]string
	// groups maps group names to their member pollaris entries (key -> name)
	groups map[string]map[string]string
	// log provides logging capabilities for the center
	log ifs.ILogger
	// mtx protects concurrent access to key2Name and groups maps
	mtx *sync.RWMutex
}

// newPollarisCenter creates and initializes a new PollarisCenter instance.
// It sets up the distributed cache, registers the L8Pollaris type with the
// introspector, and populates initial data from the service level agreement.
// The cache is created without synchronization (NoSync) for better performance.
func newPollarisCenter(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) *PollarisCenter {
	pc := &PollarisCenter{}
	pc.key2Name = make(map[string]string)
	pc.groups = make(map[string]map[string]string)
	pc.log = vnic.Resources().Logger()
	pc.mtx = &sync.RWMutex{}
	vnic.Resources().Introspector().Decorators().AddPrimaryKeyDecorator(&l8tpollaris.L8Pollaris{}, "Name")

	if sla.InitItems() != nil {
		vnic.Resources().Logger().Info("Initializing pollarisCenter with init elements ", len(sla.InitItems()))
		for _, element := range sla.InitItems() {
			pc.addForInit(element.(*l8tpollaris.L8Pollaris))
		}
	} else {
		vnic.Resources().Logger().Info("Initializing pollarisCenter with no init elements")
	}

	pc.name2Poll = dcache.NewDistributedCacheNoSync(ServiceName, ServiceArea, &l8tpollaris.L8Pollaris{}, sla.InitItems(),
		vnic, vnic.Resources())

	return pc
}

// getPollName retrieves the pollaris name associated with the given composite key.
// Returns the name and true if found, empty string and false otherwise.
// This method is thread-safe using a read lock.
func (this *PollarisCenter) getPollName(key string) (string, bool) {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	pollName, ok := this.key2Name[key]
	return pollName, ok
}

// cleanupGroupsLocked removes a key from all groups of the existing pollaris.
// Caller must hold this.mtx write lock.
func (this *PollarisCenter) cleanupGroupsLocked(pollrs *l8tpollaris.L8Pollaris, key string) {
	ePoll, _ := this.name2Poll.Get(pollrs)
	if ePoll == nil {
		return
	}
	existPoll, _ := ePoll.(*l8tpollaris.L8Pollaris)
	if existPoll.Groups != nil {
		for _, gName := range existPoll.Groups {
			gEntry := this.groups[gName]
			if gEntry != nil {
				delete(gEntry, key)
			}
		}
	}
}

// AddAll adds multiple L8Pollaris configurations to the center.
// Each pollaris is added using Post with isNotification=false.
func (this *PollarisCenter) AddAll(pollarises []*l8tpollaris.L8Pollaris) {
	for _, l8pollaris := range pollarises {
		this.Post(l8pollaris, false)
	}
}

// Post adds a new L8Pollaris configuration to the center.
// It validates that the pollaris has a name and polling information,
// removes any existing entry with the same key, and registers the new
// pollaris in the distributed cache and group mappings.
// Returns an error if validation fails.
func (this *PollarisCenter) Post(l8pollaris *l8tpollaris.L8Pollaris, isNotification bool) error {
	if l8pollaris.Name == "" {
		return errors.New("Pollaris does not contain a Name")
	}
	if l8pollaris.Polling == nil {
		return errors.New("Pollaris does not contain any polling information")
	}

	for _, poll := range l8pollaris.Polling {
		if poll.What == "" {
			return errors.New("Pollaris " + l8pollaris.Name + ": poll does not contain a What value")
		}
	}

	key := this.PollarisKey(l8pollaris)

	// Hold write lock across cleanup and re-add to prevent races where
	// concurrent readers see the key temporarily missing from key2Name.
	this.mtx.Lock()
	this.cleanupGroupsLocked(l8pollaris, key)
	this.key2Name[key] = l8pollaris.Name
	if l8pollaris.Groups != nil {
		for _, gName := range l8pollaris.Groups {
			gEntry, ok := this.groups[gName]
			if !ok {
				this.groups[gName] = make(map[string]string)
				gEntry = this.groups[gName]
			}
			gEntry[key] = l8pollaris.Name
		}
	}
	this.mtx.Unlock()

	this.name2Poll.Post(l8pollaris, isNotification)

	return nil
}

// addForInit adds a pollaris during initialization without triggering
// distributed cache events. This is used to populate the local indexes
// (key2Name and groups) from the initial data set.
func (this *PollarisCenter) addForInit(p *l8tpollaris.L8Pollaris) {
	key := this.PollarisKey(p)
	this.key2Name[key] = p.Name
	if p.Groups != nil {
		for _, gName := range p.Groups {
			gEntry, ok := this.groups[gName]
			if !ok {
				this.groups[gName] = make(map[string]string)
				gEntry = this.groups[gName]
			}
			gEntry[key] = p.Name
		}
	}
}

// Put updates an existing L8Pollaris configuration in the center.
// It performs the same validation as Post, removes any existing entry,
// and stores the updated pollaris in the distributed cache.
// Returns an error if validation fails.
func (this *PollarisCenter) Put(l8pollaris *l8tpollaris.L8Pollaris, isNotification bool) error {
	if l8pollaris.Name == "" {
		return errors.New("Pollaris does not contain a Name")
	}
	if l8pollaris.Polling == nil {
		return errors.New("Pollaris does not contain any polling information")
	}

	for _, poll := range l8pollaris.Polling {
		if poll.What == "" {
			return errors.New("Pollaris " + l8pollaris.Name + ": poll does not contain a What value")
		}
	}

	key := this.PollarisKey(l8pollaris)

	// Hold write lock across cleanup and re-add to prevent races where
	// concurrent readers see the key temporarily missing from key2Name.
	this.mtx.Lock()
	this.cleanupGroupsLocked(l8pollaris, key)
	this.key2Name[key] = l8pollaris.Name
	if l8pollaris.Groups != nil {
		for _, gName := range l8pollaris.Groups {
			gEntry, ok := this.groups[gName]
			if !ok {
				this.groups[gName] = make(map[string]string)
				gEntry = this.groups[gName]
			}
			gEntry[key] = l8pollaris.Name
		}
	}
	this.mtx.Unlock()

	this.name2Poll.Put(l8pollaris, isNotification)

	return nil
}

// PollarisKey generates a composite key for the given L8Pollaris.
// The key is constructed from name, vendor, series, family, software,
// hardware, and version fields, concatenated with '+' separators.
func (this *PollarisCenter) PollarisKey(l8pollaris *l8tpollaris.L8Pollaris) string {
	return pollarisKey(l8pollaris.Name, l8pollaris.Vendor, l8pollaris.Series, l8pollaris.Family, l8pollaris.Software, l8pollaris.Hardware, l8pollaris.Version)
}

// PollarisByName retrieves a L8Pollaris configuration by its name.
// Returns nil if the center is nil, the cache is nil, or no pollaris
// with the given name exists.
func (this *PollarisCenter) PollarisByName(name string) *l8tpollaris.L8Pollaris {
	if this == nil || this.name2Poll == nil {
		return nil
	}
	filter := &l8tpollaris.L8Pollaris{Name: name}
	p, _ := this.name2Poll.Get(filter)
	poll, _ := p.(*l8tpollaris.L8Pollaris)
	return poll
}

// PollarisByKey retrieves a L8Pollaris using a hierarchical key lookup.
// The args should be provided in order: name, vendor, series, family,
// software, hardware, version. If an exact match is not found, the method
// recursively tries with fewer components (falling back to less specific matches).
// Returns nil if no matching pollaris is found.
func (this *PollarisCenter) PollarisByKey(args ...string) *l8tpollaris.L8Pollaris {
	if args == nil || len(args) == 0 {
		return nil
	}
	if len(args) == 1 {
		pollName := this.key2Name[args[0]]
		filter := &l8tpollaris.L8Pollaris{Name: pollName}
		p, _ := this.name2Poll.Get(filter)
		poll, _ := p.(*l8tpollaris.L8Pollaris)
		return poll
	}
	buff := strings.New()
	buff.Add(args[0])
	for i := 1; i < len(args); i++ {
		addToKey(args[i], buff)
	}
	p, ok := this.getPollName(buff.String())
	if ok {
		filter := &l8tpollaris.L8Pollaris{Name: p}
		f, _ := this.name2Poll.Get(filter)
		poll, _ := f.(*l8tpollaris.L8Pollaris)
		return poll
	}
	return this.PollarisByKey(args[0 : len(args)-1]...)
}

// Poll retrieves a specific L8Poll (polling job) from a named pollaris.
// Returns nil if the pollaris is not found or if the job name doesn't exist
// in the pollaris's polling map.
func (this *PollarisCenter) Poll(pollarisName, jobName string) *l8tpollaris.L8Poll {
	l8pollaris := this.PollarisByName(pollarisName)
	if l8pollaris == nil {
		return nil
	}
	poll, ok := l8pollaris.Polling[jobName]
	if !ok {
		return nil
	}
	return poll
}

// Names returns all pollaris names belonging to the specified group.
// The vendor, series, family, software, hardware, and version parameters
// are reserved for future filtering but currently unused.
// Returns an empty slice if the group doesn't exist.
func (this *PollarisCenter) Names(groupName, vendor, series, family, software, hardware, version string) []string {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	result := make([]string, 0)
	group, ok := this.groups[groupName]
	if !ok {
		return result
	}
	for _, name := range group {
		result = append(result, name)
	}
	return result
}

// PollsByGroup retrieves all L8Pollaris configurations belonging to a group.
// It uses hierarchical key lookup for each member, allowing vendor/series/family
// specific overrides. Returns an empty slice if no matching pollarises are found.
func (this *PollarisCenter) PollsByGroup(groupName, vendor, series, family, software, hardware, version string) []*l8tpollaris.L8Pollaris {
	names := this.Names(groupName, vendor, series, family, software, hardware, version)
	result := make([]*l8tpollaris.L8Pollaris, 0)
	for _, name := range names {
		poll := this.PollarisByKey(name, vendor, series, family, software, hardware, version)
		if poll != nil {
			result = append(result, poll)
		}
	}
	return result
}

// Pollaris retrieves the PollarisCenter from the service registry.
// This is the main entry point for accessing the polling configuration
// management functionality. Returns nil if the service is not found.
func Pollaris(resource ifs.IResources) *PollarisCenter {
	sp, ok := resource.Services().ServiceHandler(ServiceName, ServiceArea)
	if !ok {
		return nil
	}
	return (sp.(*PollarisService)).pollarisCenter
}

// Poll is a convenience function to retrieve a specific polling job.
// It looks up the PollarisCenter, finds the named pollaris, and returns
// the specified poll. Returns an error if any step fails.
func Poll(pollarisName, pollName string, resources ifs.IResources) (*l8tpollaris.L8Poll, error) {
	pollarisCenter := Pollaris(resources)
	if pollarisCenter == nil {
		return nil, resources.Logger().Error("Cannot find Pollaris service")
	}
	l8pollaris := pollarisCenter.PollarisByName(pollarisName)
	if l8pollaris == nil {
		return nil, resources.Logger().Error("Cannot find Pollaris " + pollName)
	}
	poll, ok := l8pollaris.Polling[pollName]
	if !ok {
		return nil, resources.Logger().Error("Cannot find poll " + pollName + " in Pollaris " + l8pollaris.Name)
	}
	return poll, nil
}

// PollarisByKey is a convenience function to retrieve a pollaris using
// hierarchical key lookup. It wraps the PollarisCenter.PollarisByKey method
// and returns an error if the service or pollaris is not found.
func PollarisByKey(resources ifs.IResources, args ...string) (*l8tpollaris.L8Pollaris, error) {
	pollarisCenter := Pollaris(resources)
	if pollarisCenter == nil {
		return nil, resources.Logger().Error("Cannot find Pollaris service")
	}
	p := pollarisCenter.PollarisByKey(args...)
	if p == nil {
		return nil, resources.Logger().Error("Cannot find Pollaris for keys ", args)
	}
	return p, nil
}

// PollarisByGroup is a convenience function to retrieve all pollarises
// belonging to a group. It wraps the PollarisCenter.PollsByGroup method
// and returns an error if the service is not found or no pollarises exist
// for the specified group.
func PollarisByGroup(resources ifs.IResources, groupName, vendor, series, family, software, hardware, version string) ([]*l8tpollaris.L8Pollaris, error) {
	pollarisCenter := Pollaris(resources)
	if pollarisCenter == nil {
		return nil, resources.Logger().Error("Cannot find Pollaris service")
	}
	p := pollarisCenter.PollsByGroup(groupName, vendor, series, family, software, hardware, version)
	if p == nil {
		return nil, resources.Logger().Error("Cannot find Pollaris for group ", groupName)
	}
	return p, nil
}
