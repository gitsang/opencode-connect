package chat

import "context"

type ChatApp interface {
	Name() string
	HandleMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error)
}

type MessageRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type MessageResponse struct {
	Reply             string `json:"reply"`
	SessionID         string `json:"session_id"`
	OpencodeSessionID string `json:"opencode_session_id,omitempty"`
	Model             string `json:"model,omitempty"`
	Provider          string `json:"provider,omitempty"`
	Command           string `json:"command,omitempty"`
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{StatusCode: statusCode, Message: message}
}
