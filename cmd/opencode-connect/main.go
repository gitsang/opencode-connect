package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gitsang/opencode-connect/internal/app"
	"github.com/gitsang/opencode-connect/internal/chatapi"
	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/session"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var configFiles []string

	cmd := &cobra.Command{
		Use:   "opencode-connect",
		Short: "Connect opencode to chat apps",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cmd, configFiles)
			if err != nil {
				return err
			}

			logger := app.NewLogger(cfg)
			slog.SetDefault(logger)

			opencodeClient, err := opencode.NewClient(cfg)
			if err != nil {
				return err
			}

			store := session.NewMemoryStore()
			chatApp := chatapi.NewChatAPI(opencodeClient, store, cfg)

			handler := app.NewHTTPHandler(chatApp)
			httpServer := &http.Server{
				Addr:         cfg.Server.Listen,
				Handler:      handler,
				ReadTimeout:  cfg.Server.ReadTimeout,
				WriteTimeout: cfg.Server.WriteTimeout,
				IdleTimeout:  cfg.Server.IdleTimeout,
			}

			errCh := make(chan error, 1)
			go func() {
				slog.Info("chat api server started", "listen", cfg.Server.Listen)
				errCh <- httpServer.ListenAndServe()
			}()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			select {
			case sig := <-sigCh:
				slog.Info("shutdown signal received", "signal", sig.String())
			case srvErr := <-errCh:
				if srvErr != nil && srvErr != http.ErrServerClosed {
					return srvErr
				}
				return nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			if err := httpServer.Shutdown(ctx); err != nil {
				return err
			}

			slog.Info("server stopped")
			return nil
		},
	}

	cmd.PersistentFlags().StringSliceVarP(&configFiles, "config", "c", nil, "Config file path (repeatable)")
	return cmd
}
