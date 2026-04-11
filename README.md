# Go WebSockets

Realtime sports commentary backend built with Go, Echo, PostgreSQL, Redis, and Gorilla WebSocket.

This project exposes REST endpoints for matches and match commentary, then pushes live commentary updates over WebSockets. Clients subscribe to a specific `matchId`, and only those subscribers receive commentary events for that match.

## What It Does

- Create and list matches
- Create and list commentary for a match
- Stream live commentary updates over WebSockets
- Support match-level WS subscriptions with `subscribe` and `unsubscribe`
- Apply basic HTTP and WS security protections
- Include request logging, tracing hooks, health checks, and OpenAPI docs

## Project Structure

```
go-websocket/
├── backend/                    # Go API server
│   ├── cmd/go-websocket/     # main entry
│   ├── internal/
│   │   ├── config/             # config structs, load, observability
│   │   ├── database/           # pgx pool, migrations (embed)
│   │   ├── errs/               # HTTP error types and constructors
│   │   ├── handler/            # health, openapi, base (typed Handle/HandleNoContent/HandleFile)
│   │   ├── lib/
│   │   │   ├── email/          # Resend client, templates, welcome email
│   │   │   ├── jobs/           # Asynq job service, welcome email task
│   │   │   └── utils/          # small helpers (e.g. PrintJSON)
│   │   ├── logger/             # zerolog + New Relic LoggerService, pgx logger
│   │   ├── middleware/         # CORS, secure, request ID, tracing, context, auth, rate limit, recover, global error
│   │   ├── repository/         # repository layer (currently empty struct)
│   │   ├── router/             # Echo router, system routes registration
│   │   ├── server/             # Server struct (config, DB, Redis, Job, HTTP server)
│   │   ├── service/            # Auth (Clerk), Job service ref
│   │   ├── sqlerr/             # PG error → HTTP error mapping
│   │   └── validation/         # BindAndValidate, Validatable, tag→message mapping
│   ├── static/                 # openapi.html, openapi.json (from packages/openapi gen)
│   ├── templates/emails/       # HTML email templates (e.g. welcome.html)
│   ├── Taskfile.yml            # run, migrations:new, migrations:up, tidy
│   ├── .golangci.yml           # linter config
│   ├── go.mod
│   └── go.sum
├── packages/
│   ├── openapi/                # ts-rest contracts, OpenAPI 3 generation, writes openapi.json
│   ├── zod/                    # shared Zod schemas (e.g. health response)
│   └── emails/                 # (optional) React email templates
├── package.json                # workspace root, turbo scripts
├── turbo.json
└── README.md
```

---

## Backend Highlights

- Framework: Echo v4
- Database: PostgreSQL via pgx
- Realtime: Gorilla WebSocket
- Queue/cache: Redis + Asynq
- Auth scaffolding: Clerk
- Observability: zerolog + New Relic integrations
- Validation: go-playground/validator

## Main API Surface

- `GET /status`
- `GET /docs`
- `GET /ws`
- `GET /matches/`
- `POST /matches`
- `GET /matches/:id/commentary`
- `POST /matches/:id/commentary`

## WebSocket Flow

Connect to `/ws`, then subscribe to a match:

```json
{"type":"subscribe","matchId":1}
```

Successful response:

```json
{"type":"subscribed","matchId":1,"message":"subscribed"}
```

When new commentary is created for that match, subscribed clients receive:

```json
{
  "type": "commentary.created",
  "data": {
    "id": 12,
    "matchId": 1,
    "minute": 18,
    "message": "Goal from close range",
    "createdAt": "2026-04-09T10:00:00Z"
  }
}
```

Match creation is still broadcast globally as `match.created`.

## Getting Started

1. Start PostgreSQL and Redis locally.
2. Go to [app/backend/README.md](/Users/ayushamin/Developer/repos/go-websockets/app/backend/README.md) for backend setup.
3. Run the Go server from `app/backend`.

## Workspace Scripts

From the repo root:

```bash
bun install
bun run dev
```

The backend can also be run directly with Task or Go commands from `app/backend`.

## Documentation

- Backend guide: [app/backend/README.md](/Users/ayushamin/Developer/repos/go-websockets/app/backend/README.md)
- OpenAPI UI: `http://localhost:8080/docs`
- Health check: `http://localhost:8080/status`
