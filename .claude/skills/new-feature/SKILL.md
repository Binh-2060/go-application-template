---
name: new-feature
description: Scaffold a new CRUD feature from a table in migrations/, by imitating examples/crud file for file — same layer structure, same naming, same commenting style, same tests. Use when asked to add a feature, resource, endpoint set or CRUD for a table.
---

# New CRUD feature from a migration

`examples/crud` is a complete, working users CRUD. **This skill's job is to produce the same thing for a different table.** The output should be recognisable as a sibling of `examples/crud` — same files, same order, same naming scheme, same density of explanation — not a generic Go CRUD that happens to satisfy a list of rules.

This document is deliberately thin on rules that the example already demonstrates. Where the two disagree, **the example wins**, except for the short list in §6.

## 1. Read the reference first

**Read these files before writing a single line.** Not skimmed, not inferred from this document — opened. They are the specification; everything below is only commentary.

```
examples/crud/README.md
examples/crud/models/user.go
examples/crud/schemas/requestbody/user.go
examples/crud/schemas/responsebody/user.go
examples/crud/repositories/user.go
examples/crud/services/user.go
examples/crud/controllers/user.go
examples/crud/routes/user.go
```

And, if the user wants tests (§2), also:

```
examples/crud/testsupport/testsupport.go
examples/crud/tests/main_test.go
examples/crud/tests/user_repository_sql_test.go
examples/crud/tests/user_repository_test.go
examples/crud/tests/user_service_test.go
examples/crud/tests/user_route_test.go
```

Then read the chosen `migrations/*.sql`.

Ignore `examples/crud/presenters/response.go`. It is dead code — a byte-for-byte copy of `internal/api/presenters/response.go` that nothing imports. **Do not give the new feature a `presenters/` package**; import `internal/api/presenters` the way `examples/crud/controllers/user.go` does.

## 2. Establish inputs

**Ask the user these two questions, with `AskUserQuestion`, before writing any code.** Both change what gets built, and neither is derivable from the prompt. Ask them in one call.

1. **Which migration file?** Run `ls migrations/` and offer each file as an option, plus a final **"A new migration"** option. **Ask even when there is only one file** — a one-option question is still the confirmation that the user meant that table, and this scaffolds a lot of code off it. The options are a convenience, not a restriction: `AskUserQuestion` always carries a free-text "Other" choice, so say in the question text that the user may type a filename directly. If the user's prompt named a table with no matching `migrations/*.sql`, still ask — with that name offered as the new-migration option.

   **If they pick "A new migration" (or name a file that does not exist):** ask a second `AskUserQuestion` for the **filename** before anything else — offer `migrations/<table>.sql` spellings derived from any resource name already mentioned, and rely on "Other" for a name you could not guess. Only once the filename is fixed, ask for the columns in prose (name, Postgres type, nullability, default — not enumerable as options). Write the file, show it, and get confirmation before scaffolding. The schema drives everything below, so never guess columns.

2. **Generate tests?** Options: yes (full suite per §7) / no (feature code only). Default recommendation is yes. If no, skip §7 entirely and do not create a `tests/` folder — but still run the build/vet checks in §8.

**Question 2 is asked every single run, without exception.** Nothing in a feature request implies a tests answer, so there is nothing to derive it from. If the user's prompt already named the table, drop question 1 and ask question 2 alone — a one-question call is still a call. The only way to skip a question is for the user to have already answered *that* question explicitly.

Do not skip the call because the answer looks obvious, because only one migration exists, or because you are confident in a default. Ask, wait for the answer, then build.

Derive the rest without asking:

- **Where it goes.** A real feature belongs in `internal/api/`. Only put it under `examples/` if it is another teaching example.
- **Singular resource name** (`user`) and **plural route path** (`/users`) — from the table name.

## 3. Files to create

`internal/api/` currently contains only `presenters/`, `routes/`, `schemas/`, `validators/`. **`models/`, `repositories/`, `services/` and `controllers/` do not exist yet — create them.** There is no sibling code in those directories to copy tone from, which is exactly why §1 is not optional.

For resource `<thing>`:

