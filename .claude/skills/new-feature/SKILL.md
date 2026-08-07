---
name: new-feature
description: Scaffold a new CRUD feature from a table in migrations/, by imitating examples/crud file for file — same layer structure, same naming, same commenting style, same tests. Use when asked to add a feature, resource, endpoint set or CRUD for a table.
---

# New CRUD feature from a migration

`examples/crud` is a complete, working users CRUD. **Produce the same thing for a different table.** The output should read as a sibling of the example — same files, same names, same density of explanation. Where this document and the example disagree, **the example wins**, except for the four fixes in §6.

## 1. Scope: the example is the ceiling

**If `examples/crud` does not have it, do not build it.** The deliverable is that feature with the table swapped, nothing added:

- **No file** without a counterpart in the example. The §4 table is the complete list — no `errors.go`, `mappers/`, `dto/`, `interfaces.go`.
- **No exported symbol** without a counterpart. Same set, same names, same signatures, renamed for the resource. A helper the example lacks is out of scope even when it would be an improvement — especially then.
- **No extra endpoints**, query parameters, filters, sorts, soft-delete or bulk-update beyond the example's six routes.
- **No new dependency**, no logging/metrics/caching layer, no interface-and-mock indirection.
- **No edits to shared code.** `internal/api/presenters`, `internal/api/validators`, `pkg/db`, `cmd/api/main.go`, `internal/config/*` are read-only. The only file outside the feature that changes is `internal/api/routes/routes.go`, to mount it.

A gap you notice in the example is a **finding, not a work item**: §7 lists the four already agreed; report anything else in your final message and leave the code alone. If the user asks for something the example does not cover, build the example-shaped part and say what you left out.

## 2. Read the reference first

**Open these before writing a line** — they are the specification, this document is commentary:

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

With tests (§3), also `examples/crud/testsupport/testsupport.go` and all five files in `examples/crud/tests/`. Then read the chosen `migrations/*.sql`.

Ignore `examples/crud/presenters/response.go` — dead code, a copy of `internal/api/presenters/response.go` that nothing imports. **The new feature gets no `presenters/` package**; it imports `internal/api/presenters`, as the example's controllers do.

## 3. Ask before building

**Two questions, one `AskUserQuestion` call, before any code.** Neither is derivable from the prompt.

1. **Which migration file?** Run `ls migrations/` and offer each file, plus **"A new migration"**. Ask even when there is only one — it is the confirmation that a lot of code is about to be generated off that table. Say in the question text that a filename can be typed into "Other".

   *If they choose a new migration:* ask for the **filename** first (a second call), then for the columns in prose — name, type, nullability, default. Write the file, show it, get confirmation, then scaffold. Never guess columns.

2. **Generate tests?** Yes (full suite, §8) or no (feature code only). Recommend yes. If no, skip §8 and create no `tests/` folder, but still run §9.

**Question 2 is asked every run.** If the prompt already named the table, ask question 2 alone — a one-question call is still a call. The only way to skip a question is the user having answered *that* question explicitly.

Derive without asking: a real feature goes in `internal/api/` (`examples/` only for another teaching example); singular name (`user`) and plural route path (`/users`) come from the table.

## 4. Files to create

Under `internal/api/`, only `presenters/`, `routes/`, `schemas/` and `validators/` hold Go files. `models/`, `repositories/`, `services/`, `controllers/` may exist as empty leftover directories — an `ls` showing them means nothing. For resource `<thing>`:

| File | Holds | Mirror |
| --- | --- | --- |
| `internal/api/models/<thing>.go` | Struct mirroring the table, one field per column | `models/user.go` |
| `internal/api/schemas/requestbody/<thing>.go` | `Create<Thing>`, `Create<Things>`, `Update<Thing>`, `List<Things>` | `schemas/requestbody/user.go` |
| `internal/api/schemas/responsebody/<thing>.go` | Wire struct + `New<Thing>` / `New<Things>` | `schemas/responsebody/user.go` |
| `internal/api/repositories/<thing>.go` | SQL only, sentinels, `<Thing>Filter` | `repositories/user.go` |
| `internal/api/services/<thing>.go` | Orchestration, transactions, pagination maths | `services/user.go` |
| `internal/api/controllers/<thing>.go` | Bind → validate → service → presenter | `controllers/user.go` |
| `internal/api/routes/<thing>.go` | `Set<Thing>Route(router fiber.Router)` | `routes/user.go` |
| `internal/api/tests/*_test.go` | All tests, one package — **only with tests** (§8) | `tests/` |
| `internal/api/testsupport/testsupport.go` | `TestMain` + marker helper — **only with tests** (§8) | `testsupport/` |

