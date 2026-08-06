---
name: new-feature
description: Scaffold a new CRUD feature from a table in migrations/, following the layer structure, conventions and error handling of examples/crud. Use when asked to add a feature, resource, endpoint set or CRUD for a table.
---

# New CRUD feature from a migration

Build a feature the same way `examples/crud` is built, driven by a table in `migrations/`.

`examples/crud` is the reference implementation, not a template to paste. **Read it before writing anything** — it is the source of truth for structure and style, and it will have drifted from this file. This document only records the decisions the code cannot show you.

## 1. Establish inputs

**Ask the user these two questions first, with `AskUserQuestion`, before writing any code.** Both change what gets built, and neither is derivable from the prompt. Ask them in one call.

1. **Which migration file?** Run `ls migrations/` and offer each file as an option, plus a final **"A new migration"** option. **Ask even when there is only one file** — a one-option question is still the confirmation that the user meant that table, and this scaffolds a lot of code off it. The options are a convenience, not a restriction: `AskUserQuestion` always carries a free-text "Other" choice, so say in the question text that the user may type a filename directly. If the user's prompt named a table with no matching `migrations/*.sql`, still ask — with that name offered as the new-migration option.

   **If they pick "A new migration" (or name a file that does not exist):** ask a second `AskUserQuestion` for the **filename** before anything else — offer `migrations/<table>.sql` spellings derived from any resource name already mentioned, and rely on "Other" for a name you could not guess. Only once the filename is fixed, ask for the columns in prose (name, Postgres type, nullability, default — not enumerable as options). Write the file, show it, and get confirmation before scaffolding. The schema drives everything below, so never guess columns.
2. **Generate tests?** Options: yes (full suite per §5) / no (feature code only). Default recommendation is yes. If no, skip §5 entirely and do not create a `tests/` folder — but still run the build/vet checks in §6.

**Question 2 is asked every single run, without exception.** Nothing in a feature request implies a tests answer, so there is nothing to derive it from. If the user's prompt already named the table, drop question 1 and ask question 2 alone — a one-question call is still a call. The only way to skip a question is for the user to have already answered *that* question explicitly.

Do not skip the call because the answer looks obvious, because only one migration exists, or because you are confident in a default. Ask, wait for the answer, then build.

Then read the chosen `migrations/*.sql` before writing anything.

Derive the rest without asking:

- **Where it goes.** A real feature belongs in `internal/api/`, which already has `models/`, `repositories/`, `controllers/`, `services/`, `schemas/`, `routes/`. Only put it under `examples/` if it is another teaching example.
- **Singular resource name** (`user`) and **plural route path** (`/users`) — from the table name.

## 2. Files to create

For resource `<thing>` in `internal/api/`:

| File | Holds |
| --- | --- |
| `models/<thing>.go` | Struct mirroring the table, one field per column |
| `schemas/requestbody/<thing>.go` | `Create<Thing>`, `Create<Things>`, `Update<Thing>`, `List<Things>` |
| `schemas/responsebody/<thing>.go` | Wire struct + `New<Thing>` / `New<Things>` mappers |
| `repositories/<thing>.go` | SQL only, sentinels, `<Thing>Filter` |
| `services/<thing>.go` | Orchestration, transactions, pagination maths |
| `controllers/<thing>.go` | Bind → validate → service → presenter |
| `routes/<thing>.go` | `Set<Thing>Route(router fiber.Router)` |
| `tests/*_test.go` | All tests for the feature, one package — **only if the user asked for tests** (see §5) |
| `testsupport/testsupport.go` | `TestMain` + marker helper for this feature — only with tests (see §5) |

Then register in `routes/routes.go`:

```go
thingRoutes := router.Group("/things")
SetThingRoute(thingRoutes)
```

Mounting is part of the job for a real feature. (`examples/crud` is deliberately left unmounted; that is the exception.)

## 3. Column → Go type

| Postgres | Go | Note |
| --- | --- | --- |
| `uuid` | `string` | Validate with `validators.ValidateUuid` at the edge |
| `varchar(n)` / `text` | `string` | Carry `n` into a `max=n` validate tag |
| `int` / `bigint` | `int` / `int64` | |
| `numeric` | `pgtype.Numeric` | Never `float64` for money |
| `boolean` | `bool` | |
| `timestamptz` | `time.Time` | |
| `jsonb` | `[]byte` or a typed struct | |
| any **nullable** column | pointer (`*string`) | A `NOT NULL`-less column cannot scan into a value type |

