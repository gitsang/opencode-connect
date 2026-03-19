package plugin

import (
	"fmt"

	"github.com/gitsang/opencode-connect/internal/config"
)

func BuildEnabledPlugins(deps Dependencies) ([]Plugin, error) {
	if deps.Logger == nil {
		return nil, fmt.Errorf("plugin dependencies.logger is required")
	}
	if deps.Config == nil {
		return nil, fmt.Errorf("plugin dependencies.config is required")
	}

	registrations := []Registration{
		{
			Key:     "chatapi",
			Enabled: func(cfg *config.Config) bool { return cfg.Plugins.ChatAPI.Enabled },
			Build: func(deps Dependencies) (Plugin, error) {
				return NewChatAPI(deps.Logger, deps.Config), nil
			},
		},
		{
			Key:     "ume",
			Enabled: func(cfg *config.Config) bool { return cfg.Plugins.UME.Enabled },
			Build:   unsupportedFactory("UME"),
		},
		{
			Key:     "mattermost",
			Enabled: func(cfg *config.Config) bool { return cfg.Plugins.Mattermost.Enabled },
			Build:   unsupportedFactory("Mattermost"),
		},
	}

	plugins := make([]Plugin, 0, len(registrations))
	for _, registration := range registrations {
		if !registration.Enabled(deps.Config) {
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
