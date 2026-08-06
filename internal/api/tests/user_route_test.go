//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Binh-2060/go-application-template/internal/api/routes"
	"github.com/Binh-2060/go-application-template/internal/api/schemas/responsebody"
	"github.com/Binh-2060/go-application-template/internal/api/validators"
	"github.com/gofiber/fiber/v3"
)

/*
End-to-end tests through the Fiber app.

	go test -tags=integration ./internal/api/...

These cover what the repository tests cannot see: that binding and validation
actually run, that a sentinel becomes the right status code, and that every
response carries the project's envelope. They drive the app with app.Test, so
nothing listens on a port and no test needs a free one.
*/

/*
Build an app with just what these tests exercise: the error handler, the
validators, and this feature's routes.

The ErrorHandler is copied from cmd/api/main.go because main.go builds its app
inline, so there is nothing importable to reuse. That duplication is the reason
these tests can drift from production — extracting a newApp() in main.go and
calling it here would fix it.

CORS, compression and helmet are left out deliberately: they are global concerns
tested where they are configured, and including them here would only obscure
which layer produced a given response.
*/
func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			return ctx.Status(code).JSON(fiber.Map{
				"timestamp": time.Now().Format("2006-01-02-15-04-05"),
				"status":    0,
				"items":     nil,
				"error":     err.Error(),
			})
		},
	})

	validators.Init()
	routes.SetUserRoute(app.Group("/users"))

	return app
}

// The response shape every endpoint must produce, success or failure.
type envelope struct {
	Timestamp string          `json:"timestamp"`
	Status    int             `json:"status"`
	Items     json.RawMessage `json:"items"`
	Error     *string         `json:"error"`
}

// The extra nesting ResponseSuccessListData adds inside items.
type listItems struct {
	ListData   json.RawMessage `json:"list_data"`
	Pagination struct {
		CurrentPage          int `json:"current_page"`
		CurrentPageTotalItem int `json:"current_page_total_item"`
		TotalPage            int `json:"total_page"`
	} `json:"pagination"`
}

/*
Issue a request and decode the envelope. body may be nil for GET and DELETE.
*/
func do(t *testing.T, app *fiber.App, method, target string, body any) (int, envelope) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(t.Context(), method, target, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("%s %s returned non-envelope body %q: %v", method, target, raw, err)
	}

	return res.StatusCode, env
}

func decodeUser(t *testing.T, items json.RawMessage) responsebody.User {
	t.Helper()

	var user responsebody.User
	if err := json.Unmarshal(items, &user); err != nil {
		t.Fatalf("decode user from %q: %v", items, err)
	}

	return user
}

/*
A created resource answers 201 and is immediately readable at its own id — the
one path every other test depends on.
*/
func TestHTTP_CreateThenGet(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	status, env := do(t, app, http.MethodPost, "/users", map[string]string{
		"name": m + "Ada", "surename": "Lovelace",
	})
	if status != http.StatusCreated {
		t.Fatalf("POST /users = %d, want 201 (%v)", status, env.Error)
	}
	if env.Status != 1 {
		t.Errorf("envelope status = %d, want 1", env.Status)
	}
	if env.Error != nil {
		t.Errorf("envelope error = %v, want null", *env.Error)
	}

	created := decodeUser(t, env.Items)
	if created.ID == "" {
		t.Fatal("created user has no id")
	}

	status, env = do(t, app, http.MethodGet, "/users/"+created.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("GET /users/:id = %d, want 200", status)
	}

	got := decodeUser(t, env.Items)
	if got.ID != created.ID || got.Name != m+"Ada" || got.Surename != "Lovelace" {
		t.Errorf("fetched %+v, want the row just created", got)
	}
}

