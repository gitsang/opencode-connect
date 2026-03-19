package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gitsang/opencode-connect/internal/connect"
)

type ChatAPI struct {
	logger *slog.Logger
	cfg    ChatAPIConfig
}

func NewChatAPI(logger *slog.Logger, cfg ChatAPIConfig) *ChatAPI {
	return &ChatAPI{
		logger: logger,
		cfg:    cfg,
	}
}

func (p *ChatAPI) Name() string {
	return "chatapi"
}

func (p *ChatAPI) Serve(ctx context.Context, handle HandleFunc) error {
	if handle == nil {
		return fmt.Errorf("chatapi handle is required")
	}

	serverConfig := p.cfg
	server := &http.Server{
		Addr:         serverConfig.Listen,
		Handler:      p.newHTTPHandler(handle),
		ReadTimeout:  serverConfig.ReadTimeout,
		WriteTimeout: serverConfig.WriteTimeout,
		IdleTimeout:  serverConfig.IdleTimeout,
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

func (p *ChatAPI) Send(_ context.Context, _ *connect.Message) (*connect.Message, error) {
	return nil, fmt.Errorf("chatapi plugin does not support proactive send")
}

func (p *ChatAPI) newHTTPHandler(handle HandleFunc) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "method not allowed"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		var req connect.Message
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid json"})
			return
		}

		resp, err := handle(r.Context(), &req)
		if err != nil {
			status := http.StatusInternalServerError
			var connectError *connect.Error
			if errors.As(err, &connectError) {
				status = connectError.StatusCode
			}
			writeJSON(w, status, map[string]interface{}{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, resp)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
