# Backend

Go backend for the realtime sports commentary project.

It exposes REST endpoints for matches and commentary, plus a WebSocket endpoint for live commentary delivery. Commentary updates are match-scoped on the socket layer, so clients only receive events for the matches they subscribe to.

## Features

- Echo-based HTTP API
- PostgreSQL persistence for matches and commentary
- WebSocket server on the same backend process
- Match-level WS pub/sub for commentary updates
- Request validation with typed handlers
- Rate limiting for HTTP and WebSocket traffic
- Secure headers, request body limits, connection limits, ping/pong keepalive
- Redis and Asynq integration for background jobs
- Health checks and OpenAPI docs
- Structured logging and New Relic observability hooks

## Tech Stack

- Go
- Echo
- pgx / PostgreSQL
- Gorilla WebSocket
- Redis
- Asynq
- zerolog
- New Relic

## Local Setup

### Prerequisites

- Go installed
- PostgreSQL running locally
- Redis running locally
- `task` installed if you want to use the Taskfile commands

### Environment

The backend loads config from environment variables prefixed with `BOILERPLATE_`.

There is already a local [`.env`](/Users/ayushamin/Developer/repos/go-websockets/app/backend/.env) file with sample values. Key runtime settings include:

- `BOILERPLATE_SERVER.PORT`
- `BOILERPLATE_SERVER.CORS_ALLOWED_ORIGINS`
- `BOILERPLATE_SERVER.BODY_LIMIT`
- `BOILERPLATE_SERVER.HTTP_RATE_LIMIT_RPS`
- `BOILERPLATE_SERVER.WS_MAX_CONNECTIONS_PER_IP`
- `BOILERPLATE_DATABASE.*`
- `BOILERPLATE_REDIS.ADDRESS`

### Run the Server

From `app/backend`:

```bash
go run ./cmd/go-boilerplate
```

Or with Task:

```bash
task run
```

### Run Migrations

From `app/backend`:

```bash
task migrations:up
```

If you need a new migration:

```bash
task migrations:new name=add_something
```

## Project Structure

```text
app/backend/
├── cmd/go-boilerplate/            # application entrypoint
├── internal/
│   ├── config/                    # env config and defaults
│   ├── database/                  # pgx pool and embedded migrations
│   ├── errs/                      # HTTP error types
│   ├── handler/                   # HTTP and WS handlers
│   ├── middleware/                # security, tracing, rate limiting
│   ├── model/                     # domain models
│   ├── repository/                # SQL access layer
│   ├── router/                    # route registration
│   ├── server/                    # server wiring
│   └── service/                   # business logic
├── internal/database/migrations/  # SQL migrations
├── static/                        # OpenAPI artifacts
├── templates/                     # email templates
└── Taskfile.yml
```

## HTTP Endpoints

### System

- `GET /status`
  Returns backend health and dependency checks.

- `GET /docs`
  Serves the API documentation UI.

- `GET /ws`
  Upgrades the request to a WebSocket connection.

### Matches

- `GET /matches/`
  Lists matches ordered by newest first.

  Example:
  ```bash
  curl --request GET \
    --url 'http://localhost:8080/matches/?limit=20'
  ```

- `POST /matches`
  Creates a match.

  Example:
  ```bash
  curl --request POST \
    --url 'http://localhost:8080/matches' \
    --header 'Content-Type: application/json' \
    --data '{
      "sport": "Football",
      "homeTeam": "Manchester City",
      "awayTeam": "Arsenal",
      "startTime": "2026-04-09T18:00:00Z",
      "endTime": "2026-04-09T20:00:00Z"
    }'
  ```

### Commentary

- `GET /matches/:id/commentary`
  Lists commentary for one match, ordered by `created_at DESC`.

  Example:
  ```bash
  curl --request GET \
    --url 'http://localhost:8080/matches/1/commentary?limit=20'
  ```

- `POST /matches/:id/commentary`
  Creates commentary for a specific match.

  Example:
  ```bash
  curl --request POST \
    --url 'http://localhost:8080/matches/1/commentary' \
    --header 'Content-Type: application/json' \
    --data '{
      "minute": 18,
      "sequence": 3,
      "period": "1H",
      "eventType": "goal",
      "actor": "Erling Haaland",
      "team": "Manchester City",
      "message": "Haaland finishes from close range after a low cross.",
      "metadata": {
        "assistBy": "Kevin De Bruyne",
        "xg": 0.62
      },
      "tags": ["goal", "city", "open-play"]
    }'
  ```

  Minimal valid body:
  ```bash
  curl --request POST \
    --url 'http://localhost:8080/matches/1/commentary' \
    --header 'Content-Type: application/json' \
    --data '{
      "minute": 0,
      "message": "Kick-off"
    }'
  ```

## WebSocket API

Connect:

```text
ws://localhost:8080/ws
```

### Server Welcome Message

Immediately after connecting, the server sends:

```json
{"type":"welcome","message":"welcome"}
```

### Subscribe to Commentary for a Match

Client message:

```json
{"type":"subscribe","matchId":1}
```

Server response:

```json
{"type":"subscribed","matchId":1,"message":"subscribed"}
```

### Unsubscribe from a Match

Client message:

```json
{"type":"unsubscribe","matchId":1}
```

Server response:

```json
{"type":"unsubscribed","matchId":1,"message":"unsubscribed"}
```

### Commentary Event Delivery

When someone creates commentary through `POST /matches/:id/commentary`, the backend stores it in PostgreSQL and then emits:

```json
{
  "type": "commentary.created",
  "data": {
    "id": 12,
    "matchId": 1,
    "minute": 18,
    "sequence": 3,
    "period": "1H",
    "eventType": "goal",
    "actor": "Erling Haaland",
    "team": "Manchester City",
    "message": "Haaland finishes from close range after a low cross.",
    "metadata": {
      "assistBy": "Kevin De Bruyne",
      "xg": 0.62
    },
    "tags": ["goal", "city", "open-play"],
    "createdAt": "2026-04-09T18:18:00Z"
  }
}
```

Only clients subscribed to that `matchId` receive the event.

### Match Broadcasts

New match creation is still broadcast globally to all connected clients as:

```json
{
  "type": "match.created",
  "data": {
    "...": "match payload"
  }
}
```

## Security and Operational Notes

The backend currently includes:

- Per-IP HTTP rate limiting
- WS upgrade rate limiting
- WS inbound message rate limiting
- Request body limits
- Origin checks for WebSocket upgrades
- Max active WS connections per IP
- WS read/write deadlines and ping/pong keepalive
- Structured request logging with request IDs

If you change the frontend origin or deploy this, make sure `BOILERPLATE_SERVER.CORS_ALLOWED_ORIGINS` is set correctly.

## Development Notes

- Local env skips automatic DB migration on startup.
- Non-local env runs migrations during startup.
- SQL migrations live under [internal/database/migrations](/Users/ayushamin/Developer/repos/go-websockets/app/backend/internal/database/migrations).
- OpenAPI assets live under [static](/Users/ayushamin/Developer/repos/go-websockets/app/backend/static).

## Useful Commands

From `app/backend`:

```bash
go test ./...
go run ./cmd/go-boilerplate
task run
task migrations:up
task tidy
```
