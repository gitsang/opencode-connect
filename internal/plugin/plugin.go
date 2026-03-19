package plugin

import (
	"context"
	"log/slog"

	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/opencode"
)

type Plugin interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Dependencies struct {
	Logger         *slog.Logger
	OpencodeClient *opencode.Client
	Config         *config.Config
}

type Factory func(deps Dependencies) (Plugin, error)

type Registration struct {
	Key     string
	Enabled func(cfg *config.Config) bool
	Build   Factory
}
