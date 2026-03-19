package connect

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gitsang/opencode-connect/internal/config"
	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/session"
)

type OpencodeConnect struct {
	opencodeClient *opencode.Client
	sessionStore   session.Store
	cfg            *config.Config
	resolveMu      sync.Mutex
}

func New(opencodeClient *opencode.Client, sessionStore session.Store, cfg *config.Config) *OpencodeConnect {
	return &OpencodeConnect{
		opencodeClient: opencodeClient,
		sessionStore:   sessionStore,
		cfg:            cfg,
	}
}

func (c *OpencodeConnect) Handle(ctx context.Context, req *Message) (*Message, error) {
	if req == nil {
		return nil, NewError(http.StatusBadRequest, "request is required")
	}

	if strings.TrimSpace(req.SessionID) == "" {
		return nil, NewError(http.StatusBadRequest, "session_id is required")
	}

	parsed, err := ParseMessage(req.Message)
	if err != nil {
		return nil, NewError(http.StatusBadRequest, err.Error())
	}

	if parsed.SlashCommand == slashSessions {
		listing, err := c.listSessions(ctx)
		if err != nil {
			return nil, NewError(http.StatusBadGateway, err.Error())
		}

		return &Message{
			SessionID: req.SessionID,
			Reply:     listing,
			Command:   slashSessions,
		}, nil
	}

	targetOpencodeSessionID, err := c.resolveOpencodeSessionID(ctx, req.SessionID, parsed.SessionCommand)
	if err != nil {
		return nil, NewError(http.StatusBadGateway, err.Error())
	}

	result, err := c.opencodeClient.Prompt(ctx, targetOpencodeSessionID, parsed.Body, parsed.ModelCommand)
	if err != nil {
		return nil, NewError(http.StatusBadGateway, err.Error())
	}

	if parsed.SessionCommand != "" {
		c.sessionStore.Set(req.SessionID, parsed.SessionCommand)
	}

	return &Message{
		SessionID:         req.SessionID,
		Reply:             result.Reply,
		OpencodeSessionID: result.OpencodeSessionID,
		Provider:          result.ProviderID,
		Model:             result.ModelID,
	}, nil
}

func (c *OpencodeConnect) resolveOpencodeSessionID(ctx context.Context, chatSessionID string, sessionOverride string) (string, error) {
	if strings.TrimSpace(sessionOverride) != "" {
		target := strings.TrimSpace(sessionOverride)
		if _, err := c.opencodeClient.GetSession(ctx, target); err != nil {
			return "", fmt.Errorf("session not found: %s", target)
		}
		return target, nil
	}

	c.resolveMu.Lock()
	defer c.resolveMu.Unlock()

	if opencodeSessionID, ok := c.sessionStore.Get(chatSessionID); ok {
		return opencodeSessionID, nil
	}

	created, err := c.opencodeClient.CreateSession(ctx, c.opencodeClient.NewSessionTitle(chatSessionID))
	if err != nil {
		return "", err
	}

	c.sessionStore.Set(chatSessionID, created.ID)
	return created.ID, nil
}

func (c *OpencodeConnect) listSessions(ctx context.Context) (string, error) {
	sessions, err := c.opencodeClient.ListSessions(ctx)
	if err != nil {
		return "", err
	}

	if len(sessions) == 0 {
		return "- " + c.cfg.Opencode.Directory, nil
	}

	byDirectory := map[string][]string{}
	for _, currentSession := range sessions {
		directory := currentSession.Directory
		if strings.TrimSpace(directory) == "" {
			directory = c.cfg.Opencode.Directory
		}

		title := strings.TrimSpace(currentSession.Title)
		if title == "" {
			title = "Untitled"
		}

		line := fmt.Sprintf("  - %s (%s)", title, currentSession.ID)
		byDirectory[directory] = append(byDirectory[directory], line)
	}

	directories := make([]string, 0, len(byDirectory))
	for directory := range byDirectory {
		directories = append(directories, directory)
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
