package plugin

import (
	"context"
	"log/slog"

	"github.com/gitsang/opencode-connect/internal/connect"
)

type HandleFunc func(ctx context.Context, req *connect.Message) (*connect.Message, error)

type Plugin interface {
	Name() string
	Serve(ctx context.Context, handle HandleFunc) error
	Send(ctx context.Context, req *connect.Message) (*connect.Message, error)
}

type Infrastructure struct {
	Logger *slog.Logger
}

type Factory func(name string, configRaw any, infra Infrastructure) (Plugin, error)

type Registration struct {
	Key   string
	Build Factory
}
