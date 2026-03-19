package plugin

import (
	"context"
	"log/slog"

	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/connect"
)

type HandleFunc func(ctx context.Context, req *connect.Message) (*connect.Message, error)

type Plugin interface {
	Name() string
	Serve(ctx context.Context, handle HandleFunc) error
	Send(ctx context.Context, req *connect.Message) (*connect.Message, error)
}

type Dependencies struct {
	Logger *slog.Logger
	Config *config.Config
}

type Factory func(deps Dependencies) (Plugin, error)

type Registration struct {
	Key     string
	Enabled func(cfg *config.Config) bool
	Build   Factory
}
