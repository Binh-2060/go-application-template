---
name: new-api
description: Scaffold a new API feature skeleton into internal/api/ by imitating examples/api file for file — one route group, one handler returning the response envelope, and empty placeholder packages for the layers not written yet. No database, no migration, no tests. Use when asked for a sample/skeleton/starter API, a new endpoint, or a feature stub by name. For a full CRUD off a table in migrations/, use new-feature instead.
---

# New API skeleton from a feature name

`examples/api` is the smallest thing this template calls a feature: a mounted route group, one handler, and the empty packages the rest of the layers will go in. **Produce the same thing under a different name.** Where this document and the example disagree, **the example wins**, except for the one deviation in §5.

**This is the non-CRUD path, and it is meant to be fast.** The user is scaffolding so they can start writing the actual logic themselves. Seven small files, a mount, a build — done in one pass.

- **Do not ask questions** when the prompt names the feature (§4 is the only exception).
- **Do not plan, propose, or summarise before writing.** Read §2, write the files, verify, report.
- **Do not fill anything in.** Guessed business logic is worse than the empty file it replaced — the user is about to write the real thing and would have to read yours first to delete it.

**Wrong skill?** If the request names a table in `migrations/` or asks for create/list/update/delete, it wants `new-feature` (full CRUD from `examples/crud`) — say so and stop. This skill writes no SQL.

## 1. Scope: the example is the ceiling

**If `examples/api` does not have it, do not build it.** The deliverable is that skeleton renamed:

- **No file** without a counterpart in the example. The §3 table is the complete list.
- **No exported symbol** without a counterpart — one handler and one `Set<Thing>Route`, nothing else.
- **No bodies in the placeholder packages.** `services/`, `repositories/`, `schemas/requestbody/`, `schemas/responsebody/` get a `package` line and nothing else, exactly as the example has them. Empty is the point: the skeleton shows where the code goes, it does not guess at it.
- **No database, no migration, no `pkg/db` import, no validators, no request struct, no tests.** The handler takes no input.
- **No extra routes or methods** beyond the example's single `GET /`.
- **No new dependency**, no logging/metrics/middleware layer.
- **No edits to shared code.** `internal/api/presenters`, `internal/api/validators`, `pkg/db`, `cmd/api/main.go`, `internal/config/*` are read-only. The only file outside the feature that changes is `internal/api/routes/routes.go`, to mount it.
- **No `README.md` under `internal/api/`.** Write one only for another example under `examples/`.

A gap you notice in the example is a **finding, not a work item**: report it in your final message and leave the code alone. If the user asks for something the example does not cover, build the example-shaped part and say what you left out.

## 2. Read the reference first

**Open all of these before writing a line** — it is eight small files and they are the specification:

```
examples/api/routes/routes.go
examples/api/routes/sameple.go
examples/api/controllers/sample.go
examples/api/services/sample.go
examples/api/repositories/sample.go
examples/api/schemas/requestbody/sample.go
examples/api/schemas/responsebody/sample.go
```

Ignore `examples/api/presenters/response.go` — dead code, a byte-for-byte copy of `internal/api/presenters/response.go` that nothing imports. **The new feature gets no `presenters/` package**; it imports `internal/api/presenters`, as `examples/crud/controllers/user.go` does.

## 3. Files to create

Under `internal/api/`, only `presenters/`, `routes/`, `schemas/` and `validators/` hold Go files today. `controllers/`, `services/`, `repositories/`, `models/` may not exist at all — create the directories you need. For feature `<thing>` (singular, lower-case):

| File | Holds | Mirror |
| --- | --- | --- |
| `internal/api/controllers/<thing>.go` | `Get<Thing>Controller(c fiber.Ctx) error` | `controllers/sample.go` |
| `internal/api/routes/<thing>.go` | `Set<Thing>Route(router fiber.Router)` | `routes/sameple.go` |
| `internal/api/services/<thing>.go` | `package services` only | `services/sample.go` |
| `internal/api/repositories/<thing>.go` | `package repositories` only | `repositories/sample.go` |
| `internal/api/schemas/requestbody/<thing>.go` | `package requestbody` only | `schemas/requestbody/sample.go` |
| `internal/api/schemas/responsebody/<thing>.go` | `package responsebody` only | `schemas/responsebody/sample.go` |

The handler is the example's, renamed:

```go
func GetThingController(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess("SUCCESS"))
}
```

Then mount it in `internal/api/routes/routes.go`, whose `SetRoutes` is empty — this mirrors `examples/api/routes/routes.go`, which is the only place the example shows the wiring:

```go
// things route
thingRoutes := router.Group("/things")
SetThingRoute(thingRoutes)
```

`SetRoutes` is already called from `cmd/api/main.go` under `/api/{API_VERSION}`, so the live path is `/api/v1/things`. Nothing in `main.go` changes.

## 4. Naming and the route path

- **Feature name** — take it from the prompt. Ask (one `AskUserQuestion`) only if the prompt does not name one; do not invent a name.
- Singular, lower-case for **files and identifiers**: `<thing>.go`, `Get<Thing>Controller`, `Set<Thing>Route`.
- Plural, lower-case for the **route path**: `/things`. The example's `/sameple` is singular only because it is a typo (§5).
- A multi-word name is one word in the path (`/orderitems`) and Pascal case in identifiers (`GetOrderItemController`), matching how Go names read in the rest of this repo.

## 5. The one deviation: do not copy `sameple`

`examples/api` misspells "sample" as `sameple` in a filename, a function name and a route path. **This is the only place the example is not the authority** — spell the feature name correctly everywhere.

Contrast `examples/crud`, which deliberately keeps `users.surename` misspelled because the *column* is misspelled and the code must match the database. There is no database here, so nothing forces the typo.

## 6. Comments

`examples/api` is nearly comment-free — one `//sameple route` line. Do not import `examples/crud`'s comment density into a skeleton with no logic to explain; a comment on a `package` line is noise. Write:

- A short `/* … */` doc comment on the handler saying its method and path, as `examples/crud/controllers/user.go` does (`/* GET /things — … */`).
- The `// things route` line above the group in `routes.go`, as the example has it.

Nothing else. The placeholder files stay bare.

## 7. Verify

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./...
```

`gofmt -l .` must print nothing. There are no tests to add and no integration run — this feature touches no database.

Then confirm it actually serves. With the app running (`go run ./cmd/api`, needs the vars in `.env`):

```bash
curl -s localhost:${PORT}/api/${API_VERSION}/things
```

Expect `{"timestamp":…,"status":1,"items":"SUCCESS","error":null}`. A 404 means the group was never mounted in `routes.go`.

**Check the scope (§1):**

```bash
git status --short   # new files: each needs a counterpart in examples/api
git diff --stat      # modified files: only routes.go outside the feature
```

## 8. Hand it back

Keep the closing message short — the user is going straight into the code. Give them:

- the **live path** (`/api/{API_VERSION}/things`) and what `curl` returned, or plainly that you could not start the server;
- the **file to open first**, `internal/api/controllers/<thing>.go`, and the two next to it that are waiting (`services/<thing>.go`, `repositories/<thing>.go`);
- one line: `new-feature` is the skill if this later turns into CRUD over a table.

No summary of the files you wrote — `git status` already says that. Do not commit unless asked.
