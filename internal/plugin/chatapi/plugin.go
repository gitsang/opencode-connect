package chatapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gitsang/opencode-connect/internal/connect"
	coreplugin "github.com/gitsang/opencode-connect/internal/plugin"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen string `yaml:"listen"`
}

type Plugin struct {
	name   string
	logger *slog.Logger
	cfg    Config
}

func init() {
	constructor := func(name string, configRaw any, infra coreplugin.Infrastructure) (coreplugin.Plugin, error) {
		cfg := defaultConfig()
		configBytes, err := yaml.Marshal(configRaw)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(configBytes, &cfg); err != nil {
			return nil, err
		}

		if infra.Logger == nil {
			return nil, fmt.Errorf("chatapi infrastructure logger is required")
		}

		return New(name, infra.Logger, cfg), nil
	}

	coreplugin.Register(coreplugin.PluginFactory{
		Name:      "chatapi",
		Construct: constructor,
	})
}

func defaultConfig() Config {
	return Config{Listen: ":8192"}
}

func New(name string, logger *slog.Logger, cfg Config) *Plugin {
	return &Plugin{
		name:   name,
		logger: logger.With("plugin_name", name, "plugin_type", "chatapi"),
		cfg:    cfg,
	}
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Serve(ctx context.Context, handle coreplugin.HandleFunc) error {
	if handle == nil {
		return fmt.Errorf("chatapi handle is required")
	}

	serverConfig := p.cfg
	server := &http.Server{
		Addr:    serverConfig.Listen,
		Handler: p.newHTTPHandler(handle),
	}

	errCh := make(chan error, 1)
	go func() {
		p.logger.Info("chatapi plugin started", "listen", serverConfig.Listen)
		errCh <- server.ListenAndServe()
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err := <-errCh
	if err == nil || err == http.ErrServerClosed {
		p.logger.Info("chatapi plugin stopped")
		return nil
	}

	return fmt.Errorf("listen chatapi http server: %w", err)
}

func (p *Plugin) Send(_ context.Context, _ *connect.Message) (*connect.Message, error) {
	return nil, fmt.Errorf("chatapi plugin does not support proactive send")
}

func (p *Plugin) newHTTPHandler(handle coreplugin.HandleFunc) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		var req connect.Message
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}

		resp, err := handle(r.Context(), &req)
		if err != nil {
			status := http.StatusInternalServerError
			var connectError *connect.Error
			if errors.As(err, &connectError) {
				status = connectError.StatusCode
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
