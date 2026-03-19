package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gitsang/opencode-connect/internal/app"
	corechatapi "github.com/gitsang/opencode-connect/internal/chatapi"
	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/session"
)

type ChatAPI struct {
	logger         *slog.Logger
	opencodeClient *opencode.Client
	cfg            *config.Config
}

func NewChatAPI(logger *slog.Logger, opencodeClient *opencode.Client, cfg *config.Config) *ChatAPI {
	return &ChatAPI{
		logger:         logger,
		opencodeClient: opencodeClient,
		cfg:            cfg,
	}
}

func (p *ChatAPI) Name() string {
	return "chatapi"
}

func (p *ChatAPI) Run(ctx context.Context) error {
	sessionStore := session.NewMemoryStore()
	chatApp := corechatapi.NewChatAPI(p.opencodeClient, sessionStore, p.cfg)

	serverConfig := p.cfg.Plugins.ChatAPI
	httpServer := &http.Server{
		Addr:         serverConfig.Listen,
		Handler:      app.NewHTTPHandler(chatApp),
		ReadTimeout:  serverConfig.ReadTimeout,
		WriteTimeout: serverConfig.WriteTimeout,
		IdleTimeout:  serverConfig.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		p.logger.Info("chatapi plugin started", "listen", serverConfig.Listen)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		err := <-errCh
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server stopped with error: %w", err)
		}

		p.logger.Info("chatapi plugin stopped")
		return nil
	case err := <-errCh:
		if err == nil || err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
