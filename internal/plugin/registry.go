package plugin

import (
	"fmt"
)

func BuildEnabledPlugins(deps Dependencies) ([]Plugin, error) {
	if deps.Logger == nil {
		return nil, fmt.Errorf("plugin dependencies.logger is required")
	}

	registrations := []Registration{
		{
			Key:     "chatapi",
			Enabled: func(deps Dependencies) bool { return deps.EnableChatAPI },
			Build: func(deps Dependencies) (Plugin, error) {
				return NewChatAPI(deps.Logger, deps.ChatAPI), nil
			},
		},
		{
			Key:     "ume",
			Enabled: func(deps Dependencies) bool { return deps.EnableUME },
			Build:   unsupportedFactory("UME"),
		},
		{
			Key:     "mattermost",
			Enabled: func(deps Dependencies) bool { return deps.EnableMattermost },
			Build:   unsupportedFactory("Mattermost"),
		},
	}

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

func unsupportedFactory(pluginName string) Factory {
	return func(_ Dependencies) (Plugin, error) {
		return nil, fmt.Errorf("%s plugin is not implemented", pluginName)
	}
}
