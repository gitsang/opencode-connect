package opencode

import (
	"context"
	"fmt"
	"strings"
	"time"

	ocsdk "github.com/sst/opencode-sdk-go"
)

type PromptResult struct {
	Reply             string
	OpencodeSessionID string
}

type Client struct {
	client  *ocsdk.Client
	timeout time.Duration
}

func NewClient(sdkClient *ocsdk.Client) *Client {
	return &Client{
		client:  sdkClient,
		timeout: 10 * time.Minute,
	}
}

func (c *Client) ListSessions(ctx context.Context) ([]ocsdk.Session, error) {
	resp, err := c.client.Session.List(ctx, ocsdk.SessionListParams{})
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return []ocsdk.Session{}, nil
	}

	return *resp, nil
}

func (c *Client) GetSession(ctx context.Context, sessionId string) (*ocsdk.Session, error) {
	if strings.TrimSpace(sessionId) == "" {
		return nil, fmt.Errorf("opencode session id is required")
	}

	return c.client.Session.Get(ctx, sessionId, ocsdk.SessionGetParams{})
}

func (c *Client) CreateSession(ctx context.Context, sessionId string) (*ocsdk.Session, error) {
	title := fmt.Sprintf("chat-session-%s", sessionId)
	return c.client.Session.New(ctx, ocsdk.SessionNewParams{
		Title: ocsdk.F(title),
	})
}

func (c *Client) Prompt(ctx context.Context, sessionId string, message string) (*PromptResult, error) {
	if sessionId == "" {
		return nil, fmt.Errorf("opencode session id is required")
	}
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("message is required")
	}

	promptCtx, promptCancel := context.WithTimeout(ctx, c.timeout)
	defer promptCancel()

	parts := []ocsdk.SessionPromptParamsPartUnion{
		ocsdk.TextPartInputParam{
			Type: ocsdk.F(ocsdk.TextPartInputTypeText),
			Text: ocsdk.F(message),
		},
	}

	params := ocsdk.SessionPromptParams{
		Parts: ocsdk.F(parts),
		Model: ocsdk.F(ocsdk.SessionPromptParamsModel{
			ProviderID: ocsdk.F(""),
			ModelID:    ocsdk.F(""),
		}),
	}

	resp, err := c.client.Session.Prompt(promptCtx, sessionId, params)
	if err != nil {
		return nil, err
	}

	return &PromptResult{
		Reply:             extractReply(resp.Parts),
		OpencodeSessionID: sessionId,
	}, nil
}

func extractReply(parts []ocsdk.Part) string {
	builder := strings.Builder{}

	for _, part := range parts {
		if part.Type != ocsdk.PartTypeText {
			continue
		}

		if text := strings.TrimSpace(part.Text); text != "" {
			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(text)
		}
	}

	return strings.TrimSpace(builder.String())
}

