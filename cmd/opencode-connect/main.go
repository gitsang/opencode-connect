package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gitsang/configer"
	"github.com/gitsang/logi"
	"github.com/gitsang/opencode-connect/internal/connect"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/plugin"
	"github.com/gitsang/opencode-connect/internal/session"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "opencode-connect",
	Short: "Connect opencode to chat apps",
	RunE:  Run,
}

var rootFlags = struct {
	ConfigFile string
}{}

var cfger *configer.Configer

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootFlags.ConfigFile, "config-file", "c",
		"conf/config.yaml", "specify the config file.")

	cfger = configer.New(
		configer.WithTemplate((*Config)(nil)),
		configer.WithEnvBind(
			configer.WithEnvPrefix("OPENCODE_CONNECT"),
			configer.WithEnvDelim("_"),
		),
		configer.WithFlagBind(
			configer.WithCommand(rootCmd),
			configer.WithFlagPrefix("occ"),
			configer.WithFlagDelim("."),
		),
	)
}

func Run(cmd *cobra.Command, _ []string) error {
	// Create context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Load configuration
	var c Config
	err := cfger.Load(&c, rootFlags.ConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	handler := logi.NewHandler(logi.HandlerOptions{
		Format:    cfg.Log.Format,
		Color:     cfg.Log.Color,
		Level:     cfg.Log.Level,
		Verbosity: cfg.Log.Verbosity,
		Writers:   []io.Writer{os.Stdout},
		Attrs: map[string]interface{}{
			"service": "opencode-connect",
		},
	})

	logger := slog.New(handler)

	opencodeClient, err := opencode.NewClient(cfg)
	if err != nil {
		return err
	}

	conn := connect.New(opencodeClient, session.NewMemoryStore(), cfg)

	plugins, err := plugin.BuildEnabledPlugins(plugin.Dependencies{
		Logger: logger,
		Config: cfg,
	})
	if err != nil {
		return err
	}

	runCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := plugin.Run(runCtx, plugins, conn.Handle); err != nil {
		return err
	}

	slog.Info("all plugins stopped")
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
