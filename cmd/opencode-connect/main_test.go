package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/gitsang/opencode-connect/internal/plugin"
)

func TestBuildPluginsSupportsMultipleInstancesOfSameType(t *testing.T) {
	t.Parallel()

	plugins, err := buildPlugins(map[string]any{
		"webui-chat": map[string]any{
			"chatapi": map[string]any{
				"listen": ":8193",
			},
		},
		"openai-chat": map[string]any{
			"chatapi": map[string]any{
				"listen": ":8192",
			},
		},
	}, testInfras())
	if err != nil {
		t.Fatalf("buildPlugins() error = %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("buildPlugins() len = %d, want 2", len(plugins))
	}
	if got := plugins[0].Name(); got != "openai-chat" {
		t.Fatalf("plugins[0].Name() = %q, want %q", got, "openai-chat")
	}
	if got := plugins[1].Name(); got != "webui-chat" {
		t.Fatalf("plugins[1].Name() = %q, want %q", got, "webui-chat")
	}
}

func TestBuildPluginsRejectsMultiplePluginTypesPerInstance(t *testing.T) {
	t.Parallel()

	_, err := buildPlugins(map[string]any{
		"openai-chat": map[string]any{
			"chatapi": map[string]any{"listen": ":8192"},
			"other":   map[string]any{},
		},
	}, testInfras())
	if err == nil {
		t.Fatal("buildPlugins() error = nil, want error")
	}
}

func testInfras() plugin.Infrastructure {
	return plugin.Infrastructure{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}