Rules that follow from the schema:

- **Follow the column's spelling, even when it is wrong.** `users.surename` is misspelled and the code matches it. Diverging silently makes the mapping a lie; fix it in a migration if it matters.
- Columns with a `DEFAULT` (`id`, `created_at`) are **not** in the create request body — the database supplies them and `RETURNING` reads them back.
- `NOT NULL` without a default → `validate:"required"` on create.

## 4. Layer rules

These are the parts that go wrong when copied carelessly.

**models** — field order must match `<thing>Columns` in the repository, so one scan order serves every query.

**repositories**

- Resolve the connection with `db.Q(ctx)`, **never** `db.Pool()`. `Q` returns the in-flight transaction when there is one and the pool otherwise, so the same function works inside or outside `db.ExecTx`.
- Declare `var <thing>Columns = []string{...}` once and derive the SQL fragment from it. Never write a second column list.
- Return sentinels (`Err<Thing>NotFound`, `ErrNoUpdateFields`), never HTTP status codes. Nothing here knows a web server exists.
- Translate `pgx.ErrNoRows` → `Err<Thing>NotFound` in one shared `scan<Thing>` helper.
- `DELETE` does not produce `ErrNoRows`; read absence off `tag.RowsAffected() == 0`.
- Wrap every other error with `fmt.Errorf("...: %w", err)`.
- **Raw SQL when the statement is fixed** (create, get-by-id, delete). **Squirrel when the SQL is only known at runtime** (filters, partial `SET`). Do not use a builder to construct a constant.
- Squirrel is a **string builder only** — always `.ToSql()`, then hand the SQL and args to `db.Q(ctx)`. `RunWith`, `.Query` and `.Scan` are `database/sql`-based and this project uses native pgx.
- Build from `psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)`. Squirrel defaults to `?` (MySQL).
- Filter values stay bound arguments (`sq.ILike{"name": "%"+v+"%"}` → `name ILIKE $1`). Never concatenate a value into SQL text.
- `ORDER BY` for a paged list needs a unique tiebreaker (`created_at DESC, id DESC`), or rows repeat across pages.
- Return `make([]T, 0, n)` so an empty list encodes as `[]`, not `null`.
- Check `rows.Err()` after the loop — `Next()` reports a mid-stream failure only as "no more rows".

**services**

- Re-export the repository sentinels (`var ErrXNotFound = repositories.ErrXNotFound`) so controllers need not import the repository. Same value, so `errors.Is` still matches.
- Multi-row writes go inside `db.ExecTx` so a failure on the last row rolls back the rest.
- Read a list's count and its page inside **one** `db.ExecTx`, sharing one filter value, so the total always describes the rows returned.
- Default the pagination here (`page=1`, `per_page=20`), not in the controller.
- Ceiling division without floats: `(total + perPage - 1) / perPage`.
- Return a `Paged<Things>` struct rather than four loose ints.

**schemas**

