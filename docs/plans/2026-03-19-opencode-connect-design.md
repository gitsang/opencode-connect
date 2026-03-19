# Opencode Connect V1 Design

## Goal

Build a Go-based `opencode connect` server that exposes a unified chat application integration model, with Chat API support in phase 1.

## Scope

- Use libraries:
  - `github.com/sst/opencode-sdk-go` (module path of anomalyco repository)
  - `github.com/gitsang/logi`
  - `github.com/gitsang/configer`
- Provide Chat API HTTP server without inbound auth in v1
- Provide synchronous chat endpoint
- Map Chat API `session_id` to opencode session
- Support message-head commands:
  - `@session:{opencode-session-id}`
  - `@model:{model}`
  - `/sessions`
- Provide shell curl script for testing

## Assumptions

- The SDK repository URL is `anomalyco`, but Go import path is `github.com/sst/opencode-sdk-go`
- `@model` supports:
  - explicit `provider/model`
  - configured aliases (e.g. `gpt-5.4 -> provider+model`)
  - fallback to server default provider/model

## Approaches

### Approach A (Recommended): Unified interface + in-memory mapping

- Create a generic `ChatApp` interface and a Chat API implementation
- Keep `chat_session_id -> opencode_session_id` in memory (`sync.RWMutex` map)
- Parse commands only from message head
- `/sessions` reads sessions from opencode and formats text output

Trade-offs:
- Pros: simple, fast to ship, clean extension point for mattermost/ume
- Cons: mapping is not persistent across restarts

### Approach B: Direct HTTP-only flow without abstraction

- Hardcode chat logic in HTTP handler

Trade-offs:
- Pros: minimum lines today
- Cons: hard to extend to other chat apps later

### Approach C: Persistent mapping + repository layer in v1

- Add database-backed mapping for chat sessions

Trade-offs:
- Pros: durable mapping
- Cons: extra complexity and infra before v1 needs it

## Architecture

- `cmd/opencode-connect/main.go`: process bootstrap and graceful shutdown
- `internal/config`: config schema and loading via `configer`
- `internal/app`: server bootstrap and logger bootstrap
- `internal/chat`: unified `ChatApp` interface and request/response models
- `internal/session`: in-memory chat-to-opencode session mapping store
- `internal/opencode`: SDK client wrapper, prompt/session/model operations
- `internal/chatapi`: Chat API implementation and command parser

## Data Flow

1. HTTP `POST /chat` receives `{message, session_id}`
2. Chat API parser checks first lines for slash/directive commands
3. If `/sessions`, list sessions and return formatted text
4. Otherwise resolve target opencode session:
   - `@session` override if present
   - else map from chat session id
   - else create new opencode session and bind
5. Resolve model:
   - `@model` explicit or alias
   - else default configured model
6. Send sync `Session.Prompt`
7. Return assistant text and used session/model metadata

## Error Handling

- Invalid JSON / empty message / empty `session_id` => `400`
- Invalid command format => `400`
- Unknown model alias or invalid model token => `400`
- SDK/network/provider errors => `502`
- Internal failures => `500`

## Testing Strategy

- Unit tests for message-head parser:
  - plain message
  - directives at head
  - slash command
  - commands not at head should be treated as content
- Build verification with `go test ./...`

## Future Extensions

- Add persistent session mapping backend
- Add `mattermost` and `ume` adapters implementing `ChatApp`
- Add inbound auth middleware for Chat API
