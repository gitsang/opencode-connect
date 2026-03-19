package chatapi

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gitsang/opencode-connect/internal/chat"
	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/session"
)

type ChatAPI struct {
	opencodeClient *opencode.Client
	sessionStore   session.Store
	cfg            *config.Config
	resolveMu      sync.Mutex
}

func NewChatAPI(opencodeClient *opencode.Client, sessionStore session.Store, cfg *config.Config) *ChatAPI {
	return &ChatAPI{
		opencodeClient: opencodeClient,
		sessionStore:   sessionStore,
		cfg:            cfg,
	}
}

func (a *ChatAPI) Name() string {
	return "chat-api"
}

func (a *ChatAPI) HandleMessage(ctx context.Context, req chat.MessageRequest) (*chat.MessageResponse, error) {
	if strings.TrimSpace(req.SessionID) == "" {
		return nil, chat.NewHTTPError(http.StatusBadRequest, "session_id is required")
	}

	parsed, err := ParseMessage(req.Message)
	if err != nil {
		return nil, chat.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if parsed.SlashCommand == slashSessions {
		listing, err := a.listSessions(ctx)
		if err != nil {
			return nil, chat.NewHTTPError(http.StatusBadGateway, err.Error())
		}

		return &chat.MessageResponse{
			Reply:     listing,
			SessionID: req.SessionID,
			Command:   slashSessions,
		}, nil
	}

	targetOpencodeSessionID, err := a.resolveOpencodeSessionID(ctx, req.SessionID, parsed.SessionCommand)
	if err != nil {
		return nil, chat.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	result, err := a.opencodeClient.Prompt(ctx, targetOpencodeSessionID, parsed.Body, parsed.ModelCommand)
	if err != nil {
		return nil, chat.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	if parsed.SessionCommand != "" {
		a.sessionStore.Set(req.SessionID, parsed.SessionCommand)
	}

	return &chat.MessageResponse{
		Reply:             result.Reply,
		SessionID:         req.SessionID,
		OpencodeSessionID: result.OpencodeSessionID,
		Provider:          result.ProviderID,
		Model:             result.ModelID,
	}, nil
}

func (a *ChatAPI) resolveOpencodeSessionID(ctx context.Context, chatSessionID string, sessionOverride string) (string, error) {
	if strings.TrimSpace(sessionOverride) != "" {
		target := strings.TrimSpace(sessionOverride)
		if _, err := a.opencodeClient.GetSession(ctx, target); err != nil {
			return "", fmt.Errorf("session not found: %s", target)
		}
		return target, nil
	}

	a.resolveMu.Lock()
	defer a.resolveMu.Unlock()

	if opencodeSessionID, ok := a.sessionStore.Get(chatSessionID); ok {
		return opencodeSessionID, nil
	}

	created, err := a.opencodeClient.CreateSession(ctx, a.opencodeClient.NewSessionTitle(chatSessionID))
	if err != nil {
		return "", err
	}

	a.sessionStore.Set(chatSessionID, created.ID)
	return created.ID, nil
}

func (a *ChatAPI) listSessions(ctx context.Context) (string, error) {
	sessions, err := a.opencodeClient.ListSessions(ctx)
	if err != nil {
		return "", err
	}

	if len(sessions) == 0 {
		return "- " + a.cfg.Opencode.Directory, nil
	}

	byDirectory := map[string][]string{}
	for _, s := range sessions {
		directory := s.Directory
		if strings.TrimSpace(directory) == "" {
			directory = a.cfg.Opencode.Directory
		}

		title := strings.TrimSpace(s.Title)
		if title == "" {
			title = "Untitled"
		}

		line := fmt.Sprintf("  - %s (%s)", title, s.ID)
		byDirectory[directory] = append(byDirectory[directory], line)
	}

	directories := make([]string, 0, len(byDirectory))
	for dir := range byDirectory {
		directories = append(directories, dir)
	}
	sort.Strings(directories)

	builder := strings.Builder{}
	for index, directory := range directories {
		if index > 0 {
			builder.WriteString("\n")
		}

		builder.WriteString("- ")
		builder.WriteString(directory)
		builder.WriteString("\n")

		items := byDirectory[directory]
		sort.Strings(items)
		for _, item := range items {
			builder.WriteString(item)
			builder.WriteString("\n")
		}
	}

	return strings.TrimSpace(builder.String()), nil
}
