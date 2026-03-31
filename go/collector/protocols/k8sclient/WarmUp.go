package k8sclient

import (
	"fmt"
	"sort"

	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

func CacheSpecsFromBootModels() ([]*CacheSpec, error) {
	return CacheSpecsFromPollarisModels(boot.GetAllPolarisModels())
}

func CacheSpecsFromPollarisModels(models []*l8tpollaris.L8Pollaris) ([]*CacheSpec, error) {
	unique := make(map[string]*CacheSpec)
	for _, model := range models {
		if model == nil {
			continue
		}
		for _, poll := range model.Polling {
			if poll == nil || poll.Protocol != l8tpollaris.L8PProtocol_L8PKubernetesAPI {
				continue
			}
			spec, err := ParseCacheSpec(poll.What, poll)
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", model.Name, poll.Name, err)
			}
			key := spec.GVR + "::" + spec.Namespace
			if _, ok := unique[key]; !ok {
				unique[key] = spec
			}
		}
	}

	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]*CacheSpec, 0, len(keys))
	for _, key := range keys {
		result = append(result, unique[key])
	}
	return result, nil
}
