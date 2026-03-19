package plugin

import (
	"fmt"
	"sort"
	"sync"
)

var (
	registrationMu  sync.RWMutex
	registrationMap = map[string]Registration{}
)

func Register(registration Registration) {
	if registration.Key == "" {
		panic("plugin registration key is required")
	}
	if registration.Build == nil {
		panic(fmt.Sprintf("plugin %s build function is required", registration.Key))
	}
	if registration.Enabled == nil {
		panic(fmt.Sprintf("plugin %s enabled function is required", registration.Key))
	}

	registrationMu.Lock()
	defer registrationMu.Unlock()
	registrationMap[registration.Key] = registration
}

func BuildEnabledPlugins(deps Dependencies) ([]Plugin, error) {
	if deps.Logger == nil {
		return nil, fmt.Errorf("plugin dependencies.logger is required")
	}

	registrations := make([]Registration, 0, len(registrationMap))
	registrationMu.RLock()
	for _, registration := range registrationMap {
		registrations = append(registrations, registration)
	}
	registrationMu.RUnlock()
	sort.Slice(registrations, func(i int, j int) bool {
		return registrations[i].Key < registrations[j].Key
	})

	plugins := make([]Plugin, 0, len(registrations))
	for _, registration := range registrations {
		if !registration.Enabled(deps) {
			continue
		}

		currentPlugin, err := registration.Build(deps)
		if err != nil {
			return nil, fmt.Errorf("build %s plugin: %w", registration.Key, err)
		}
		if currentPlugin == nil {
			return nil, fmt.Errorf("build %s plugin: factory returned nil plugin", registration.Key)
		}

		plugins = append(plugins, currentPlugin)
	}

	return plugins, nil
}
