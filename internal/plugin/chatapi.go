package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gitsang/opencode-connect/internal/app"
	corechatapi "github.com/gitsang/opencode-connect/internal/chatapi"
	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/session"
)

type ChatAPI struct {
	logger   *slog.Logger
	cfg      *config.Config
	server   *http.Server
	stopOnce sync.Once
}

func NewChatAPI(logger *slog.Logger, opencodeClient *opencode.Client, cfg *config.Config) *ChatAPI {
	sessionStore := session.NewMemoryStore()
	chatApp := corechatapi.NewChatAPI(opencodeClient, sessionStore, cfg)
	serverConfig := cfg.Plugins.ChatAPI

	return &ChatAPI{
		logger: logger,
		cfg:    cfg,
		server: &http.Server{
			Addr:         serverConfig.Listen,
			Handler:      app.NewHTTPHandler(chatApp),
			ReadTimeout:  serverConfig.ReadTimeout,
			WriteTimeout: serverConfig.WriteTimeout,
			IdleTimeout:  serverConfig.IdleTimeout,
		},
	}
}

func (p *ChatAPI) Name() string {
	return "chatapi"
}

func (p *ChatAPI) Start(ctx context.Context) error {
	serverConfig := p.cfg.Plugins.ChatAPI

	errCh := make(chan error, 1)
	go func() {
		p.logger.Info("chatapi plugin started", "listen", serverConfig.Listen)
		errCh <- p.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		if err == nil || err == http.ErrServerClosed {
			p.logger.Info("chatapi plugin stopped")
			return nil
		}
		return fmt.Errorf("listen chatapi http server: %w", err)
	}
}

func (p *ChatAPI) Stop(ctx context.Context) error {
	var stopErr error
	p.stopOnce.Do(func() {
		if err := p.server.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
			stopErr = fmt.Errorf("shutdown http server: %w", err)
			return
		}

		p.logger.Info("chatapi plugin stopped")
	})

	return stopErr
}
