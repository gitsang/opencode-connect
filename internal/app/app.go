package app

import (
	"io"
	"log/slog"
	"os"

	"github.com/gitsang/logi"
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
