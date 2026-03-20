package plugin

import (
	"fmt"
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

func GetRegistration(key string) (Registration, bool) {
	registrationMu.RLock()
	defer registrationMu.RUnlock()

	registration, ok := registrationMap[key]
	return registration, ok
}