| File | Holds | Mirror |
| --- | --- | --- |
| `internal/api/models/<thing>.go` | Struct mirroring the table, one field per column | `examples/crud/models/user.go` |
| `internal/api/schemas/requestbody/<thing>.go` | `Create<Thing>`, `Create<Things>`, `Update<Thing>`, `List<Things>` | `examples/crud/schemas/requestbody/user.go` |
| `internal/api/schemas/responsebody/<thing>.go` | Wire struct + `New<Thing>` / `New<Things>` mappers | `examples/crud/schemas/responsebody/user.go` |
| `internal/api/repositories/<thing>.go` | SQL only, sentinels, `<Thing>Filter` | `examples/crud/repositories/user.go` |
| `internal/api/services/<thing>.go` | Orchestration, transactions, pagination maths | `examples/crud/services/user.go` |
| `internal/api/controllers/<thing>.go` | Bind → validate → service → presenter | `examples/crud/controllers/user.go` |
| `internal/api/routes/<thing>.go` | `Set<Thing>Route(router fiber.Router)` | `examples/crud/routes/user.go` |
| `internal/api/tests/*_test.go` | All tests, one package — **only if the user asked for tests** (§7) | `examples/crud/tests/` |
| `internal/api/testsupport/testsupport.go` | `TestMain` + marker helper for this feature — only with tests (§7) | `examples/crud/testsupport/` |

Then register in `internal/api/routes/routes.go`, which today has an empty `SetRoutes`:

```go
thingRoutes := router.Group("/things")
SetThingRoute(thingRoutes)
```

Mounting is part of the job for a real feature. (`examples/crud` is deliberately left unmounted; that is the exception, and the reason `SetRoutes` is empty.)

**Write a `README.md` next to the feature**, modelled on `examples/crud/README.md`: the table as SQL, an endpoint table (method / path / body / success status), what each layer is doing, and the raw-SQL-vs-Squirrel split as it applies to this resource. The example's README is the most visible thing it produces; a feature without one does not match.

## 4. Match the example's writing style

This is where generated output usually diverges, and it is not cosmetic — the comments are what make the example a teaching artifact.

- **Every exported symbol gets a `/* … */` block comment, and it explains *why*, not *what*.** Compare: `repositories.Builder` does not say "a Squirrel builder"; it says Squirrel defaults to MySQL `?` placeholders, that `RunWith` is unused because this project is native pgx, and that it is exported only for the tests. Reproduce that register.
- **Roughly one comment line per six lines of code.** `examples/crud/repositories/user.go` is 37 comment lines in 246; `services/user.go` is 20 in 178; `controllers/user.go` is 20 in 142. Sparse, purely functional code does not match.
- **Package-level orientation comments** where the example has them — the free-floating note above the handlers in `controllers/user.go`, the `db.Q(ctx)`-never-`db.Pool()` note above the query functions in `repositories/user.go`.
- **Inline comments on the non-obvious line**, in the example's voice: `// Non-nil empty slice: encodes as [] rather than null.`, `// rows.Err reports failures that happened mid-stream…`.
- **Name things the way the example does**: `<thing>Columns`, `<thing>ColumnList`, `Builder`, `<Thing>Filter`, `Apply<Thing>Filter`, `Build List<Things>Query`, `scan<thing>`, `Err<Thing>NotFound`, `ErrNoUpdateFields`, `Paged<Things>`, `<thing>ID(c)`.
- **Doc-comment the model with the migration's DDL pasted in**, as `models/user.go` does. It is how a reader checks field order against the table.
- Keep the endpoint set and their statuses identical in shape: `POST /` 201, `POST /bulk` 201, `GET /` 200, `GET /:id` 200, `PATCH /:id` 200, `DELETE /:id` 200.

## 5. Column → Go type

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

- **Follow the column's spelling, even when it is wrong.** `users.surename` is misspelled and the code matches it — see the NOTE in `models/user.go`. Diverging silently makes the mapping a lie; fix it in a migration if it matters.
- Columns with a `DEFAULT` (`id`, `created_at`) are **not** in the create request body — the database supplies them and `RETURNING` reads them back. Note that a column can be `NOT NULL` *and* defaulted (`products.name` is `default 'N/A' not null`); the default is what decides, not the nullability.
- `NOT NULL` without a default → `validate:"required"` on create.
- Pick the text column the list filter searches (`users` filters on `name`) and say so in the `<Thing>Filter` doc comment.

## 6. Where the example is deliberately incomplete

`examples/crud` is a teaching example with four known gaps, three of which its own comments or tests call out. **A real feature must not copy these.** This is the only place the example is not the authority.

1. **No `toHTTPError`.** Every controller error in the example becomes a 500 via `fiber.NewError(fiber.StatusInternalServerError, …)`, so a missing row answers 500 instead of 404 — `tests/user_route_test.go` documents this as a shortcoming. The new feature gets a small `toHTTPError` switch mapping `Err<Thing>NotFound` → 404 and `ErrNoUpdateFields` → 400, **returning anything unrecognised untouched** so `main.go`'s `ErrorHandler` renders it. Assert the 404s in the route tests.
2. **`DeleteUser` ignores `tag.RowsAffected()`.** Its doc comment promises `ErrUserNotFound`; the body never produces it. `DELETE` does not raise `pgx.ErrNoRows`, so read absence off the command tag and return the sentinel.
3. **`services.UpdateUser` swallows the transaction error** — it captures `err` from `db.ExecTx` and then `return nil`. Return it.
4. **`services.ListUsers` reads its count and its page outside a transaction**, though its comment claims otherwise. Wrap both in one `db.ExecTx` sharing one filter value, so the total always describes the rows returned.

