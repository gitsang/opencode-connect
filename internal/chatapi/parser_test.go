package chatapi

import "testing"

func TestParseMessagePlain(t *testing.T) {
	parsed, err := ParseMessage("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Body != "hello world" {
		t.Fatalf("unexpected body: %q", parsed.Body)
	}
}

func TestParseMessageDirectiveHead(t *testing.T) {
	input := "@model:openai/gpt-5.4\n@session:abc123\n\nHi!"

	parsed, err := ParseMessage(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.ModelCommand != "openai/gpt-5.4" {
		t.Fatalf("unexpected model: %q", parsed.ModelCommand)
	}

	if parsed.SessionCommand != "abc123" {
		t.Fatalf("unexpected session: %q", parsed.SessionCommand)
	}

	if parsed.Body != "Hi!" {
		t.Fatalf("unexpected body: %q", parsed.Body)
	}
}

func TestParseMessageSlashCommand(t *testing.T) {
	parsed, err := ParseMessage("/sessions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.SlashCommand != "/sessions" {
		t.Fatalf("unexpected command: %q", parsed.SlashCommand)
	}
}

func TestParseMessageDirectiveNotAtHead(t *testing.T) {
	parsed, err := ParseMessage("Hi\n@model:openai/gpt-5.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Body != "Hi\n@model:openai/gpt-5.4" {
		t.Fatalf("unexpected body: %q", parsed.Body)
	}

	if parsed.ModelCommand != "" {
		t.Fatalf("model should not be parsed: %q", parsed.ModelCommand)
	}
}

func TestParseMessageDuplicateSessionDirective(t *testing.T) {
	_, err := ParseMessage("@session:abc\n@session:def\n\nhello")
	if err == nil {
		t.Fatalf("expected duplicate @session error")
	}
}

func TestParseMessageDuplicateModelDirective(t *testing.T) {
	_, err := ParseMessage("@model:openai/gpt-5.4\n@model:anthropic/claude\n\nhello")
	if err == nil {
		t.Fatalf("expected duplicate @model error")
	}
}