Then mount it in `internal/api/routes/routes.go`, whose `SetRoutes` is empty:

```go
thingRoutes := router.Group("/things")
SetThingRoute(thingRoutes)
```

(`examples/crud` is deliberately unmounted — that is why `SetRoutes` is empty. A real feature is mounted.)

**No `README.md` under `internal/api/`.** The example has one because it exists to be read; a real feature is documented by its doc comments (§5). Write a README only for another example under `examples/`, modelled on `examples/crud/README.md`.

## 5. Match the example's writing style

The comments are what make the example worth copying — this is where generated output usually diverges.

- **Every exported symbol gets a `/* … */` block comment explaining *why*, not *what*.** `repositories.Builder` does not say "a Squirrel builder"; it says Squirrel defaults to MySQL `?` placeholders, that `RunWith` is unused because this project is native pgx, and that it is exported for the tests. Reproduce that register.
- **Roughly one comment line per six of code** — the example runs 37/246 (repository), 20/178 (service), 20/142 (controller). Sparse functional code does not match.
- **Package-level orientation comments** where the example has them: the note above the handlers in `controllers/user.go`, the `db.Q(ctx)`-never-`db.Pool()` note in `repositories/user.go`.
- **Inline comments on the non-obvious line**, in the example's voice — `// Non-nil empty slice: encodes as [] rather than null.`
- **Names**: `<thing>Columns`, `<thing>ColumnList`, `Builder`, `<Thing>Filter`, `Apply<Thing>Filter`, `BuildList<Things>Query`, `scan<thing>`, `Err<Thing>NotFound`, `ErrNoUpdateFields`, `Paged<Things>`, `<thing>ID(c)`.
- **Doc-comment the model with the migration's DDL pasted in**, as `models/user.go` does — it is how a reader checks field order against the table.
- **Copy the register, not the typos.** The comments on `UserFilter.Q` and `ApplyUserFilter` are half-finished generalisations that read as nonsense. Write those two properly: name the column actually searched, and say the pattern is a bound argument, not spliced into the SQL.
- Same endpoints and statuses: `POST /` 201, `POST /bulk` 201, `GET /` 200, `GET /:id` 200, `PATCH /:id` 200, `DELETE /:id` 200.

## 6. Column → Go type

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

- **Follow the column's spelling, even when it is wrong** — `users.surename` is misspelled and the code matches it (see the NOTE in `models/user.go`). Fix it in a migration or not at all.
- **Every text column is in the create body and `required`, defaulted or not.** A `varchar`/`text` column holds a value the client owns, so the API always lets it supply one. A `DEFAULT` on such a column is a fallback for hand-written SQL inserts, not a reason to deny the field over HTTP: `users.name` is `default 'N/A' not null` and `requestbody.CreateUser` requires it anyway. Never leave a text column out of `Create<Thing>` because it is defaulted — the result is a resource the client cannot name at creation.
- **Only server-generated columns stay out of the create body** — the uuid primary key, `created_at`/`updated_at`. Those are values the client has no business choosing; the database supplies them and `RETURNING` reads them back.
- `NOT NULL` without a default → `validate:"required"` on create, same as the defaulted text columns above.
- Name the text column the list filter searches in the `<Thing>Filter` doc comment.

## 7. The four fixes

These four are the only places the example is not the authority. **Everything else — layering, sentinels, `db.Q(ctx)` discipline, the raw-SQL/Squirrel split, pagination maths, the response envelope, and the error handling below — is copied as-is.**

1. **`DeleteUser` ignores `tag.RowsAffected()`.** Its doc comment promises `ErrUserNotFound`; the body never produces it. `DELETE` raises no `pgx.ErrNoRows`, so read absence off the command tag and return the sentinel.
2. **`services.UpdateUser` swallows the transaction error** — it captures `err` from `db.ExecTx`, then `return nil`. Return it.
3. **`services.ListUsers` reads count and page outside a transaction**, though its comment claims otherwise. Wrap both in one `db.ExecTx` sharing one filter value.
4. **`services.UpdateUser` returns only `error`** while its doc comment promises "the stored row", and the controller answers the literal `"SUCCESS"`. Return `responsebody.<Thing>` and pass it to `presenters.ResponseSuccess`, as `CreateUser` does — `UPDATE … RETURNING` already fetched it. This also drops the `GetUserByID` pre-read, which exists only to re-fetch an id the caller supplied.

