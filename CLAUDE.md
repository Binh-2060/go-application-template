# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A minimal boilerplate/template for a Go REST API built on [Fiber v3](https://github.com/gofiber/fiber) (v3.4.0). It exists to be copied/forked as the starting point for real services — most packages currently contain only "sample" placeholder code demonstrating the intended structure.

**Requires Go 1.25+** — Fiber v3's own `go.mod` declares `go 1.25.0`, so the toolchain floor is not optional.

## Common commands

```bash
# Run the app locally with live reload (uses .air.toml)
air

# Run without live reload
go run cmd/api/main.go

# Build the binary (matches Dockerfile build)
go build -o ./tmp/main cmd/api/main.go

# Tidy/verify deps
go mod tidy

# Run tests (no test files exist yet, but this is the invocation)
go test ./...
go test ./api/services/... -run TestName -v   # single package / test

# Docker build (multi-stage, static binary on alpine)
docker build --build-arg API_VERSION=v1 --build-arg BUILD_DATE=$(date +%F) -t fiber-api .
```

Environment variables are loaded from `.env.development` via `config/dotenv` when `GO_ENV` is unset (see `cmd/api/main.go` `init()`). Required vars: `GO_ENV`, `API_NAME`, `API_VERSION`, `PORT`.

## Fiber v3 conventions (migrated from v2)

These are the v3 API rules that differ from the v2 idioms found in most online Fiber examples — follow them when adding code:

- **Handlers take `fiber.Ctx` by value, not `*fiber.Ctx`.** `Ctx` is an interface in v3. This applies to route handlers, the `ErrorHandler`, and any helper that accepts a context (e.g. `validators.ParseAndValidateBody`).
- **Binding replaces parsing.** Use `c.Bind().Body(out)` and `c.Bind().Query(out)` instead of v2's `c.BodyParser` / `c.QueryParser`. For path params use `c.Bind().URI(out)` with `uri:"..."` struct tags (v2 used `params:"..."`). Typed query helpers (`QueryInt`, `QueryBool`, …) are gone — use the generic `fiber.Query[T](c, key)`.
- **Logger config uses `Stream`, not `Output`**, for the destination writer. `CustomTags` and the `LogFunc` signature (`func(output logger.Buffer, c fiber.Ctx, data *logger.Data, extraParam string) (int, error)`) are otherwise unchanged from v2 apart from the ctx type.
- **Request ID is not in Locals.** v3's requestid middleware stores the value under an unexported context key, so `${locals:requestid}` silently logs empty. The middleware auto-registers global logger tags, so use `${requestid}` in a log format string, and `requestid.FromContext(c)` in code.
- `app.Listen(addr, ...ListenConfig)` is variadic, so the plain `app.Listen(":"+port)` call still works; pass a `fiber.ListenConfig` for TLS or graceful-context options (the v2 `ListenTLS*` methods were removed).

## Architecture

Request flow: `main.go` → global middleware → versioned route group → feature routes → controller → service → presenter.

- **`cmd/api/main.go`** — composition root. Builds the Fiber app, wires global middleware in a fixed order (CORS → request ID → validators init → compress → helmet), mounts everything under `/api/{API_VERSION}`, adds inline `/` (build info) and `/healthz` endpoints, then delegates feature routes to `routes.SetRoutes`. Also owns graceful shutdown via signal handling. The Fiber `ErrorHandler` here is the single place uncaught errors are converted to the standard JSON error envelope.
- **`config/*`** — one package per cross-cutting concern (cors, compress, helmet, requestId, logger, dotenv), each exposing a single `SetXMiddleware(app)`-style function called from `main.go`. Add new global middleware as a new package here, then wire it in `main.go`.
- **`api/routes/`** — `routes.go` has the top-level `SetRoutes(router fiber.Router)` that groups sub-routers by feature path (e.g. `/sample-routes`) and delegates to a per-feature `Set<Feature>Route` function (see `sample_route.go`). Add a new feature by creating `<feature>_route.go` here and registering it in `routes.go`.
- **`api/controllers/`** — Fiber handlers (`func(c *fiber.Ctx) error`). Should stay thin: parse/validate input, call into `api/services`, format output via `api/presenters`.
- **`api/services/`** — business logic, called by controllers. Currently empty/placeholder.
- **`api/schemas/`** — request/response struct definitions, validated via `github.com/go-playground/validator/v10` struct tags. Currently empty/placeholder.
- **`api/validators/`** — wraps `go-playground/validator`. `Init()` must run once before use (already called in `main.go`). Use `ParseAndValidateBody` / `ParseAndValidateQueryParam` in controllers to bind+validate in one step, and `ValidateUuid` for standalone UUID checks. Note this package validates *independently* of Fiber's own `StructValidator` hook (which is not configured on the app), so `go-playground` tags are only enforced when you route through these helpers.
- **`api/presenters/response.go`** — the single source of truth for the JSON response envelope. Always shape controller output through `ResponseSuccess(data)` or `ResponseSuccessListData(data, currentPage, currentPageTotalItem, totalPage)` (pass `-1` for the pagination args when pagination doesn't apply) rather than constructing `fiber.Map` responses ad hoc. Error responses use the same envelope shape but are only constructed centrally in the `main.go` `ErrorHandler`.
- **`api/middlewares/`** — route-level (as opposed to global) middleware; currently empty/placeholder.

### Response envelope contract

Every JSON response (success or error) follows: `{ "timestamp", "status" (1 success / 0 fail), "items", "error" }`. Keep any new endpoint consistent with this shape via the presenters package instead of hand-rolling responses.

### Logging

`config/logger` configures a single-line JSON access log per request (method, status, latency, path, request ID, request body, etc.) and is mounted after the API root/healthz routes but before `routes.SetRoutes` in `main.go` — so route ordering there matters if you need requests logged. The `customReqBody` custom tag renders JSON and `multipart/form-data` bodies inline; it logs bodies verbatim, so scrub or disable it before sending logs anywhere sensitive.
