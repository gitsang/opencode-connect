package opencode

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	ocsdk "github.com/sst/opencode-sdk-go"
)

type Option func(*clientConfig)

type clientConfig struct {
	promptTimeout time.Duration
	modelAliases  map[string]string
}

func WithPromptTimeout(timeout time.Duration) Option {
	return func(cfg *clientConfig) {
		cfg.promptTimeout = timeout
	}
}

func WithModelAliases(aliases map[string]string) Option {
	return func(cfg *clientConfig) {
		if aliases == nil {
			cfg.modelAliases = nil
			return
		}

		copied := make(map[string]string, len(aliases))
		maps.Copy(copied, aliases)
		cfg.modelAliases = copied
	}
}

type ModelRef struct {
	ProviderID string
	ModelID    string
}

type PromptResult struct {
	Reply             string
	OpencodeSessionID string
	ProviderID        string
	ModelID           string
}

type Client struct {
	client         *ocsdk.Client
	promptTimeout  time.Duration
	modelAliases   map[string]string
	defaultModel   ModelRef
	providerLoaded bool
}

func NewClient(sdkClient *ocsdk.Client, opts ...Option) *Client {
	cfg := clientConfig{
		promptTimeout: 5 * time.Minute,
	}

	for _, applyOption := range opts {
		if applyOption == nil {
			continue
		}
		applyOption(&cfg)
	}

	return &Client{
		client:        sdkClient,
		promptTimeout: cfg.promptTimeout,
		modelAliases:  cfg.modelAliases,
	}
}

func (c *Client) CreateSession(ctx context.Context, chatSessionID string) (*ocsdk.Session, error) {
	title := fmt.Sprintf("chat-session-%s", chatSessionID)
	return c.client.Session.New(ctx, ocsdk.SessionNewParams{
		Title: ocsdk.F(title),
	})
}

func (c *Client) Prompt(ctx context.Context, opencodeSessionID string, message string, modelToken string) (*PromptResult, error) {
	if opencodeSessionID == "" {
		return nil, fmt.Errorf("opencode session id is required")
	}
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("message is required")
	}

	modelRef, err := c.resolveModel(ctx, modelToken)
	if err != nil {
		return nil, err
	}

	promptCtx, cancel := context.WithTimeout(ctx, c.promptTimeout)
	defer cancel()

	parts := []ocsdk.SessionPromptParamsPartUnion{
		ocsdk.TextPartInputParam{
			Type: ocsdk.F(ocsdk.TextPartInputTypeText),
			Text: ocsdk.F(message),
		},
	}

	params := ocsdk.SessionPromptParams{
		Parts: ocsdk.F(parts),
	}

	if modelRef.ProviderID != "" && modelRef.ModelID != "" {
		params.Model = ocsdk.F(ocsdk.SessionPromptParamsModel{
			ProviderID: ocsdk.F(modelRef.ProviderID),
			ModelID:    ocsdk.F(modelRef.ModelID),
		})
	}

	resp, err := c.client.Session.Prompt(promptCtx, opencodeSessionID, params)
	if err != nil {
		return nil, err
	}

	return &PromptResult{
		Reply:             extractReply(resp.Parts),
		OpencodeSessionID: opencodeSessionID,
		ProviderID:        modelRef.ProviderID,
		ModelID:           modelRef.ModelID,
	}, nil
}

func (c *Client) GetSession(ctx context.Context, opencodeSessionID string) (*ocsdk.Session, error) {
	if strings.TrimSpace(opencodeSessionID) == "" {
		return nil, fmt.Errorf("opencode session id is required")
	}

	return c.client.Session.Get(ctx, opencodeSessionID, ocsdk.SessionGetParams{})
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

func (c *Client) resolveModel(ctx context.Context, token string) (ModelRef, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		if err := c.loadDefaultModel(ctx); err != nil {
			return ModelRef{}, err
		}
		return c.defaultModel, nil
	}

	if alias, ok := c.modelAliases[token]; ok {
		parsed, err := parseModelRef(alias)
		if err != nil {
			return ModelRef{}, fmt.Errorf("invalid model alias %q: %w", token, err)
		}
		return parsed, nil
	}

	return parseModelRef(token)
}

func (c *Client) loadDefaultModel(ctx context.Context) error {
	if c.providerLoaded {
		if c.defaultModel.ProviderID == "" || c.defaultModel.ModelID == "" {
			return fmt.Errorf("default model is not configured")
		}
		return nil
	}

	c.providerLoaded = true

	resp, err := c.client.App.Providers(ctx, ocsdk.AppProvidersParams{})
	if err != nil {
		return err
	}

	if resp == nil {
		return fmt.Errorf("providers response is nil")
	}

	c.defaultModel.ProviderID = strings.TrimSpace(resp.Default["provider"])
	c.defaultModel.ModelID = strings.TrimSpace(resp.Default["model"])

	if c.defaultModel.ProviderID == "" || c.defaultModel.ModelID == "" {
		return fmt.Errorf("default provider/model unavailable from opencode")
	}

	return nil
}

func parseModelRef(value string) (ModelRef, error) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 {
		return ModelRef{}, fmt.Errorf("model must be provider/model")
	}

	provider := strings.TrimSpace(parts[0])
	model := strings.TrimSpace(parts[1])

	if provider == "" || model == "" {
		return ModelRef{}, fmt.Errorf("model must be provider/model")
	}

	return ModelRef{ProviderID: provider, ModelID: model}, nil
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