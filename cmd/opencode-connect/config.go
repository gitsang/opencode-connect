package main

type Config struct {
	Log struct {
		Handlers struct {
			Default string
		}
		Providers map[string][]LogConfig
	}
	Plugins struct {
		ChatAPI struct {
			Enabled bool   `default:"true" usage:"enable Chat API plugin"`
			Listen  string `default:":8192" usage:"HTTP server listen address"`
		}
	}
	Opencode struct {
		BaseURL  string `default:"http://127.0.0.1:4096" usage:"opencode server base URL"`
		Username string `default:"opencode" usage:"opencode server username"`
		Password string `usage:"opencode server password"`
	}
}
