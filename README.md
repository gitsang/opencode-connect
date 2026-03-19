# opencode-connect
An opencode plugin for connecting opencode to chat application

## Phase 1: Chat API

This repository now includes a plugin-oriented `opencode-connect` runtime.

### Features

- Configurable opencode server `base_url` and password header
- Plugin-based integration entry (`plugins.chatapi`, `plugins.ume`, `plugins.mattermost`)
- Each plugin manages its own runtime lifecycle (for example, HTTP server or websocket client)
- ChatAPI plugin provides a `POST /chat` synchronous endpoint
- In-memory mapping from chat `session_id` to opencode session
- Message head commands:
  - `@session:{opencode-session-id}`
  - `@model:{provider/model}` or alias from config
  - `/sessions`

### Request

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "message": "hello world",
    "session_id": "1"
  }' \
  http://127.0.0.1:8192/chat
```

### Build & Run

```bash
cp configs/config.example.yaml configs/config.yaml
go run ./cmd/opencode-connect -c configs/config.yaml
```

### Test script

```bash
chmod +x scripts/chat-curl.sh
./scripts/chat-curl.sh "hello world" 1
```

### Config via env

Environment variables are supported by `configer` with prefix `OPENCODE_CONNECT_`, for example:

- `OPENCODE_CONNECT_OPENCODE_BASE_URL`
- `OPENCODE_CONNECT_OPENCODE_PASSWORD`
- `OPENCODE_CONNECT_PLUGINS_CHATAPI_LISTEN`

### Plugin config example

```yaml
plugins:
  chatapi:
    enabled: true
    listen: ":8192"
  ume:
    enabled: false
  mattermost:
    enabled: false
```
