package plugin

import (
	"fmt"
	"sync"
)

var (
	registrationMu  sync.RWMutex
	pluginFactoryMap = map[string]PluginFactory{}
)

func Register(registration PluginFactory) {
	if registration.Key == "" {
		panic("plugin registration key is required")
	}
	if registration.Build == nil {
		panic(fmt.Sprintf("plugin %s build function is required", registration.Key))
	}

	registrationMu.Lock()
	defer registrationMu.Unlock()
	pluginFactoryMap[registration.Key] = registration
}

func GetRegistration(key string) (PluginFactory, bool) {
	registrationMu.RLock()
	defer registrationMu.RUnlock()

	registration, ok := pluginFactoryMap[key]
	return registration, ok
}
