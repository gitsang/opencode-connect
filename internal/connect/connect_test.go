package connect

import (
	"context"
	"fmt"
	"testing"

	"github.com/gitsang/opencode-connect/internal/opencode"
	"github.com/gitsang/opencode-connect/internal/session"
)

func TestHandleCreatesAndStoresSessionWhenMissing(t *testing.T) {
	t.Parallel()

	store := session.NewMemoryStore()
	client := &fakeSessionClient{
		createSessionID: "session-created",
		promptResult: &opencode.PromptResult{
			Reply:             "hello",
			OpencodeSessionID: "session-created",
		},
	}

	connector := New(client, store)
	resp, err := connector.Handle(context.Background(), &Message{
		SessionID: "chat-1",
		Message:   "hello world",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if resp.OpencodeSessionID != "session-created" {
		t.Fatalf("Handle() session = %q, want %q", resp.OpencodeSessionID, "session-created")
	}
	if client.createCalls != 1 {
		t.Fatalf("CreateSession() calls = %d, want 1", client.createCalls)
	}
	if client.promptSessionID != "session-created" {
		t.Fatalf("Prompt() session = %q, want %q", client.promptSessionID, "session-created")
	}
	if stored, ok := store.Get("chat-1"); !ok || stored != "session-created" {
		t.Fatalf("store.Get() = (%q, %t), want (%q, true)", stored, ok, "session-created")
	}
}

func TestHandleUsesDirectiveSessionAndStoresBinding(t *testing.T) {
	t.Parallel()

	store := session.NewMemoryStore()
	client := &fakeSessionClient{
		promptResult: &opencode.PromptResult{
			Reply:             "hello",
			OpencodeSessionID: "existing-session",
		},
	}

	connector := New(client, store)
	resp, err := connector.Handle(context.Background(), &Message{
		SessionID: "chat-2",
		Message:   "@session:existing-session\n\nhello world",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if resp.OpencodeSessionID != "existing-session" {
		t.Fatalf("Handle() session = %q, want %q", resp.OpencodeSessionID, "existing-session")
	}
	if client.getSessionID != "existing-session" {
		t.Fatalf("GetSession() session = %q, want %q", client.getSessionID, "existing-session")
	}
	if client.promptSessionID != "existing-session" {
		t.Fatalf("Prompt() session = %q, want %q", client.promptSessionID, "existing-session")
	}
	if stored, ok := store.Get("chat-2"); !ok || stored != "existing-session" {
		t.Fatalf("store.Get() = (%q, %t), want (%q, true)", stored, ok, "existing-session")
	}
}

type fakeSessionClient struct {
	createSessionID string
	promptResult    *opencode.PromptResult
	listSessions    []opencode.Session
	getErr          error
	createErr       error
	promptErr       error
	getSessionID    string
	promptSessionID string
	createCalls     int
}

func (f *fakeSessionClient) ListSessions(context.Context) ([]opencode.Session, error) {
	return f.listSessions, nil
}

func (f *fakeSessionClient) GetSession(_ context.Context, sessionID string) (*opencode.Session, error) {
	f.getSessionID = sessionID
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &opencode.Session{ID: sessionID}, nil
}

func (f *fakeSessionClient) CreateSession(_ context.Context, sessionID string) (*opencode.Session, error) {
	f.createCalls++
	if f.createErr != nil {
		return nil, f.createErr
	}
	createdID := f.createSessionID
	if createdID == "" {
		createdID = sessionID
	}
	return &opencode.Session{ID: createdID}, nil
}

func (f *fakeSessionClient) Prompt(_ context.Context, sessionID string, _ string) (*opencode.PromptResult, error) {
	f.promptSessionID = sessionID
	if f.promptErr != nil {
		return nil, f.promptErr
	}
	if f.promptResult == nil {
		return nil, fmt.Errorf("prompt result is required")
	}
	return f.promptResult, nil
}
