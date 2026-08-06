# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A minimal boilerplate/template for a Go REST API built on [Fiber v3](https://github.com/gofiber/fiber) (v3.4.0). It exists to be copied/forked as the starting point for real services — the packages under `internal/api/` contain only "sample" placeholder code demonstrating the intended structure. The one fully worked feature is [`examples/crud`](examples/crud/README.md) — a users CRUD on `migrations/users.sql`, compiled but deliberately not mounted.

**Requires Go 1.25+** — Fiber v3's own `go.mod` declares `go 1.25.0`, so the toolchain floor is not optional.

Environment variables are loaded by `internal/config/dotenv` when `GO_ENV` is unset (see `cmd/api/main.go` `init()`). It calls bare `godotenv.Load()`, which reads **`.env`** — not `.env.development`, despite the name. `.env` is gitignored; `.env.development` is the tracked template you copy from. Adding a var to `.env.development` alone has no runtime effect.

Required vars: `GO_ENV`, `API_NAME`, `API_VERSION`, `PORT`. Postgres vars: `DATABASE_URL` (overrides all others), or `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE`, plus optional pool tuning `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME`, `DB_CONNECT_TIMEOUT`.

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
- **`internal/config/*`** — one package per cross-cutting concern (cors, compress, helmet, requestId, logger, dotenv), each exposing a single `SetXMiddleware(app)`-style function called from `main.go`. Add new global middleware as a new package here, then wire it in `main.go`.
- **`pkg/db`** — Postgres connection pool on `pgx/v5` + `pgxpool`. Holds a process-wide `*pgxpool.Pool`; `Init(ctx, ConfigFromEnv())` opens and pings it (a bad host/password fails at startup rather than on the first request), `Pool()` returns it for queries, `Ping(ctx)` backs `/healthz`, and `Close()` runs during graceful shutdown *after* `app.Shutdown()` so no handler holds a connection from a closed pool. `Pool()` panics if `Init` never ran. `Config.dsn()` URL-encodes the password, so `@` / `:` / `/` in passwords are safe.

  `tx.go` adds the transaction layer. Repositories should take a `db.Querier` (the read/write surface shared by `*pgxpool.Pool` and `pgx.Tx`) and resolve it with `db.Q(ctx)`, **not** `db.Pool()` — `Q` returns the in-flight transaction when there is one and the pool otherwise, so the same function works inside or outside a tx. `db.ExecTx(ctx, fn)` commits on nil error and rolls back otherwise (`ExecTxOptions` for isolation levels); it puts the tx on the context it hands `fn`, so a nested `ExecTx` becomes a **savepoint** rather than a second transaction. Rollback runs on `context.WithoutCancel`, so a cancelled request still cleans up instead of leaving the connection for the pool to reset.
- **`internal/api/routes/`** — `routes.go` has the top-level `SetRoutes(router fiber.Router)` that groups sub-routers by feature path (e.g. `/sample-routes`) and delegates to a per-feature `Set<Feature>Route` function (see `sample_route.go`). Add a new feature by creating `<feature>_route.go` here and registering it in `routes.go`.
- **`internal/api/controllers/`** — Fiber handlers (`func(c fiber.Ctx) error` — by value, `Ctx` is an interface in v3). Should stay thin: parse/validate input, call into `internal/api/services`, format output via `internal/api/presenters`.
- **`internal/api/services/`** — business logic, called by controllers. Owns transactions (`db.ExecTx`) and pagination defaults. Currently empty/placeholder; `examples/crud/services` is the worked version.
- **`internal/api/schemas/`** — request/response struct definitions, validated via `github.com/go-playground/validator/v10` struct tags, split into `requestbody/` and `responsebody/`. Currently empty/placeholder; `examples/crud/schemas` is the worked version.
- **`internal/api/validators/`** — wraps `go-playground/validator`. `Init()` must run once before use (already called in `main.go`). Use `ParseAndValidateBody` / `ParseAndValidateQueryParam` in controllers to bind+validate in one step, and `ValidateUuid` for standalone UUID checks. Note this package validates *independently* of Fiber's own `StructValidator` hook (which is not configured on the app), so `go-playground` tags are only enforced when you route through these helpers.
- **`internal/api/presenters/response.go`** — the single source of truth for the JSON response envelope. Always shape controller output through `ResponseSuccess(data)` or `ResponseSuccessListData(data, currentPage, currentPageTotalItem, totalPage)` (pass `-1` for the pagination args when pagination doesn't apply) rather than constructing `fiber.Map` responses ad hoc. Error responses use the same envelope shape but are only constructed centrally in the `main.go` `ErrorHandler`.
- **`internal/api/middlewares/`** — route-level (as opposed to global) middleware; currently empty/placeholder.

### Response envelope contract

Every JSON response (success or error) follows: `{ "timestamp", "status" (1 success / 0 fail), "items", "error" }`. Keep any new endpoint consistent with this shape via the presenters package instead of hand-rolling responses.

### Tests

```bash
go test ./...                    # SQL generation and other pure logic. No database.
go test -tags=integration ./...  # everything else. Needs Postgres and the vars in .env
```

Anything provable from the generated SQL alone stays untagged so CI runs it on every commit; anything needing a real server goes behind `//go:build integration`. A plain `go test ./...` must stay green with no container running.

Copy the layout of `examples/crud/tests`:

- **One `tests/` package per feature**, outside the packages under test — so anything a test touches must be exported (`repositories.Builder`, `BuildListUsersQuery` are public only for this). Unexportable values (pagination defaults) become literals with a comment naming the real definition.
- **Assert against the function production calls**, extracting one if needed. A test that rebuilds the SQL itself only proves it agrees with itself — that already happened here once.
- **Isolate by marker, never truncation.** `testsupport.Marker(t)` gives a random prefix, carried on every row, filtered in every query, deleted in `t.Cleanup`. No `_` or `%` in it — both are `LIKE` wildcards. (Tx-rollback isolation fails here: `t.Fatal` is `runtime.Goexit`, which skips the deferred rollback.)
- **`testsupport.Main` owns `TestMain`** — `go test` runs in the package dir, so `.env` has to be found by walking up. Missing database under the tag panics rather than skips.
- Test each thing at the layer where it is reachable; the bulk-create rollback is a service test because `max=200` rejects the bad value before HTTP could reach the database.
- **Confirm the tests can fail** — flip one thing they cover (`ILIKE`→`LIKE`, drop the `ORDER BY` tiebreaker), check the right test fails, put it back.

### Logging

`internal/config/logger` emits one JSON object per request. Mount it **before** any route on the group — Fiber only applies middleware to routes registered after it, so anything above `SetLoggerMiddlewareJSON` is silently absent from the log (this already bit the `/api/{version}` root endpoint).