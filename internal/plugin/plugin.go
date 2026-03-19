package plugin

import (
	"context"
	"log/slog"
	"time"

	"github.com/gitsang/opencode-connect/internal/connect"
)

type HandleFunc func(ctx context.Context, req *connect.Message) (*connect.Message, error)

type Plugin interface {
	Name() string
	Serve(ctx context.Context, handle HandleFunc) error
	Send(ctx context.Context, req *connect.Message) (*connect.Message, error)
}

type Dependencies struct {
	Logger           *slog.Logger
	EnableChatAPI    bool
	EnableUME        bool
	EnableMattermost bool
	ChatAPI          ChatAPIConfig
}

type ChatAPIConfig struct {
	Listen       string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type Factory func(deps Dependencies) (Plugin, error)

type Registration struct {
	Key     string
	Enabled func(deps Dependencies) bool
	Build   Factory
}
