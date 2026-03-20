package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/gitsang/configer"
	"github.com/gitsang/opencode-connect/internal/connect"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/plugin"
	_ "github.com/gitsang/opencode-connect/internal/plugin/chatapi"
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

	// Dependency Injection
	opencodeClient := opencode.NewClient(
		c.Opencode.BaseURL,
		opencode.WithAuthentication(c.Opencode.Username, c.Opencode.Password),
	)
	sessionStore := session.NewMemoryStore()
	connector := connect.New(opencodeClient, sessionStore)

	infra := plugin.Infrastructure{
		Logger: logger,
	}

	plugins, err := buildPlugins(c.Plugins, infra)
	if err != nil {
		return err
	}

	runCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return plugin.Run(runCtx, plugins, connector.Handle)
}

func buildPlugins(configMap map[string]any, infra plugin.Infrastructure) ([]plugin.Plugin, error) {
	instanceNames := make([]string, 0, len(configMap))
	for instanceName := range configMap {
		instanceNames = append(instanceNames, instanceName)
	}
	sort.Strings(instanceNames)

	plugins := make([]plugin.Plugin, 0, len(instanceNames))
	for _, instanceName := range instanceNames {
		instanceConfigRaw := configMap[instanceName]
		instanceConfigMap, ok := instanceConfigRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("plugin %s config must be a map", instanceName)
		}
		if len(instanceConfigMap) != 1 {
			return nil, fmt.Errorf("plugin %s config must define exactly one plugin type", instanceName)
		}

		for instanceType, typeConfigRaw := range instanceConfigMap {
			if typeConfigRaw == nil {
				return nil, fmt.Errorf("plugin %s config is nil", instanceName)
			}

			registration, ok := plugin.GetRegistration(instanceType)
			if !ok {
				return nil, fmt.Errorf("unsupported plugin type %q for %q", instanceType, instanceName)
			}

			currentPlugin, err := registration.Build(instanceName, typeConfigRaw, infra)
			if err != nil {
				return nil, fmt.Errorf("build plugin %s (%s): %w", instanceName, instanceType, err)
			}
			if currentPlugin == nil {
				return nil, fmt.Errorf("build plugin %s (%s): factory returned nil plugin", instanceName, instanceType)
			}

			plugins = append(plugins, currentPlugin)
		}
	}

	return plugins, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
