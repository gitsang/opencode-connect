package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/gitsang/logi"
	"github.com/gitsang/opencode-connect/internal/chat"
	"github.com/gitsang/opencode-connect/internal/config"
)

func NewLogger(cfg *config.Config) *slog.Logger {
	handler := logi.NewHandler(logi.HandlerOptions{
		Format:    cfg.Log.Format,
		Color:     cfg.Log.Color,
		Level:     cfg.Log.Level,
		Verbosity: cfg.Log.Verbosity,
		Writers:   []io.Writer{os.Stdout},
		Attrs: map[string]interface{}{
			"service": "opencode-connect",
		},
	})

	return slog.New(handler)
}

func NewHTTPHandler(chatApp chat.ChatApp) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"error": "method not allowed",
			})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		var req chat.MessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": "invalid json",
			})
			return
		}

		resp, err := chatApp.HandleMessage(r.Context(), req)
		if err != nil {
			status := http.StatusInternalServerError
			if err, ok := err.(*chat.HTTPError); ok {
				status = err.StatusCode
			}

			writeJSON(w, status, map[string]interface{}{
				"error": err.Error(),
			})
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