- Response types are separate from models on purpose — the model tracks the table, the schema tracks the API contract.
- `Update<Thing>` uses plain, **required** fields — `PATCH` means "replace", not "merge" (`examples/crud`'s `UpdateUser` follows this). The repository's `Update<Thing>` can still take `*string`/pointer args per column and skip nils in the `SET` clause; only expose that as `omitempty` pointers on the request schema if the feature genuinely needs partial-field PATCH.
- Query structs use `query:"..."` tags (Fiber v3 binds via `c.Bind().Query`).
- `min`/`max` tags should mirror the column constraints so an oversized value returns a readable 400 instead of a database error.

**controllers**

- `func(c fiber.Ctx) error` — **by value**. `Ctx` is an interface in v3, not a struct pointer.
- `validators.ParseAndValidateBody(c, &body)` / `ParseAndValidateQueryParam`. Fiber's own `StructValidator` hook is not configured, so tags are enforced *only* through these helpers.
- Pass `c.Context()` (a `context.Context`) to services; `c.RequestCtx()` is the fasthttp one.
- Validate a `:id` param with `validators.ValidateUuid` before querying — 400 for malformed input, 404 for valid input with no row.
- Shape output with `presenters.ResponseSuccess(data)` or `ResponseSuccessListData(data, page, pageTotal, totalPage)`. Never hand-roll a `fiber.Map` response.
- Map sentinels to status in one small `toHTTPError` switch; **return** anything unrecognised untouched. `main.go`'s `ErrorHandler` is the only place an error becomes JSON.
- There is no exception/error-class layer in this project. Sentinels plus `fiber.NewError` is the whole mechanism — do not introduce one.

**routes** — register literal segments before param routes (`/bulk` before `/:id`), or the param route swallows them.

## 5. Tests

**Skip this whole section if the user answered "no tests" in §1.** Otherwise write the full suite below — all five files, both tags. A single happy-path test is not what was asked for.

**Tests go in a `tests/` folder next to the feature's other packages**, not beside each file they exercise — `examples/crud/tests` is the working example. One package, `main_test.go` holding `TestMain` and the marker helper, one file per layer under test.

That layout means the tests are outside the packages they test, so **anything they touch must be exported**. Export deliberately and say why in the doc comment (`repositories.BuildListUsersQuery` is public only so a test can assert the SQL). Where a value cannot reasonably be exported — the pagination defaults, for instance — write the expected value as a literal in the test with a comment naming the real definition.

Write these five files, mirroring `examples/crud/tests`:

| File | Build tag | Covers |
| --- | --- | --- |
| `<thing>_repository_sql_test.go` | none | Generated SQL: `$`-placeholders (not MySQL `?`), the `ORDER BY` tiebreaker, `LIMIT`/`OFFSET`, filter values arriving as bound arguments |
| `<thing>_repository_test.go` | `integration` | Sentinel translation (`Err<Thing>NotFound` for a missing row and for `DELETE`), partial update skipping nils, `ILIKE` case-insensitivity, paging |
| `<thing>_service_test.go` | `integration` | Bulk-create rollback, pagination defaults, count and page agreeing |
| `<thing>_route_test.go` | `integration` | Status codes (201/200/400/404), validation rejections, the `{timestamp, status, items, error}` envelope |
| `main_test.go` | `integration` | `TestMain` and the local `marker(t)` wrapper |

The untagged SQL file is the one that runs in CI on every commit, so a plain `go test ./...` must stay green with no container. Everything else needs Postgres: `go test -tags=integration ./...`.

Rules that make the suite worth having:

- **Assert against the function production calls.** If the SQL is built inline inside a query function, extract and export a `Build<Thing>Query(...)` so the test cannot rebuild the string itself — a test that reconstructs the query only proves it agrees with itself, and keeps passing after the real one breaks.
- **Isolate by marker, never by truncating the table.** The marker helper gives a random prefix; put it in a text column, filter every query on it, delete by it in `t.Cleanup`. The prefix must contain no `_` or `%` — both are `LIKE` wildcards.
- **`examples/crud/testsupport` is not reusable as-is.** `Main(m)` is generic — copy or import it. `Marker(t)` is not: its cleanup is hardcoded to ``DELETE FROM users WHERE name LIKE $1``. Give the new feature its own `testsupport` package (or its own marker in `main_test.go`) deleting from *its* table on *its* text column. Never point a new feature's cleanup at `users`.
- `go test` runs in the package directory, so `.env` has to be found by walking up — copy that loop from `testsupport.Main` rather than calling bare `godotenv.Load()`.
- Tx-rollback isolation does not work here: `t.Fatal` is `runtime.Goexit`, which skips deferred rollbacks. Marker + `t.Cleanup` is the reason for the whole approach.
- Test each thing at the layer where it is reachable. A rollback triggered by an over-long value cannot be tested over HTTP, because the `max=n` validate tag rejects it first.

## 6. Verify

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./...
go test -tags=integration ./...   # needs Postgres; check `docker ps`, vars in .env
```

`gofmt -l .` must print nothing. If the user declined tests, run everything except the `-tags=integration` line and stop here.

Then confirm the tests can actually fail: change one thing in the code they cover (an `ILIKE` to `LIKE`, drop the `ORDER BY` tiebreaker), re-run, check the expected test fails, and put it back. A test that passes both ways is not testing anything.

Report what you verified and what you did not. Do not commit unless asked.