/*
A malformed id is bad input, not a missing resource: 400, and no query is sent.

The distinction matters — answering 404 here tells a client the id might exist
somewhere, when it could never be valid.
*/
func TestHTTP_MalformedIDIs400(t *testing.T) {
	app := newTestApp()

	status, env := do(t, app, http.MethodGet, "/users/not-a-uuid", nil)

	if status != http.StatusBadRequest {
		t.Fatalf("GET /users/not-a-uuid = %d, want 400", status)
	}
	if env.Status != 0 {
		t.Errorf("envelope status = %d, want 0", env.Status)
	}
	if env.Error == nil {
		t.Error("envelope error is null on a failed request")
	}
}

/*
controllers/user.go has no sentinel-to-status mapping: every service error,
ErrUserNotFound included, becomes a 500. A true 404 here would need a
toHTTPError switch translating the sentinel before it reaches fiber.NewError.
*/
func TestHTTP_MissingUserIs500(t *testing.T) {
	app := newTestApp()

	status, _ := do(t, app, http.MethodGet, "/users/00000000-0000-0000-0000-000000000000", nil)

	if status != http.StatusInternalServerError {
		t.Fatalf("GET missing user = %d, want 500", status)
	}
}

/*
Validation tags are enforced only when input goes through the validators
package — Fiber's own StructValidator hook is not configured on this app. This
fails if a handler is ever rewritten to call c.Bind().Body directly.
*/
func TestHTTP_ValidationRejectsBadBodies(t *testing.T) {
	app := newTestApp()

	cases := map[string]map[string]any{
		"missing surename": {"name": "Ada"},
		"empty name":       {"name": "", "surename": "Lovelace"},
		"name over 200":    {"name": string(bytes.Repeat([]byte("a"), 201)), "surename": "Lovelace"},
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			status, _ := do(t, app, http.MethodPost, "/users", body)
			if status != http.StatusBadRequest {
				t.Errorf("POST /users = %d, want 400", status)
			}
		})
	}
}

/*
PATCH means "replace", not "merge" — both fields are required, so an empty body
cannot be turned into an UPDATE. This is the client's mistake: 400, not 500.
*/
func TestHTTP_PatchWithNoFieldsIs400(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	_, env := do(t, app, http.MethodPost, "/users", map[string]string{
		"name": m + "Ada", "surename": "Lovelace",
	})
	id := decodeUser(t, env.Items).ID

	status, _ := do(t, app, http.MethodPatch, "/users/"+id, map[string]any{})

	if status != http.StatusBadRequest {
		t.Fatalf("PATCH with empty body = %d, want 400", status)
	}
}

/*
PATCH with only one field is the same 400 as an empty body: both fields are
required, so sending just "name" cannot be turned into an UPDATE either.
*/
func TestHTTP_PatchWithOneFieldIs400(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	_, env := do(t, app, http.MethodPost, "/users", map[string]string{
		"name": m + "Ada", "surename": "Lovelace",
	})
	id := decodeUser(t, env.Items).ID

	status, _ := do(t, app, http.MethodPatch, "/users/"+id, map[string]string{"name": m + "Grace"})

	if status != http.StatusBadRequest {
		t.Fatalf("PATCH with only name = %d, want 400", status)
	}
}

/*
The full-replace guarantee, seen from outside: send both fields, PATCH answers
200. The controller reports the literal "SUCCESS" rather than the updated row
(see the other endpoints' envelope), so the fetch-back is what actually proves
the write landed.
*/
func TestHTTP_PatchReplacesBothFields(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	_, env := do(t, app, http.MethodPost, "/users", map[string]string{
		"name": m + "Ada", "surename": "Lovelace",
	})
	id := decodeUser(t, env.Items).ID

	status, _ := do(t, app, http.MethodPatch, "/users/"+id, map[string]string{
		"name": m + "Grace", "surename": "Hopper",
	})
	if status != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200", status)
	}

	_, env = do(t, app, http.MethodGet, "/users/"+id, nil)
	updated := decodeUser(t, env.Items)
	if updated.Name != m+"Grace" {
		t.Errorf("Name = %q, want %q", updated.Name, m+"Grace")
	}
	if updated.Surename != "Hopper" {
		t.Errorf("Surename = %q, want %q", updated.Surename, "Hopper")
	}
}

