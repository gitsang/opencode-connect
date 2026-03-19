package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gitsang/opencode-connect/internal/app"
	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/plugin"
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
			logger.Error("preparing...", slog.Any("cfg", cfg))

			opencodeClient, err := opencode.NewClient(cfg)
			if err != nil {
				return err
			}

			plugins, err := plugin.BuildEnabledPlugins(plugin.Dependencies{
				Logger:         logger,
				OpencodeClient: opencodeClient,
				Config:         cfg,
			})
			if err != nil {
				return err
			}

			runCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			if err := plugin.Run(runCtx, plugins); err != nil {
				return err
			}

			slog.Info("all plugins stopped")
			return nil
		},
	}

	cmd.PersistentFlags().StringSliceVarP(&configFiles, "config", "c", nil, "Config file path (repeatable)")
	return cmd
}
