package config

import (
	"fmt"
	"time"

	"github.com/gitsang/configer"
	"github.com/spf13/cobra"
)

type Config struct {
	Server   ServerConfig   `json:"server" yaml:"server"`
	Opencode OpencodeConfig `json:"opencode" yaml:"opencode"`
	Log      LogConfig      `json:"log" yaml:"log"`
}

type ServerConfig struct {
	Listen       string        `json:"listen" yaml:"listen" default:":8080" usage:"HTTP server listen address"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" default:"15s" usage:"HTTP read timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" default:"300s" usage:"HTTP write timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout" default:"60s" usage:"HTTP idle timeout"`
}

type OpencodeConfig struct {
	BaseURL         string                 `json:"base_url" yaml:"base_url" default:"http://127.0.0.1:4096" usage:"opencode server base URL"`
	Password        string                 `json:"password" yaml:"password" usage:"opencode server password"`
	PasswordHeader  string                 `json:"password_header" yaml:"password_header" default:"Authorization" usage:"header key for password authentication"`
	PasswordScheme  string                 `json:"password_scheme" yaml:"password_scheme" default:"Bearer" usage:"header auth scheme, empty means raw password"`
	Directory       string                 `json:"directory" yaml:"directory" default:"." usage:"working directory for opencode sessions"`
	PromptTimeout   time.Duration          `json:"prompt_timeout" yaml:"prompt_timeout" default:"5m" usage:"prompt timeout"`
	DefaultProvider string                 `json:"default_provider" yaml:"default_provider" default:"" usage:"default provider ID"`
	DefaultModel    string                 `json:"default_model" yaml:"default_model" default:"" usage:"default model ID"`
	ModelAliases    map[string]string      `json:"model_aliases" yaml:"model_aliases" usage:"alias to provider/model, e.g. gpt-5.4: openai/gpt-5.4"`
	SessionTitleTpl string                 `json:"session_title_tpl" yaml:"session_title_tpl" default:"chat-session-{session_id}" usage:"new opencode session title template"`
	ExtraHeaders    map[string]interface{} `json:"extra_headers" yaml:"extra_headers" usage:"extra request headers"`
}

type LogConfig struct {
	Level     string `json:"level" yaml:"level" default:"info" usage:"log level"`
	Format    string `json:"format" yaml:"format" default:"console" usage:"log format: console/json/text"`
	Color     bool   `json:"color" yaml:"color" default:"true" usage:"enable color output"`
	Verbosity int    `json:"verbosity" yaml:"verbosity" default:"1" usage:"log source verbosity"`
}

func Load(cmd *cobra.Command, files []string) (*Config, error) {
	cfg := new(Config)

	cfger := configer.New(
		configer.WithTemplate(new(Config)),
		configer.WithEnvBind(
			configer.WithEnvPrefix("OPENCODE_CONNECT"),
			configer.WithEnvDelim("_"),
		),
		configer.WithFlagBind(
			configer.WithCommand(cmd),
			configer.WithFlagPrefix("opencode-connect"),
			configer.WithFlagDelim("."),
		),
	)

	if err := cfger.Load(cfg, files...); err != nil {
		return nil, err
	}

	if cfg.Opencode.BaseURL == "" {
		return nil, fmt.Errorf("opencode.base_url is required")
	}

	if cfg.Opencode.Directory == "" {
		cfg.Opencode.Directory = "."
	}

	if cfg.Opencode.SessionTitleTpl == "" {
		cfg.Opencode.SessionTitleTpl = "chat-session-{session_id}"
	}

	if cfg.Opencode.ModelAliases == nil {
		cfg.Opencode.ModelAliases = map[string]string{}
	}

	if cfg.Opencode.ExtraHeaders == nil {
		cfg.Opencode.ExtraHeaders = map[string]interface{}{}
	}

	return cfg, nil
}
