package chatapi

import (
	"fmt"
	"strings"
)

const (
	commandSession  = "session"
	commandModel    = "model"
	slashSessions   = "/sessions"
	directivePrefix = "@"
)

type ParsedMessage struct {
	Body           string
	SlashCommand   string
	SessionCommand string
	ModelCommand   string
}

func ParseMessage(message string) (*ParsedMessage, error) {
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}

	lines := strings.Split(message, "\n")
	firstLine := lines[0]
	if strings.HasPrefix(firstLine, "/") {
		command := strings.TrimSpace(firstLine)
		if command == slashSessions {
			return &ParsedMessage{SlashCommand: slashSessions}, nil
		}

		return nil, fmt.Errorf("unsupported slash command: %s", command)
	}

	if !strings.HasPrefix(firstLine, directivePrefix) {
		return &ParsedMessage{Body: message}, nil
	}

	parsed := &ParsedMessage{}
	lineIndex := 0
	for lineIndex < len(lines) {
		line := strings.TrimSpace(lines[lineIndex])
		if line == "" {
			lineIndex++
			break
		}

		if !strings.HasPrefix(line, directivePrefix) {
			break
		}

		if err := applyDirective(parsed, line); err != nil {
			return nil, err
		}

		lineIndex++
	}

	if lineIndex < len(lines) {
		parsed.Body = strings.Join(lines[lineIndex:], "\n")
	}

	parsed.Body = strings.TrimSpace(parsed.Body)
	if parsed.Body == "" {
		return nil, fmt.Errorf("message body cannot be empty")
	}

	return parsed, nil
}

func applyDirective(parsed *ParsedMessage, line string) error {
	pair := strings.SplitN(strings.TrimPrefix(line, directivePrefix), ":", 2)
	if len(pair) != 2 {
		return fmt.Errorf("invalid directive format: %s", line)
	}

	key := strings.TrimSpace(strings.ToLower(pair[0]))
	value := strings.TrimSpace(pair[1])
	if value == "" {
		return fmt.Errorf("directive value cannot be empty: %s", key)
	}

	switch key {
	case commandSession:
		if parsed.SessionCommand != "" {
			return fmt.Errorf("duplicate @session directive")
		}
		parsed.SessionCommand = value
	case commandModel:
		if parsed.ModelCommand != "" {
			return fmt.Errorf("duplicate @model directive")
		}
		parsed.ModelCommand = value
	default:
		return fmt.Errorf("unsupported directive: %s", key)
	}

	return nil
}
