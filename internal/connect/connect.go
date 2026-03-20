package connect

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/session"
)

type sessionClient interface {
	ListSessions(ctx context.Context) ([]opencode.Session, error)
	GetSession(ctx context.Context, sessionID string) (*opencode.Session, error)
	CreateSession(ctx context.Context, sessionID string) (*opencode.Session, error)
	Prompt(ctx context.Context, sessionID string, message string) (*opencode.PromptResult, error)
}

type OpencodeConnect struct {
	opencodeClient sessionClient
	sessionStore   session.Store
	resolveMu      sync.Mutex
}

func New(opencodeClient sessionClient, sessionStore session.Store) *OpencodeConnect {
	return &OpencodeConnect{
		opencodeClient: opencodeClient,
		sessionStore:   sessionStore,
	}
}

func (c *OpencodeConnect) Handle(ctx context.Context, req *Message) (*Message, error) {
	if req == nil {
		return nil, NewError(http.StatusBadRequest, "request is required")
	}

	if strings.TrimSpace(req.SessionID) == "" {
		return nil, NewError(http.StatusBadRequest, "session_id is required")
	}
	if c.opencodeClient == nil {
		return nil, NewError(http.StatusInternalServerError, "opencode client is required")
	}
	if c.sessionStore == nil {
		return nil, NewError(http.StatusInternalServerError, "session store is required")
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

	targetOpencodeSessionID := strings.TrimSpace(req.OpencodeSessionID)
	if parsed.SessionCommand != "" {
		targetOpencodeSessionID = strings.TrimSpace(parsed.SessionCommand)
		if _, err := c.opencodeClient.GetSession(ctx, targetOpencodeSessionID); err != nil {
			return nil, NewError(http.StatusBadGateway, fmt.Sprintf("session not found: %s", targetOpencodeSessionID))
		}
		c.sessionStore.Set(req.SessionID, targetOpencodeSessionID)
	}

	if targetOpencodeSessionID == "" {
		resolvedSessionID, err := c.resolveOpencodeSessionID(ctx, req.SessionID)
		if err != nil {
			return nil, NewError(http.StatusBadGateway, err.Error())
		}
		targetOpencodeSessionID = resolvedSessionID
	}

	result, err := c.opencodeClient.Prompt(ctx, targetOpencodeSessionID, parsed.Body)
	if err != nil {
		return nil, NewError(http.StatusBadGateway, err.Error())
	}
	if strings.TrimSpace(result.OpencodeSessionID) != "" {
		c.sessionStore.Set(req.SessionID, result.OpencodeSessionID)
	}

	return &Message{
		SessionID:         req.SessionID,
		Reply:             result.Reply,
		OpencodeSessionID: result.OpencodeSessionID,
	}, nil
}

func (c *OpencodeConnect) resolveOpencodeSessionID(ctx context.Context, chatSessionID string) (string, error) {
	if opencodeSessionID, ok := c.sessionStore.Get(chatSessionID); ok {
		return opencodeSessionID, nil
	}

	c.resolveMu.Lock()
	defer c.resolveMu.Unlock()

	if opencodeSessionID, ok := c.sessionStore.Get(chatSessionID); ok {
		return opencodeSessionID, nil
	}

	created, err := c.opencodeClient.CreateSession(ctx, chatSessionID)
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
		return "- (no sessions)", nil
	}

	byDirectory := map[string][]string{}
	for _, currentSession := range sessions {
		directory := strings.TrimSpace(currentSession.Directory)
		if directory == "" {
			directory = "."
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
