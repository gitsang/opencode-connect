package connect

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gitsang/opencode-connect/internal/opencode"
)

type OpencodeConnect struct {
	opencodeClient *opencode.Client
}

func New(opencodeClient *opencode.Client) *OpencodeConnect {
	return &OpencodeConnect{
		opencodeClient: opencodeClient,
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

	targetOpencodeSessionID := strings.TrimSpace(req.OpencodeSessionID)
	if parsed.SessionCommand != "" {
		targetOpencodeSessionID = strings.TrimSpace(parsed.SessionCommand)
		if _, err := c.opencodeClient.GetSession(ctx, targetOpencodeSessionID); err != nil {
			return nil, NewError(http.StatusBadGateway, fmt.Sprintf("session not found: %s", targetOpencodeSessionID))
		}
	}

	if targetOpencodeSessionID == "" {
		return nil, NewError(http.StatusBadRequest, "opencode_session_id is required")
	}

	result, err := c.opencodeClient.Prompt(ctx, targetOpencodeSessionID, parsed.Body, parsed.ModelCommand)
	if err != nil {
		return nil, NewError(http.StatusBadGateway, err.Error())
	}

	return &Message{
		SessionID:         req.SessionID,
		Reply:             result.Reply,
		OpencodeSessionID: result.OpencodeSessionID,
		Provider:          result.ProviderID,
		Model:             result.ModelID,
	}, nil
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
