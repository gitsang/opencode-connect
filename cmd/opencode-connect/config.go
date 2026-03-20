package main

type Config struct {
	Log struct {
		Handlers struct {
			Default string
		}
		Providers map[string][]LogConfig
	}
	Plugins  map[string]any
	Opencode struct {
		BaseURL  string `default:"http://127.0.0.1:4096" usage:"opencode server base URL"`
		Username string `default:"opencode" usage:"opencode server username"`
		Password string `usage:"opencode server password"`
	}
}