### Error handling — copy it exactly

Controllers do what `controllers/user.go` does and no more. **The controller never inspects which error it got:**

- Bind/validate failure → `fiber.NewError(fiber.StatusBadRequest, err.Error())`
- Malformed `:id` → 400, inside `<thing>ID(c)`
- **Every service error → `fiber.NewError(fiber.StatusInternalServerError, err.Error())`**, at every call site

So a missing row answers **500, not 404** — `tests/user_route_test.go` pins that in `TestHTTP_MissingUserIs500`; port it rather than asserting a 404. An empty PATCH answers 400 only because the fields are `required`. Keep `Update<Thing>`'s fields non-pointer and `required` as the example has them: PATCH means "replace", not "merge". The repository still takes `*string` per column and skips nils — available for later, unexercised now.

`Err<Thing>NotFound` and `ErrNoUpdateFields` are still defined in the repository and re-exported by the service, with the example's doc comments. The feature just does not act on them yet.

## 8. Tests

**Skip if the user answered "no tests" in §3.** Otherwise all five files, both tags — a single happy-path test is not this. Mirror `examples/crud/tests`: one package in `tests/`, one file per layer.

| File | Build tag | Covers |
| --- | --- | --- |
| `<thing>_repository_sql_test.go` | none | Generated SQL: `$`-placeholders (not MySQL `?`), `ORDER BY` tiebreaker, `LIMIT`/`OFFSET`, filter values as bound arguments |
| `<thing>_repository_test.go` | `integration` | Sentinels (`Err<Thing>NotFound` for a missing row **and for `DELETE`**), partial update skipping nils, `ILIKE` case-insensitivity, paging |
| `<thing>_service_test.go` | `integration` | Bulk-create rollback, pagination defaults, count and page agreeing |
| `<thing>_route_test.go` | `integration` | Status codes (201/200/400, **500 for a missing row** — §7), validation rejections, the `{timestamp, status, items, error}` envelope |
| `main_test.go` | `integration` | `TestMain` and the local `marker(t)` wrapper |

The untagged file runs in CI on every commit, so `go test ./...` must stay green with no container.

- Tests sit outside the packages they test, so **anything they touch must be exported**, with a doc comment saying that is why (as `BuildListUsersQuery` and `ApplyUserFilter` do). Where a value cannot be exported (pagination defaults), write the literal with a comment naming the real definition.
- **Assert against the function production calls.** A test that rebuilds the SQL itself only proves it agrees with itself.
- **Isolate by marker, never by truncating.** Random prefix in a text column, filtered in every query, deleted in `t.Cleanup`; no `_` or `%` in it — both are `LIKE` wildcards. Tx-rollback isolation fails here because `t.Fatal` is `runtime.Goexit`, which skips deferred rollbacks.
- **`testsupport` is only half reusable.** `Main(m)` is generic — copy or import it. `Marker(t)` hardcodes `DELETE FROM users WHERE name LIKE $1`; give the feature its own, on *its* table and column. Never point a new cleanup at `users`.
- `.env` is found by walking up, because `go test` runs in the package directory — copy that loop rather than calling bare `godotenv.Load()`.
- `newTestApp()` copies the `ErrorHandler` from `cmd/api/main.go` plus `validators.Init()` and this feature's routes. Copy its comment too, including the known-drift note.
- Test each thing where it is reachable: a rollback from an over-long value cannot be tested over HTTP, because `max=n` rejects it first.

## 9. Verify

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./...
go test -tags=integration ./...   # needs Postgres; check `docker ps`, vars in .env
```

`gofmt -l .` must print nothing. Without tests, run everything but the tagged line.

**Confirm the tests can fail:** flip one thing they cover (`ILIKE`→`LIKE`, drop the `ORDER BY` tiebreaker), check the right test fails, put it back.

**Check the style:** read each generated file beside its mirror. A file noticeably shorter than its counterpart is missing the explanations.

**Check the scope (§1)** — this catches what a green build cannot:

```bash
git status --short   # new files: each needs a counterpart in examples/crud
git diff --stat      # modified files: only routes.go should appear outside the feature
grep -n "^func \|^type \|^var " internal/api/*/<thing>.go   # exported symbols: same test
```

Anything without a counterpart is out of scope — delete it, or justify it in your final message.

Report what you verified and what you did not, and list separately any example gap you noticed and left alone (§1). Do not commit unless asked.
