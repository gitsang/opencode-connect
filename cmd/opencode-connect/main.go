package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gitsang/configer"
	"github.com/gitsang/opencode-connect/internal/connect"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/plugin"
	_ "github.com/gitsang/opencode-connect/internal/plugin/chatapi"
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

	sessionStore := session.NewMemoryStore()

	connector := connect.New(opencodeClient)

	deps := plugin.Dependencies{
		Logger:         logger,
		OpencodeClient: opencodeClient,
		SessionStore:   sessionStore,
		EnableChatAPI:  c.Plugins.ChatAPI.Enabled,
		ChatAPI: plugin.ChatAPIConfig{
			Listen: c.Plugins.ChatAPI.Listen,
		},
	}

	plugins := make([]plugin.Plugin, 0, 1)
	if deps.EnableChatAPI {
		registration, ok := plugin.GetRegistration("chatapi")
		if !ok {
			return fmt.Errorf("plugin chatapi is not registered")
		}

		currentPlugin, err := registration.Build(deps)
		if err != nil {
			return fmt.Errorf("build chatapi plugin: %w", err)
		}
		if currentPlugin == nil {
			return fmt.Errorf("build chatapi plugin: factory returned nil plugin")
		}
		plugins = append(plugins, currentPlugin)
	}

	if len(plugins) == 0 {
		return fmt.Errorf("no enabled plugins configured")
	}

	runCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, len(plugins))
	var waitGroup sync.WaitGroup

	for _, currentPlugin := range plugins {
		waitGroup.Add(1)
		go func(current plugin.Plugin) {
			defer waitGroup.Done()
			if serveErr := current.Serve(runCtx, connector.Handle); serveErr != nil {
				errCh <- fmt.Errorf("%s plugin failed: %w", current.Name(), serveErr)
				cancel()
			}
		}(currentPlugin)
	}

	done := make(chan struct{})
	go func() {
		waitGroup.Wait()
		close(done)
	}()

	select {
	case runErr := <-errCh:
		<-done
		return runErr
	case <-runCtx.Done():
		<-done
		select {
		case runErr := <-errCh:
			return runErr
		default:
			return nil
		}
	case <-done:
		select {
		case runErr := <-errCh:
			return runErr
		default:
			return nil
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
