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

// Package resources provides a centralized container for shared application resources.
// It holds references to commonly used components like logger, registry, security provider,
// serializers, and configuration, enabling dependency injection across the application.
//
// Key features:
//   - Centralized resource management
//   - Type-safe component storage and retrieval
//   - Default configuration values for data size and queue limits
//   - Resource copying for creating child contexts
package resources

import (
	"reflect"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8sysconfig"
)

// DEFAULT_MAX_DATA_SIZE is the default maximum data size (50MB).
var DEFAULT_MAX_DATA_SIZE uint64 = 1024 * 1024 * 50

// DEFAULT_QUEUE_SIZE is the default queue capacity (100,000 entries).
var DEFAULT_QUEUE_SIZE uint64 = 100000

// Resources is the central container for application-wide shared components.
// It implements ifs.IResources for integration with Layer 8 services.
type Resources struct {
	logger       ifs.ILogger
	registry     ifs.IRegistry
	services     ifs.IServices
	security     ifs.ISecurityProvider
	dataListener ifs.IDatatListener
	serializers  map[ifs.SerializerMode]ifs.ISerializer
	config       *l8sysconfig.L8SysConfig
	introspector ifs.IIntrospector
}

// NewResources creates a new Resources container with the specified logger.
func NewResources(logger ifs.ILogger) ifs.IResources {
	r := &Resources{}
	r.logger = logger
	r.serializers = make(map[ifs.SerializerMode]ifs.ISerializer)
	return r
}

// AddService registers a new service with the system configuration.
func (this *Resources) AddService(serviceName string, serviceArea int32) {
	ifs.AddService(this.config, serviceName, serviceArea)
}

// Set stores a component by detecting its type via interface assertion.
// Supports IRegistry, IServices, ISecurityProvider, IDatatListener, ISerializer,
// L8SysConfig, and IIntrospector types.
func (this *Resources) Set(any interface{}) {
	if any == nil {
		return
	}
	registry, ok := any.(ifs.IRegistry)
	if ok {
		this.registry = registry
		return
	}

	services, ok := any.(ifs.IServices)
	if ok {
		this.services = services
		return
	}

	security, ok := any.(ifs.ISecurityProvider)
	if ok {
		this.security = security
		return
	}

	dataListener, ok := any.(ifs.IDatatListener)
	if ok {
		this.dataListener = dataListener
		return
	}

	serializer, ok := any.(ifs.ISerializer)
	if ok {
		this.serializers[serializer.Mode()] = serializer
	}

	config, ok := any.(*l8sysconfig.L8SysConfig)
	if ok {
		this.config = config
		return
	}

	introspector, ok := any.(ifs.IIntrospector)
	if ok {
		this.introspector = introspector
		return
	}
	v := reflect.ValueOf(any)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	this.logger.Error("Unknown Set type ", v.Type().Name())
}

// Copy copies all components from another Resources container.
func (this *Resources) Copy(other ifs.IResources) {
	this.registry = other.Registry()
	this.security = other.Security()
	this.services = other.Services()
	this.serializers[ifs.BINARY] = other.Serializer(ifs.BINARY)
	this.introspector = other.Introspector()
	this.dataListener = other.DataListener()
	this.config = other.SysConfig()
}

// Registry returns the type registry component.
func (this *Resources) Registry() ifs.IRegistry {
	return this.registry
}
// Services returns the services manager component.
func (this *Resources) Services() ifs.IServices {
	return this.services
}
// Security returns the security provider component.
func (this *Resources) Security() ifs.ISecurityProvider {
	return this.security
}
// DataListener returns the data listener component.
func (this *Resources) DataListener() ifs.IDatatListener {
	return this.dataListener
}
// Serializer returns the serializer for the specified mode.
func (this *Resources) Serializer(mode ifs.SerializerMode) ifs.ISerializer {
	return this.serializers[mode]
}
// Logger returns the logger component.
func (this *Resources) Logger() ifs.ILogger {
	return this.logger
}
// SysConfig returns the system configuration.
func (this *Resources) SysConfig() *l8sysconfig.L8SysConfig {
	return this.config
}
// Introspector returns the introspector component.
func (this *Resources) Introspector() ifs.IIntrospector {
	return this.introspector
}
