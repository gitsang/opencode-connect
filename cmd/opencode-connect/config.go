package main

import (
	"time"
)

type Config struct {
	Log struct {
		Handlers struct {
			Default string `json:"default" yaml:"default"`
		} `json:"handlers" yaml:"handlers"`
		Providers map[string][]LogConfig `json:"providers" yaml:"providers"`
	} `json:"log" yaml:"log"`
	Plugins  PluginsConfig  `json:"plugins" yaml:"plugins"`
	Opencode OpencodeConfig `json:"opencode" yaml:"opencode"`
}

type PluginsConfig struct {
	ChatAPI    ChatAPIPluginConfig `json:"chatapi" yaml:"chatapi"`
	UME        PluginConfig        `json:"ume" yaml:"ume"`
	Mattermost PluginConfig        `json:"mattermost" yaml:"mattermost"`
}

type PluginConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled" default:"false" usage:"enable this plugin"`
}

type ChatAPIPluginConfig struct {
	Enabled      bool          `json:"enabled" yaml:"enabled" default:"true" usage:"enable Chat API plugin"`
	Listen       string        `json:"listen" yaml:"listen" default:":8192" usage:"HTTP server listen address"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" default:"15s" usage:"HTTP read timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" default:"300s" usage:"HTTP write timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout" default:"60s" usage:"HTTP idle timeout"`
}

type OpencodeConfig struct {
	BaseURL        string            `json:"base_url" yaml:"base_url" default:"http://127.0.0.1:4096" usage:"opencode server base URL"`
	Password       string            `json:"password" yaml:"password" usage:"opencode server password"`
	PasswordHeader string            `json:"password_header" yaml:"password_header" default:"Authorization" usage:"header key for password authentication"`
	PasswordScheme string            `json:"password_scheme" yaml:"password_scheme" default:"Bearer" usage:"header auth scheme, empty means raw password"`
	PromptTimeout  time.Duration     `json:"prompt_timeout" yaml:"prompt_timeout" default:"5m" usage:"prompt timeout"`
	ModelAliases   map[string]string `json:"model_aliases" yaml:"model_aliases" usage:"alias to provider/model, e.g. gpt-5.4: openai/gpt-5.4"`
	ExtraHeaders   map[string]any    `json:"extra_headers" yaml:"extra_headers" usage:"extra request headers"`
}