/*
The literal /bulk route is registered before /:id. If that order is ever
reversed this request is routed to GetUser with id="bulk" and fails on the uuid
check, so the 400 this test would see is the symptom to recognise.
*/
func TestHTTP_BulkCreatesEveryRow(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	status, env := do(t, app, http.MethodPost, "/users/bulk", map[string]any{
		"users": []map[string]string{
			{"name": m + "Ada", "surename": "Lovelace"},
			{"name": m + "Grace", "surename": "Hopper"},
		},
	})
	if status != http.StatusCreated {
		t.Fatalf("POST /users/bulk = %d, want 201 (%v)", status, env.Error)
	}

	var created []responsebody.User
	if err := json.Unmarshal(env.Items, &created); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("created %d users, want 2", len(created))
	}
}

/*
The list endpoint, including the pagination block ResponseSuccessListData adds.

Filtering by the marker is what makes the counts assertable at all — the table
is shared, so an unfiltered total is whatever else happens to be there.
*/
func TestHTTP_ListFilterAndPagination(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	for _, n := range []string{"A", "B", "C"} {
		do(t, app, http.MethodPost, "/users", map[string]string{"name": m + n, "surename": "Surname"})
	}

	status, env := do(t, app, http.MethodGet, "/users?q="+m+"&page=1&per_page=2", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /users = %d, want 200", status)
	}

	var items listItems
	if err := json.Unmarshal(env.Items, &items); err != nil {
		t.Fatalf("decode list items: %v", err)
	}

	var users []responsebody.User
	if err := json.Unmarshal(items.ListData, &users); err != nil {
		t.Fatalf("decode list_data: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("page has %d rows, want 2", len(users))
	}
	if items.Pagination.CurrentPage != 1 {
		t.Errorf("current_page = %d, want 1", items.Pagination.CurrentPage)
	}
	if items.Pagination.CurrentPageTotalItem != 2 {
		t.Errorf("current_page_total_item = %d, want 2", items.Pagination.CurrentPageTotalItem)
	}
	// Ceiling of 3/2 — the assertion that catches integer division truncating.
	if items.Pagination.TotalPage != 2 {
		t.Errorf("total_page = %d, want 2", items.Pagination.TotalPage)
	}
}

/*
No matches must serialise as [] rather than null, so a client can iterate the
result without a nil check. This is asserted on the raw JSON because both decode
into an empty Go slice and the distinction would be lost.
*/
func TestHTTP_ListWithNoMatchesIsEmptyArray(t *testing.T) {
	app := newTestApp()

	_, env := do(t, app, http.MethodGet, "/users?q="+marker(t), nil)

	var items listItems
	if err := json.Unmarshal(env.Items, &items); err != nil {
		t.Fatalf("decode list items: %v", err)
	}

	if got := string(items.ListData); got != "[]" {
		t.Errorf("list_data = %s, want []", got)
	}
}

/*
DeleteUser is idempotent (no rows-affected check) and every service error maps
to 500 (no toHTTPError), so the missing-row cases below read 500 / 200 rather
than the 404s a fuller implementation would give.
*/
func TestHTTP_Delete(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	_, env := do(t, app, http.MethodPost, "/users", map[string]string{
		"name": m + "Ada", "surename": "Lovelace",
	})
	id := decodeUser(t, env.Items).ID

	if status, _ := do(t, app, http.MethodDelete, "/users/"+id, nil); status != http.StatusOK {
		t.Fatalf("DELETE = %d, want 200", status)
	}

	if status, _ := do(t, app, http.MethodGet, "/users/"+id, nil); status != http.StatusInternalServerError {
		t.Fatalf("GET after delete = %d, want 500", status)
	}

	// DELETE has no rows-affected check, so deleting an already-deleted id is
	// still a no-op success, not a 404.
	if status, _ := do(t, app, http.MethodDelete, "/users/"+id, nil); status != http.StatusOK {
		t.Fatalf("second DELETE = %d, want 200", status)
	}
}
