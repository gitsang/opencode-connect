package opencode

import (
	"context"
	"fmt"
	"strings"
	"time"

	opsdk "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

type Option func(*clientConfig)

type clientConfig struct {
	baseURL         string
	password        string
	passwordHeader  string
	passwordScheme  string
	directory       string
	promptTimeout   time.Duration
	defaultProvider string
	defaultModel    string
	modelAliases    map[string]string
	sessionTitleTpl string
	extraHeaders    map[string]any
}

func WithBaseURL(baseURL string) Option {
	return func(cfg *clientConfig) {
		cfg.baseURL = strings.TrimSpace(baseURL)
	}
}

func WithPassword(password string) Option {
	return func(cfg *clientConfig) {
		cfg.password = password
	}
}

func WithPasswordHeader(passwordHeader string) Option {
	return func(cfg *clientConfig) {
		cfg.passwordHeader = strings.TrimSpace(passwordHeader)
	}
}

func WithPasswordScheme(passwordScheme string) Option {
	return func(cfg *clientConfig) {
		cfg.passwordScheme = strings.TrimSpace(passwordScheme)
	}
}

func WithDirectory(directory string) Option {
	return func(cfg *clientConfig) {
		cfg.directory = strings.TrimSpace(directory)
	}
}

func WithPromptTimeout(timeout time.Duration) Option {
	return func(cfg *clientConfig) {
		cfg.promptTimeout = timeout
	}
}

func WithDefaultModel(providerID string, modelID string) Option {
	return func(cfg *clientConfig) {
		cfg.defaultProvider = strings.TrimSpace(providerID)
		cfg.defaultModel = strings.TrimSpace(modelID)
	}
}

func WithModelAliases(aliases map[string]string) Option {
	return func(cfg *clientConfig) {
		if aliases == nil {
			cfg.modelAliases = nil
			return
		}

		copied := make(map[string]string, len(aliases))
		for alias, modelRef := range aliases {
			copied[alias] = modelRef
		}
		cfg.modelAliases = copied
	}
}

func WithSessionTitleTemplate(template string) Option {
	return func(cfg *clientConfig) {
		cfg.sessionTitleTpl = template
	}
}

func WithExtraHeaders(headers map[string]any) Option {
	return func(cfg *clientConfig) {
		if headers == nil {
			cfg.extraHeaders = nil
			return
		}

		copied := make(map[string]any, len(headers))
		for key, value := range headers {
			copied[key] = value
		}
		cfg.extraHeaders = copied
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
	client          *opsdk.Client
	directory       string
	promptTimeout   time.Duration
	modelAliases    map[string]string
	sessionTitleTpl string
	defaultModel    ModelRef
	providerLoaded  bool
}

func NewClient(opts ...Option) (*Client, error) {
	cfg := clientConfig{
		passwordHeader: "Authorization",
		passwordScheme: "Bearer",
		directory:      ".",
		promptTimeout:  5 * time.Minute,
	}

	for _, applyOption := range opts {
		if applyOption == nil {
			continue
		}
		applyOption(&cfg)
	}

	options := []option.RequestOption{
		option.WithBaseURL(cfg.baseURL),
	}

	if cfg.password != "" {
		value := cfg.password
		if cfg.passwordScheme != "" {
			value = fmt.Sprintf("%s %s", cfg.passwordScheme, cfg.password)
		}
		options = append(options, option.WithHeader(cfg.passwordHeader, value))
	}

	for key, value := range cfg.extraHeaders {
		valueString := fmt.Sprint(value)
		if valueString == "" {
			continue
		}
		options = append(options, option.WithHeader(key, valueString))
	}

	client := opsdk.NewClient(options...)

	return &Client{
		client:          client,
		directory:       cfg.directory,
		promptTimeout:   cfg.promptTimeout,
		modelAliases:    cfg.modelAliases,
		sessionTitleTpl: cfg.sessionTitleTpl,
		defaultModel: ModelRef{
			ProviderID: cfg.defaultProvider,
			ModelID:    cfg.defaultModel,
		},
	}, nil
}

func (c *Client) CreateSession(ctx context.Context, title string) (*opsdk.Session, error) {
	if title == "" {
		title = "chat-session"
	}

	return c.client.Session.New(ctx, opsdk.SessionNewParams{
		Title:     opsdk.F(title),
		Directory: opsdk.F(c.directory),
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

	parts := []opsdk.SessionPromptParamsPartUnion{
		opsdk.TextPartInputParam{
			Type: opsdk.F(opsdk.TextPartInputTypeText),
			Text: opsdk.F(message),
		},
	}

	params := opsdk.SessionPromptParams{
		Parts:     opsdk.F(parts),
		Directory: opsdk.F(c.directory),
	}

	if modelRef.ProviderID != "" && modelRef.ModelID != "" {
		params.Model = opsdk.F(opsdk.SessionPromptParamsModel{
			ProviderID: opsdk.F(modelRef.ProviderID),
			ModelID:    opsdk.F(modelRef.ModelID),
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

func (c *Client) GetSession(ctx context.Context, opencodeSessionID string) (*opsdk.Session, error) {
	if strings.TrimSpace(opencodeSessionID) == "" {
		return nil, fmt.Errorf("opencode session id is required")
	}

	return c.client.Session.Get(ctx, opencodeSessionID, opsdk.SessionGetParams{
		Directory: opsdk.F(c.directory),
	})
}

func (c *Client) ListSessions(ctx context.Context) ([]opsdk.Session, error) {
	resp, err := c.client.Session.List(ctx, opsdk.SessionListParams{
		Directory: opsdk.F(c.directory),
	})
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return []opsdk.Session{}, nil
	}

	return *resp, nil
}

func (c *Client) resolveModel(ctx context.Context, token string) (ModelRef, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		if c.defaultModel.ProviderID != "" && c.defaultModel.ModelID != "" {
			return c.defaultModel, nil
		}

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

	resp, err := c.client.App.Providers(ctx, opsdk.AppProvidersParams{})
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

func extractReply(parts []opsdk.Part) string {
	builder := strings.Builder{}

	for _, part := range parts {
		if part.Type != opsdk.PartTypeText {
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

func (c *Client) NewSessionTitle(chatSessionID string) string {
	title := c.sessionTitleTpl
	if title == "" {
		title = "chat-session-{session_id}"
	}

	title = strings.ReplaceAll(title, "{session_id}", chatSessionID)
	if strings.TrimSpace(title) == "" {
		return fmt.Sprintf("chat-session-%d", time.Now().Unix())
	}

	return title
}
