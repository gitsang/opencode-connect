package plugin

import (
	"context"

	"github.com/gitsang/opencode-connect/internal/connect"
)

type HandleFunc func(ctx context.Context, req *connect.Message) (*connect.Message, error)

type Plugin interface {
	Name() string
	Serve(ctx context.Context, handle HandleFunc) error
	Send(ctx context.Context, req *connect.Message) (*connect.Message, error)
}

const (
	InfraLogger         = "logger"
	InfraOpencodeClient = "opencode_client"
	InfraSessionStore   = "session_store"
)

type Factory func(name string, configRaw any, infras map[string]any) (Plugin, error)

type Registration struct {
	Key   string
	Build Factory
}
