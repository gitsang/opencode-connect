package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gitsang/configer"
	"github.com/gitsang/opencode-connect/internal/connect"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/plugin"
	"github.com/gitsang/opencode-connect/internal/session"
	"github.com/spf13/cobra"
	ocsdk "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
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
	// Load configuration
	var c Config
	err := cfger.Load(&c, rootFlags.ConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// Setup log handler
	logHandlers, err := BuildLogHandlers(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// Preparing
	logger := slog.New(logHandlers.Get(c.Log.Handlers.Default))
	logger.Debug("Preparing...",
		slog.Any("flags", rootFlags),
		slog.Any("config", c),
		slog.String("pid", fmt.Sprintf("%d", os.Getpid())),
	)

	sdkOpts := []option.RequestOption{
		option.WithBaseURL(c.Opencode.BaseURL),
	}
	if c.Opencode.Username != "" || c.Opencode.Password != "" {
		creds := fmt.Sprintf("%s:%s", c.Opencode.Username, c.Opencode.Password)
		authValue := "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
		sdkOpts = append(sdkOpts, option.WithHeader("Authorization", authValue))
	}
	sdkClient := ocsdk.NewClient(sdkOpts...)

	opencodeClient := opencode.NewClient(sdkClient)

	conn := connect.New(
		opencodeClient,
		session.NewMemoryStore(),
	)

	plugins, err := plugin.BuildEnabledPlugins(plugin.Dependencies{
		Logger:        logger,
		EnableChatAPI: c.Plugins.ChatAPI.Enabled,
		ChatAPI: plugin.ChatAPIConfig{
			Listen: c.Plugins.ChatAPI.Listen,
		},
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
