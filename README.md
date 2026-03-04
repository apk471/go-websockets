# Go Boilerplate

A production-ready Go backend boilerplate with Echo, PostgreSQL, Redis, background jobs, observability (New Relic), Clerk auth, typed handlers, OpenAPI docs, and shared TypeScript packages for API contracts and Zod schemas.

---

## Table of Contents

- [Overview](#overview)
- [Project Structure](#project-structure)
- [Backend](#backend)
  - [Entry Point & Startup](#entry-point--startup)
  - [Configuration](#configuration)
  - [Server](#server)
  - [Database](#database)
  - [Router & Middleware](#router--middleware)
  - [Handlers](#handlers)
  - [Errors](#errors)
  - [Logging & Observability](#logging--observability)
  - [Services & Repositories](#services--repositories)
  - [Background Jobs](#background-jobs)
  - [Email](#email)
  - [Validation](#validation)
- [Packages (TypeScript)](#packages-typescript)
- [Tooling](#tooling)
- [Environment Variables](#environment-variables)
- [Running the Project](#running-the-project)
- [Extending the Boilerplate](#extending-the-boilerplate)

---

## Overview

- **API framework:** [Echo v4](https://echo.labstack.com/)
- **Database:** PostgreSQL via [pgx v5](https://github.com/jackc/pgx), [tern](https://github.com/jackc/tern) migrations
- **Cache/queue:** Redis ([go-redis](https://github.com/redis/go-redis)), [Asynq](https://github.com/hibiken/asynq) for background jobs
- **Auth:** [Clerk](https://clerk.com/) via `clerk-sdk-go` (JWT/session validation, user/role/permissions in context)
- **Config:** [Koanf](https://github.com/knadh/koanf) from env with `BOILERPLATE_` prefix, [go-playground/validator](https://github.com/go-playground/validator)
- **Logging:** [zerolog](https://github.com/rs/zerolog) with request-scoped loggers and optional New Relic log forwarding
- **Observability:** New Relic (APM, distributed tracing, log context, nrpgx5, nrecho, nrredis, zerolog writer)
- **API docs:** OpenAPI 3 generated from [ts-rest](https://ts-rest.com/) contracts in `packages/openapi`, served at `/docs` with Scalar
- **Shared types:** Zod schemas and OpenAPI generation in `packages/zod` and `packages/openapi`

---

## Project Structure

```
go-boilerplate/
├── backend/                    # Go API server
│   ├── cmd/go-boilerplate/     # main entry
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

## Backend

### Entry Point & Startup

- **`cmd/go-boilerplate/main.go`**
  - Loads config via `config.LoadConfig()` (env-only, `BOILERPLATE_` prefix).
  - Creates `LoggerService` (New Relic optional) and zerolog logger.
  - Runs DB migrations when `env != "local"` via `database.Migrate(...)`.
  - Builds `server.Server` (DB, Redis, Asynq job service), repositories, services, handlers, router.
  - Sets up HTTP server on `server.Port`, starts it and graceful shutdown on interrupt (30s timeout).
  - Shuts down HTTP server, DB pool, and job server.

### Configuration

- **`internal/config/config.go`**

  - **Config** struct: Primary (env), Server (port, timeouts, CORS origins), Database (host, port, user, password, name, ssl_mode, pool settings), Auth (secret for Clerk), Redis (address), Integration (e.g. Resend API key), Observability (optional).
  - Load: Koanf with `env.Provider("BOILERPLATE_", ".", lowerAndTrimPrefix)` so env vars like `BOILERPLATE_SERVER_PORT` map to `server.port`.
  - Validation with `go-playground/validator`; on failure the process exits.
  - Observability defaults: `DefaultObservabilityConfig()` and override with `observability.service_name`, `observability.environment` from primary env.

- **`internal/config/observability.go`**
  - **ObservabilityConfig:** service_name, environment, logging (level, format, slow_query_threshold), new_relic (license_key, app_log_forwarding_enabled, distributed_tracing_enabled, debug_logging), health_checks (enabled, interval, timeout, checks list).
  - `Validate()`: service_name required, log level in [debug, info, warn, error], slow_query_threshold >= 0.
  - `GetLogLevel()`: uses environment default (e.g. debug for development) when level empty.
  - `IsProduction()`: true when environment == "production".

### Server

- **`internal/server/server.go`**
  - **Server** holds: Config, Logger, LoggerService, DB (*database.Database), Redis (go-redis Client), Job (*jobs.JobService), and the HTTP server.
  - **New:** Creates DB (with optional New Relic nrpgx5 tracer, local pgx tracelog in local env), Redis client (with optional nrredis hook), Job service (Asynq client + server), starts the job server (registers task handlers).
  - **SetupHTTPServer(handler):** Sets `http.Server` (Addr from config, read/write/idle timeouts).
  - **Start:** Calls `ListenAndServe()`.
  - **Shutdown:** Shuts down HTTP server, closes DB pool, stops job server.

### Database

- **`internal/database/database.go`**

  - **Database** wraps `*pgxpool.Pool` and a logger.
  - **New:** Builds DSN (password URL-encoded), parses pool config; if LoggerService has New Relic app, sets `nrpgx5.NewTracer()`; in local env adds pgx-zerolog tracelog (or multi-tracer with both). Pool created with `pgxpool.NewWithConfig`, then ping with 10s timeout.
  - **Close:** Logs and closes pool.

- **`internal/database/migrator.go`**

  - Uses embedded `migrations/*.sql` and [tern](https://github.com/jackc/tern) with table `schema_version`.
  - **Migrate:** Connects with same DSN, creates tern migrator, loads migrations from embed, runs migrate, logs version.

- **`internal/database/migrations/001_setup.sql`**
  - Placeholder migration (empty up/down). New migrations: `task migrations:new name=something`.

### Router & Middleware

- **`internal/router/router.go`**

  - **NewRouter:** Creates Echo instance, sets **GlobalErrorHandler** from middlewares, then applies in order:
    - Rate limiter (20 req/s, memory store), DenyHandler returns 429 and records rate limit hit in New Relic.
    - CORS (origins from config), Secure(), RequestID (X-Request-ID, uuid if missing), NewRelic (nrecho), EnhanceTracing (request id, user id, status code, NoticeError), ContextEnhancer (request-scoped logger with request_id, method, path, ip, trace context, user_id, user_role), RequestLogger, Recover.
  - Registers system routes via `registerSystemRoutes`; `/api/v1` group exists for future versioned routes.

- **`internal/router/system.go`**

  - **GET /status** → HealthHandler.CheckHealth
  - **/static** → static files (e.g. openapi.json)
  - **GET /docs** → OpenAPIHandler.ServeOpenAPIUI (serves static/openapi.html, which loads /static/openapi.json and Scalar)

- **Middleware details**
  - **global (internal/middleware/global.go):** CORS, Secure, RequestLogger (status, latency, URI, etc., uses context logger and request_id/user_id), Recover, GlobalErrorHandler (sqlerr handling, then HTTP/echo error → JSON response, logging).
  - **auth (auth.go):** Clerk `WithHeaderAuthorization`; on success sets `user_id`, `user_role`, `permissions` in context; on failure returns 401 JSON.
  - **context (context.go):** Puts request-scoped logger (with request_id, method, path, ip, trace id/span id if New Relic, user_id/user_role) in context; `GetLogger(c)`, `GetUserID(c)`.
  - **request_id (request_id.go):** Reads or generates X-Request-ID, sets in context and response header.
  - **tracing (tracing.go):** Wraps nrecho middleware; EnhanceTracing adds http.real_ip, http.user_agent, request.id, user.id, http.status_code, and NoticeError on handler error.
  - **rate_limit (rate_limit.go):** RecordRateLimitHit(endpoint) for New Relic custom event when rate limit is hit.

### Handlers

- **`internal/handler/handlers.go`**

  - **Handlers** contains Health and OpenAPI. **NewHandlers** builds them from server and services.

- **`internal/handler/base.go`**

  - **Handler** is a base with server reference.
  - **HandlerFunc[Req, Res], HandlerFuncNoContent[Req]** for typed handlers.
  - **ResponseHandler** interface: Handle(c, result), GetOperation(), AddAttributes(txn, result). Implementations: **JSONResponseHandler**, **NoContentResponseHandler**, **FileResponseHandler** (filename, content-type, blob).
  - **handleRequest:** Binds and validates payload with `validation.BindAndValidate`, runs handler, records validation/handler duration and status on New Relic transaction, uses context logger; on error uses `nrpkgerrors.Wrap` and returns err; on success calls responseHandler.Handle(c, result).
  - **Handle**, **HandleNoContent**, **HandleFile** wrap handler funcs with handleRequest and the appropriate response handler.

- **`internal/handler/health.go`**

  - **CheckHealth:** Returns JSON with status (healthy/unhealthy), timestamp, environment, and **checks** (database ping, redis ping when Redis not nil). On DB/Redis failure sets check to unhealthy and records **HealthCheckError** custom event in New Relic. Returns 503 when unhealthy.

- **`internal/handler/openapi.go`**
  - **ServeOpenAPIUI:** Serves `static/openapi.html` as HTML (Cache-Control: no-cache). The HTML page loads Scalar with `/static/openapi.json`.

### Errors

- **`internal/errs/type.go`**

  - **HTTPError:** Code, Message, Status, Override, Errors (field-level), Action (e.g. redirect). Implements `error` and `Is(*HTTPError)`.

- **`internal/errs/http.go`**

  - Constructors: **NewUnauthorizedError**, **NewForbiddenError**, **NewBadRequestError**, **NewNotFoundError**, **NewInternalServerError**, **ValidationError**. **MakeUpperCaseWithUnderscores** for code formatting.

- **`internal/sqlerr/error.go`**

  - **Code** constants: Other, NotNullViolation, ForeignKeyViolation, UniqueViolation, CheckViolation, etc., with **MapCode** from PostgreSQL codes (23502, 23503, 23505, …).
  - **Severity** and **Error** struct (Code, Severity, Message, TableName, ColumnName, ConstraintName, …). **ConvertPgError** from pgconn.PgError.

- **`internal/sqlerr/handler.go`**
  - **HandleError(err):** If already HTTPError, return as-is. If pgconn.PgError, convert and map to user-facing message and **errs** (BadRequest with optional field errors for not_null, NotFound for no rows, InternalServerError for rest). **ErrNoRows** / **sql.ErrNoRows** → NotFound. Otherwise InternalServerError.
  - Global error handler (in global.go) calls **sqlerr.HandleError** for non-HTTP errors before formatting response.

### Logging & Observability

- **`internal/logger/logger.go`**

  - **LoggerService:** Holds optional New Relic Application. **NewLoggerService** from ObservabilityConfig (app name, license, log forwarding, distributed tracing, optional debug logger). **Shutdown** flushes New Relic.
  - **NewLoggerWithService:** Builds zerolog with level from config, time format, pkgerrors stack marshaler; in production with JSON format and NR app, wraps writer with **zerologWriter** for log forwarding; otherwise console writer in dev. Logger has service, environment; in non-production adds Stack().
  - **WithTraceContext:** Adds trace.id and span.id from New Relic transaction to logger.
  - **NewPgxLogger,** **GetPgxTraceLogLevel:** Used for local DB query logging when env is local.

- New Relic integrations used: main agent, nrecho-v4, nrpgx5, nrredis-v9, nrpkgerrors, logcontext-v2/zerologWriter.

### Services & Repositories

- **`internal/repository/repositories.go`**

  - **Repositories** is an empty struct; **NewRepositories(server)** returns it. Ready for DB-backed repositories.

- **`internal/service/services.go`**

  - **Services** has Auth and Job. **NewServices(server, repos)** builds AuthService (sets Clerk key from config) and attaches server’s Job service.

- **`internal/service/auth.go`**
  - **AuthService** only sets **Clerk** secret key from config; actual auth is in middleware via Clerk SDK.

### Background Jobs

- **`internal/lib/jobs/job.go`**

  - **JobService:** Asynq client + server (Redis addr from config). Queues: critical (6), default (3), low (1). **Start:** Registers **TaskWelcome** handler, starts server. **Stop:** Shutdown server, close client.

- **`internal/lib/jobs/email_task.go`**

  - **TaskWelcome** = `"email:welcome"`. **WelcomeEmailPayload:** To, FirstName. **NewWelcomeEmailTask** builds asynq task with MaxRetry(3), Queue("default"), Timeout(30s).

- **`internal/lib/jobs/handlers.go`**
  - **InitHandlers:** Creates email client from config and logger. **handleWelcomeEmailTask:** Unmarshals payload, calls **emailClient.SendWelcomeEmail(to, firstName)**, logs success/failure.

### Email

- **`internal/lib/email/client.go`**

  - **Client** wraps Resend client. **SendEmail(to, subject, templateName, data):** Loads HTML from `templates/emails/{templateName}.html`, executes with data, sends via Resend (from: Boilerplate &lt;onboarding@resend.dev&gt;).

- **`internal/lib/email/emails.go`**

  - **SendWelcomeEmail(to, firstName):** Uses TemplateWelcome and data UserFirstName.

- **`internal/lib/email/template.go`**

  - **Template** type; **TemplateWelcome** = `"welcome"`.

- **`internal/lib/email/preview.go`**

  - **PreviewData** map for template preview (e.g. welcome → UserFirstName: "John").

- **`templates/emails/welcome.html`**
  - Go HTML template with `{{.UserFirstName}}`, “Welcome to Boilerplate!”, CTA, support link.

### Validation

- **`internal/validation/utils.go`**
  - **Validatable** interface: `Validate() error`.
  - **BindAndValidate(c, payload):** Binds payload with `c.Bind(payload)`, then validates with `validateStruct(payload)`. On bind error returns BadRequest with message; on validation error returns BadRequest with **extractValidationErrors** (field + message per tag).
  - **extractValidationErrors:** Handles **validator.ValidationErrors** (required, min, max, oneof, email, e164, uuid, uuidList, dive) and custom **CustomValidationErrors**.
  - **IsValidUUID:** regex for UUID string.

---

## Packages (TypeScript)

- **`packages/zod`**

  - Shared Zod schemas; **@anatine/zod-openapi** for OpenAPI metadata. Exports e.g. **ZHealthResponse** (status, timestamp, environment, checks.database, checks.redis).

- **`packages/openapi`**

  - **ts-rest** contract: health contract (GET /status, response ZHealthResponse). **apiContract** aggregates contracts.
  - **generateOpenApi** with security (bearerAuth, x-service-token), operationMapper for security metadata. **gen.ts** string-replaces custom “file” type with OpenAPI binary, then writes **openapi.json** to repo and (in script) to `../../apps/backend/static/openapi.json` For this repo, add or change the output path in `packages/openapi/src/gen.ts` to `../../backend/static/openapi.json` so `/docs` loads the generated spec.
  - Backend serves `/docs` with Scalar and `/static/openapi.json` so docs stay in sync when you run the openapi package gen.

- **`packages/emails`**
  - Optional React-based email templates (e.g. welcome.tsx); can be used to generate or mirror HTML for backend.

---

## Tooling

- **Taskfile (backend/Taskfile.yml)**

  - **run:** `go run ./cmd/go-boilerplate`
  - **migrations:new:** `tern new -m ./internal/database/migrations {{.NAME}}` (requires `name=...`)
  - **migrations:up:** `tern migrate -m ./internal/database/migrations --conn-string {{.BOILERPLATE_DB_DSN}}` (with confirm)
  - **tidy:** `go fmt ./...`, `go mod tidy`, `go mod verify`

- **Golangci-lint (backend/.golangci.yml)**

  - Large set of linters (errcheck, staticcheck, gosec, revive, gocritic, etc.) with sensible limits (e.g. cyclop, funlen, gocognit). **gomodguard** blocks old uuid/protobuf modules. **exhaustruct** exclusions for std and third-party structs. **govet** with shadow strict.

- **Root**
  - **package.json** + **turbo.json**: Workspaces `apps/*`, `packages/*`; scripts: build, dev, format, lint, typecheck, clean. Turbo runs tasks with dependency order (^build, etc.).

---

## Environment Variables

All backend config is read from environment with prefix **BOILERPLATE\_**. Keys are lowercased and the prefix is stripped (e.g. `BOILERPLATE_SERVER_PORT` → `server.port`). Nested keys use underscore (e.g. `BOILERPLATE_DATABASE_HOST`).

Example (replace values as needed):

```bash
# Primary
BOILERPLATE_PRIMARY_ENV=local

# Server
BOILERPLATE_SERVER_PORT=8080
BOILERPLATE_SERVER_READ_TIMEOUT=30
BOILERPLATE_SERVER_WRITE_TIMEOUT=30
BOILERPLATE_SERVER_IDLE_TIMEOUT=60
BOILERPLATE_SERVER_CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Database
BOILERPLATE_DATABASE_HOST=localhost
BOILERPLATE_DATABASE_PORT=5432
BOILERPLATE_DATABASE_USER=postgres
BOILERPLATE_DATABASE_PASSWORD=secret
BOILERPLATE_DATABASE_NAME=boilerplate
BOILERPLATE_DATABASE_SSL_MODE=disable
BOILERPLATE_DATABASE_MAX_OPEN_CONNS=25
BOILERPLATE_DATABASE_MAX_IDLE_CONNS=5
BOILERPLATE_DATABASE_CONN_MAX_LIFETIME=300
BOILERPLATE_DATABASE_CONN_MAX_IDLE_TIME=60

# Auth (Clerk)
BOILERPLATE_AUTH_SECRET_KEY=sk_test_...

# Redis
BOILERPLATE_REDIS_ADDRESS=localhost:6379

# Integration (Resend)
BOILERPLATE_INTEGRATION_RESEND_API_KEY=re_...

# Observability (optional)
BOILERPLATE_OBSERVABILITY_SERVICE_NAME=boilerplate
BOILERPLATE_OBSERVABILITY_ENVIRONMENT=development
BOILERPLATE_OBSERVABILITY_LOGGING_LEVEL=debug
BOILERPLATE_OBSERVABILITY_LOGGING_FORMAT=json
BOILERPLATE_OBSERVABILITY_NEW_RELIC_LICENSE_KEY=
BOILERPLATE_OBSERVABILITY_NEW_RELIC_APP_LOG_FORWARDING_ENABLED=true
BOILERPLATE_OBSERVABILITY_NEW_RELIC_DISTRIBUTED_TRACING_ENABLED=true
BOILERPLATE_OBSERVABILITY_NEW_RELIC_DEBUG_LOGGING=false
BOILERPLATE_OBSERVABILITY_HEALTH_CHECKS_ENABLED=true
BOILERPLATE_OBSERVABILITY_HEALTH_CHECKS_INTERVAL=30s
BOILERPLATE_OBSERVABILITY_HEALTH_CHECKS_TIMEOUT=5s
BOILERPLATE_OBSERVABILITY_HEALTH_CHECKS_CHECKS=database,redis
```

For **Taskfile** migrations: set **BOILERPLATE_DB_DSN** (e.g. `postgres://user:pass@localhost:5432/boilerplate?sslmode=disable`).

---

## Running the Project

1. **Prerequisites:** Go 1.25+, PostgreSQL, Redis, Node/Bun for packages.
2. **Env:** Copy or set the variables above (e.g. `.env` and use `godotenv/autoload` or export).
3. **Backend:**
   - From repo root: `cd backend && task run` (or `go run ./cmd/go-boilerplate`).
   - Migrations (non-local): run automatically on startup; for manual run: `BOILERPLATE_DB_DSN=... task migrations:up`.
   - New migration: `task migrations:new name=add_users_table`.
4. **OpenAPI:** From repo root, build/openapi gen so `backend/static/openapi.json` exists (e.g. `cd packages/zod && bun run build && cd ../openapi && bun run gen` if gen writes there). Then open `http://localhost:8080/docs`.
5. **Health:** `GET http://localhost:8080/status`.

---

## Extending the Boilerplate

- **New route:** Add to `router/system.go` or a versioned group in `router/router.go`; use `middlewares.Auth.RequireAuth(next)` for protected routes.
- **New handler:** Implement handler func with request/response types implementing **Validatable** where needed; register with **Handle**, **HandleNoContent**, or **HandleFile** from `handler/base.go`.
- **New migration:** `task migrations:new name=your_change` in `backend`, then edit the new file under `internal/database/migrations/`.
- **New job:** Define task type and payload in `internal/lib/jobs`, add handler in `job.go` (mux.HandleFunc), enqueue via `Job.Client.Enqueue(...)` from services/handlers.
- **New email template:** Add template name in `internal/lib/email/template.go`, HTML in `templates/emails/`, and send method in `internal/lib/email/`.
- **OpenAPI:** Add contract in `packages/openapi/src/contracts/`, add Zod types in `packages/zod`, run openapi package gen and copy/openapi.json to `backend/static/` if needed.
- **Config:** Add fields to `config.Config` or `ObservabilityConfig` and corresponding env vars with `BOILERPLATE_` prefix.
