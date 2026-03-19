package plugin

import (
	"fmt"
	"log/slog"

	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/opencode"
)

type Factory func(logger *slog.Logger, opencodeClient *opencode.Client, cfg *config.Config) (Plugin, error)

func BuildEnabledPlugins(logger *slog.Logger, opencodeClient *opencode.Client, cfg *config.Config) ([]Plugin, error) {
	factories := map[string]Factory{
		"chatapi": func(logger *slog.Logger, opencodeClient *opencode.Client, cfg *config.Config) (Plugin, error) {
			return NewChatAPI(logger, opencodeClient, cfg), nil
		},
		"ume":        unsupportedFactory("UME"),
		"mattermost": unsupportedFactory("Mattermost"),
	}

	enabled := map[string]bool{
		"chatapi":    cfg.Plugins.ChatAPI.Enabled,
		"ume":        cfg.Plugins.UME.Enabled,
		"mattermost": cfg.Plugins.Mattermost.Enabled,
	}

	pluginOrder := []string{"chatapi", "ume", "mattermost"}
	plugins := make([]Plugin, 0, len(enabled))
	for _, name := range pluginOrder {
		isEnabled := enabled[name]
		if !isEnabled {
			continue
		}

		factory, ok := factories[name]
		if !ok {
			return nil, fmt.Errorf("plugin factory not found: %s", name)
		}

		plugin, err := factory(logger, opencodeClient, cfg)
		if err != nil {
			return nil, fmt.Errorf("build %s plugin: %w", name, err)
		}
		if plugin == nil {
			return nil, fmt.Errorf("build %s plugin: factory returned nil plugin", name)
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

func unsupportedFactory(pluginName string) Factory {
	return func(_ *slog.Logger, _ *opencode.Client, _ *config.Config) (Plugin, error) {
		return nil, fmt.Errorf("%s plugin is not implemented", pluginName)
	}
}