Everything else in the example — the layering, the sentinels, the `db.Q(ctx)` discipline, the raw-SQL/Squirrel split, the pagination maths, the response envelope — is correct and should be reproduced as-is.

## 7. Tests

**Skip this whole section if the user answered "no tests" in §2.** Otherwise write the full suite below — all five files, both tags. A single happy-path test is not what was asked for.

Mirror `examples/crud/tests` exactly: one package in a `tests/` folder next to the feature's other packages, `main_test.go` holding `TestMain` and the `marker(t)` wrapper, one file per layer.

| File | Build tag | Covers |
| --- | --- | --- |
| `<thing>_repository_sql_test.go` | none | Generated SQL: `$`-placeholders (not MySQL `?`), the `ORDER BY` tiebreaker, `LIMIT`/`OFFSET`, filter values arriving as bound arguments |
| `<thing>_repository_test.go` | `integration` | Sentinel translation (`Err<Thing>NotFound` for a missing row **and for `DELETE`**), partial update skipping nils, `ILIKE` case-insensitivity, paging |
| `<thing>_service_test.go` | `integration` | Bulk-create rollback, pagination defaults, count and page agreeing |
| `<thing>_route_test.go` | `integration` | Status codes (201/200/400/**404**), validation rejections, the `{timestamp, status, items, error}` envelope |
| `main_test.go` | `integration` | `TestMain` and the local `marker(t)` wrapper |

The untagged SQL file is the one that runs in CI on every commit, so a plain `go test ./...` must stay green with no container. Everything else needs Postgres: `go test -tags=integration ./...`.

Rules that make the suite worth having:

- The tests sit outside the packages they test, so **anything they touch must be exported** — and the doc comment should say it is exported for that reason, as `BuildListUsersQuery` and `ApplyUserFilter` do. Where a value cannot reasonably be exported (the pagination defaults), write the expected value as a literal with a comment naming the real definition.
- **Assert against the function production calls.** If the SQL is built inline inside a query function, extract and export a `Build<Thing>Query(...)`. A test that reconstructs the query only proves it agrees with itself, and keeps passing after the real one breaks.
- **Isolate by marker, never by truncating the table.** The marker helper gives a random prefix; put it in a text column, filter every query on it, delete by it in `t.Cleanup`. The prefix must contain no `_` or `%` — both are `LIKE` wildcards.
- **`examples/crud/testsupport` is not reusable as-is.** `Main(m)` is generic — copy or import it. `Marker(t)` is not: its cleanup is hardcoded to ``DELETE FROM users WHERE name LIKE $1``. Give the new feature its own `testsupport` package deleting from *its* table on *its* text column. Never point a new feature's cleanup at `users`.
- `go test` runs in the package directory, so `.env` has to be found by walking up — copy that loop from `testsupport.Main` rather than calling bare `godotenv.Load()`.
- Tx-rollback isolation does not work here: `t.Fatal` is `runtime.Goexit`, which skips deferred rollbacks. Marker + `t.Cleanup` is the reason for the whole approach.
- `newTestApp()` in the route test builds a Fiber app with the `ErrorHandler` copied from `cmd/api/main.go`, plus `validators.Init()` and this feature's routes only. Copy that helper and its comment — including the note that the duplication is a known drift risk.
- Test each thing at the layer where it is reachable. A rollback triggered by an over-long value cannot be tested over HTTP, because the `max=n` validate tag rejects it first.

## 8. Verify

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./...
go test -tags=integration ./...   # needs Postgres; check `docker ps`, vars in .env
```

`gofmt -l .` must print nothing. If the user declined tests, run everything except the `-tags=integration` line and stop here.

Then confirm the tests can actually fail: change one thing in the code they cover (an `ILIKE` to `LIKE`, drop the `ORDER BY` tiebreaker), re-run, check the expected test fails, and put it back. A test that passes both ways is not testing anything.

Finally, re-read the generated files against §4 side by side with their `examples/crud` counterparts. If a file is noticeably shorter than its mirror, the missing lines are the explanations, and it does not match.

Report what you verified and what you did not. Do not commit unless asked.
