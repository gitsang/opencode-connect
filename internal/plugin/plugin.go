package plugin

import (
	"context"
	"log/slog"

	"github.com/gitsang/opencode-connect/internal/connect"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/session"
)

type HandleFunc func(ctx context.Context, req *connect.Message) (*connect.Message, error)

type Plugin interface {
	Name() string
	Serve(ctx context.Context, handle HandleFunc) error
	Send(ctx context.Context, req *connect.Message) (*connect.Message, error)
}

type Dependencies struct {
	Logger           *slog.Logger
	OpencodeClient   *opencode.Client
	SessionStore     session.Store
	EnableChatAPI    bool
	EnableUME        bool
	EnableMattermost bool
	ChatAPI          ChatAPIConfig
}

type ChatAPIConfig struct {
	Listen string
}

type Factory func(deps Dependencies) (Plugin, error)

type Registration struct {
	Key     string
	Enabled func(deps Dependencies) bool
	Build   Factory
}
