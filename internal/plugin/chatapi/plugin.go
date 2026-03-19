package chatapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gitsang/opencode-connect/internal/connect"
	"github.com/gitsang/opencode-connect/internal/opencode"
	coreplugin "github.com/gitsang/opencode-connect/internal/plugin"
	"github.com/gitsang/opencode-connect/internal/session"
)

type Plugin struct {
	logger         *slog.Logger
	cfg            coreplugin.ChatAPIConfig
	opencodeClient *opencode.Client
	sessionStore   session.Store
	resolveMu      sync.Mutex
}

func init() {
	coreplugin.Register(coreplugin.Registration{
		Key:     "chatapi",
		Enabled: func(deps coreplugin.Dependencies) bool { return deps.EnableChatAPI },
		Build: func(deps coreplugin.Dependencies) (coreplugin.Plugin, error) {
			if deps.OpencodeClient == nil {
				return nil, fmt.Errorf("chatapi dependencies.opencodeClient is required")
			}
			if deps.SessionStore == nil {
				return nil, fmt.Errorf("chatapi dependencies.sessionStore is required")
			}

			return New(deps.Logger, deps.OpencodeClient, deps.SessionStore, deps.ChatAPI), nil
		},
	})
}

func New(logger *slog.Logger, opencodeClient *opencode.Client, sessionStore session.Store, cfg coreplugin.ChatAPIConfig) *Plugin {
	return &Plugin{
		logger:         logger,
		cfg:            cfg,
		opencodeClient: opencodeClient,
		sessionStore:   sessionStore,
	}
}

func (p *Plugin) Name() string {
	return "chatapi"
}

func (p *Plugin) Serve(ctx context.Context, handle coreplugin.HandleFunc) error {
	if handle == nil {
		return fmt.Errorf("chatapi handle is required")
	}

	serverConfig := p.cfg
	server := &http.Server{
		Addr:    serverConfig.Listen,
		Handler: p.newHTTPHandler(handle),
	}

	errCh := make(chan error, 1)
	go func() {
		p.logger.Info("chatapi plugin started", "listen", serverConfig.Listen)
		errCh <- server.ListenAndServe()
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err := <-errCh
	if err == nil || err == http.ErrServerClosed {
		p.logger.Info("chatapi plugin stopped")
		return nil
	}

	return fmt.Errorf("listen chatapi http server: %w", err)
}

func (p *Plugin) Send(_ context.Context, _ *connect.Message) (*connect.Message, error) {
	return nil, fmt.Errorf("chatapi plugin does not support proactive send")
}

func (p *Plugin) newHTTPHandler(handle coreplugin.HandleFunc) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		var req connect.Message
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}

		targetOpencodeSessionID, err := p.resolveOpencodeSessionID(r.Context(), req.SessionID, req.Message)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		req.OpencodeSessionID = targetOpencodeSessionID

		resp, err := handle(r.Context(), &req)
		if err != nil {
			status := http.StatusInternalServerError
			var connectError *connect.Error
			if errors.As(err, &connectError) {
				status = connectError.StatusCode
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}

		if strings.TrimSpace(resp.OpencodeSessionID) != "" {
			p.sessionStore.Set(req.SessionID, resp.OpencodeSessionID)
		}

		writeJSON(w, http.StatusOK, resp)
	})

	return mux
}

func (p *Plugin) resolveOpencodeSessionID(ctx context.Context, chatSessionID string, message string) (string, error) {
	if strings.TrimSpace(chatSessionID) == "" {
		return "", fmt.Errorf("session_id is required")
	}

	parsed, err := connect.ParseMessage(message)
	if err != nil {
		return "", nil
	}

	if parsed.SlashCommand != "" {
		return "", nil
	}

	if strings.TrimSpace(parsed.SessionCommand) != "" {
		target := strings.TrimSpace(parsed.SessionCommand)
		if _, getErr := p.opencodeClient.GetSession(ctx, target); getErr != nil {
			return "", fmt.Errorf("session not found: %s", target)
		}
		p.sessionStore.Set(chatSessionID, target)
		return target, nil
	}

	p.resolveMu.Lock()
	defer p.resolveMu.Unlock()

	if opencodeSessionID, ok := p.sessionStore.Get(chatSessionID); ok {
		return opencodeSessionID, nil
	}

	created, err := p.opencodeClient.CreateSession(ctx, chatSessionID)
	if err != nil {
		return "", err
	}

	p.sessionStore.Set(chatSessionID, created.ID)
	return created.ID, nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
