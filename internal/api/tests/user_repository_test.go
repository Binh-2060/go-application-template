//go:build integration

package tests

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Binh-2060/go-application-template/internal/api/models"
	"github.com/Binh-2060/go-application-template/internal/api/repositories"
	"github.com/Binh-2060/go-application-template/pkg/db"
)

/*
Repository tests against a real PostgreSQL.

	go test -tags=integration ./internal/api/...

These assert behaviour that only the real server can confirm: what pgx returns
for a missing row, whether ILIKE is genuinely case-insensitive, whether a
rollback actually removes rows. Everything provable from the generated SQL
alone lives in user_repository_sql_test.go, which needs no database.
*/

// Create a user and fail the test if it does not work — for the rows that are
// setup rather than the thing under test.
func mustCreate(t *testing.T, ctx context.Context, name, surename string) string {
	t.Helper()

	u, err := repositories.CreateUser(ctx, name, surename)
	if err != nil {
		t.Fatalf("repositories.CreateUser(%q): %v", name, err)
	}

	return u.ID
}

/*
The database supplies id and created_at; RETURNING must bring them back, or
every caller needs a second query to learn what it just wrote.
*/
func TestCreateUser_ReturnsGeneratedFields(t *testing.T) {
	ctx := context.Background()
	m := marker(t)

	user, err := repositories.CreateUser(ctx, m+"Ada", "Lovelace")
	if err != nil {
		t.Fatalf("repositories.CreateUser: %v", err)
	}

	if user.ID == "" {
		t.Error("ID is empty; the generated uuid was not returned")
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero; the default was not returned")
	}
	if user.Name != m+"Ada" || user.Surename != "Lovelace" {
		t.Errorf("stored row = %q / %q, want %q / %q", user.Name, user.Surename, m+"Ada", "Lovelace")
	}
}

/*
pgx reports a missing row as pgx.ErrNoRows; the repository must translate it, or
every caller ends up importing pgx to find out whether a 404 is warranted.
*/
func TestGetUserByID_MissingReturnsErrUserNotFound(t *testing.T) {
	_, err := repositories.GetUserByID(context.Background(), "00000000-0000-0000-0000-000000000000")

	if !errors.Is(err, repositories.ErrUserNotFound) {
		t.Fatalf("err = %v, want repositories.ErrUserNotFound", err)
	}
}

/*
The point of the pointer fields: a nil one means "leave this column alone".

This is the regression test for the dynamic SET clause. A builder bug that
emitted every column would silently overwrite surename with its zero value, and
nothing else in the suite would notice.
*/
func TestUpdateUser_PartialKeepsOtherColumn(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	id := mustCreate(t, ctx, m+"Ada", "Lovelace")

	newName := m + "Grace"
	updated, err := repositories.UpdateUser(ctx, id, &newName, nil)
	if err != nil {
		t.Fatalf("repositories.UpdateUser: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("Name = %q, want %q", updated.Name, newName)
	}
	if updated.Surename != "Lovelace" {
		t.Errorf("Surename = %q, want it untouched as %q", updated.Surename, "Lovelace")
	}
}

func TestUpdateUser_AllFieldsAreApplied(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	id := mustCreate(t, ctx, m+"Ada", "Lovelace")

	name, surename := m+"Grace", "Hopper"
	updated, err := repositories.UpdateUser(ctx, id, &name, &surename)
	if err != nil {
		t.Fatalf("repositories.UpdateUser: %v", err)
	}

	if updated.Name != name || updated.Surename != surename {
		t.Errorf("row = %q / %q, want %q / %q", updated.Name, updated.Surename, name, surename)
	}
}

/*
An UPDATE with an empty SET is not valid SQL, so this is caught before the query
is built. The alternative is a syntax error from the driver, which tells the
caller nothing about what they did wrong.
*/
func TestUpdateUser_NoFieldsReturnsErrNoUpdateFields(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	id := mustCreate(t, ctx, m+"Ada", "Lovelace")

	_, err := repositories.UpdateUser(ctx, id, nil, nil)

	if !errors.Is(err, repositories.ErrNoUpdateFields) {
		t.Fatalf("err = %v, want repositories.ErrNoUpdateFields", err)
	}
}

/*
UPDATE ... RETURNING on a row that does not exist returns no rows, which must
come back as the not-found sentinel rather than an empty struct and a nil error.
*/
func TestUpdateUser_MissingRowReturnsErrUserNotFound(t *testing.T) {
	name := "irrelevant"

	_, err := repositories.UpdateUser(context.Background(), "00000000-0000-0000-0000-000000000000", &name, nil)

	if !errors.Is(err, repositories.ErrUserNotFound) {
		t.Fatalf("err = %v, want repositories.ErrUserNotFound", err)
	}
}

/*
DELETE does not produce ErrNoRows — deleting nothing is a perfectly successful
statement. DeleteUser does not check rows-affected either, so deleting an id
that matches nothing is a no-op: nil error, not ErrUserNotFound.
*/
func TestDeleteUser_MissingRowIsANoOp(t *testing.T) {
	err := repositories.DeleteUser(context.Background(), "00000000-0000-0000-0000-000000000000")

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestDeleteUser_RemovesTheRow(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	id := mustCreate(t, ctx, m+"Ada", "Lovelace")

	if err := repositories.DeleteUser(ctx, id); err != nil {
		t.Fatalf("repositories.DeleteUser: %v", err)
	}

	if _, err := repositories.GetUserByID(ctx, id); !errors.Is(err, repositories.ErrUserNotFound) {
		t.Fatalf("after delete, repositories.GetUserByID err = %v, want repositories.ErrUserNotFound", err)
	}
}

/*
ILIKE, not LIKE: the filter is meant to be case-insensitive and to match
anywhere in the name. Searching an upper-cased fragment of a mixed-case name
exercises both halves at once.
*/
func TestListUsers_FilterIsCaseInsensitiveSubstring(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	mustCreate(t, ctx, m+"Ada", "Lovelace")
	mustCreate(t, ctx, m+"Grace", "Hopper")

	users, err := repositories.ListUsers(ctx, repositories.UserFilter{Q: strings.ToUpper(m + "ada")}, 10, 0)
	if err != nil {
		t.Fatalf("repositories.ListUsers: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("got %d users, want 1: %+v", len(users), users)
	}
	if users[0].Name != m+"Ada" {
		t.Errorf("Name = %q, want %q", users[0].Name, m+"Ada")
	}
}

/*
The end-to-end version of TestApplyUserFilter_BindsValueAsArgument.

A filter that closes a quote and ORs a tautology is just an unusual string. If
this ever returns rows, the value stopped being a bound argument.
*/
func TestListUsers_InjectionShapedFilterMatchesNothing(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	mustCreate(t, ctx, m+"Ada", "Lovelace")

	users, err := repositories.ListUsers(ctx, repositories.UserFilter{Q: m + "' OR '1'='1"}, 10, 0)
	if err != nil {
		t.Fatalf("repositories.ListUsers: %v", err)
	}

	if len(users) != 0 {
		t.Fatalf("got %d users, want 0 — the filter value reached the SQL as text", len(users))
	}
}

/*
An empty result must be an empty slice, not nil, so the JSON is [] and clients
do not have to handle null as a third case alongside "some" and "none".
*/
func TestListUsers_NoMatchesReturnsEmptyNotNil(t *testing.T) {
	users, err := repositories.ListUsers(context.Background(), repositories.UserFilter{Q: marker(t)}, 10, 0)
	if err != nil {
		t.Fatalf("repositories.ListUsers: %v", err)
	}

	if users == nil {
		t.Fatal("got nil slice, want empty non-nil")
	}
	if len(users) != 0 {
		t.Fatalf("got %d users, want 0", len(users))
	}
}

/*
Three rows created in the same statement burst share a created_at to the
microsecond, which is exactly the case where ordering by created_at alone stops
being deterministic. With id as the tiebreaker the pages must partition the set:
no row on both pages, none missing.
*/
func TestListUsers_PagesDoNotOverlap(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	for _, n := range []string{"A", "B", "C"} {
		mustCreate(t, ctx, m+n, "Surname")
	}
	filter := repositories.UserFilter{Q: m}

	first, err := repositories.ListUsers(ctx, filter, 2, 0)
	if err != nil {
		t.Fatalf("repositories.ListUsers page 1: %v", err)
	}
	second, err := repositories.ListUsers(ctx, filter, 2, 2)
	if err != nil {
		t.Fatalf("repositories.ListUsers page 2: %v", err)
	}

	if len(first) != 2 {
		t.Errorf("page 1 has %d rows, want 2", len(first))
	}
	if len(second) != 1 {
		t.Errorf("page 2 has %d rows, want 1", len(second))
	}

	seen := map[string]bool{}
	for _, u := range append(append([]models.User{}, first...), second...) {
		if seen[u.ID] {
			t.Errorf("id %s appeared on both pages", u.ID)
		}
		seen[u.ID] = true
	}
	if len(seen) != 3 {
		t.Errorf("pages covered %d distinct rows, want 3", len(seen))
	}
}

/*
The count and the page must describe the same set of rows. A total computed
without the filter reports a page count the list can never fill.
*/
func TestCountUsers_HonoursTheSameFilter(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	mustCreate(t, ctx, m+"Ada", "Lovelace")
	mustCreate(t, ctx, m+"Grace", "Hopper")

	total, err := repositories.CountUsers(ctx, repositories.UserFilter{Q: m})
	if err != nil {
		t.Fatalf("repositories.CountUsers: %v", err)
	}

	if total != 2 {
		t.Errorf("count = %d, want 2", total)
	}
}

/*
The property that makes db.Q(ctx) worth having: repositories.CreateUser was written with no
knowledge of transactions, yet joins one when the context carries it — and is
undone with it.
*/
func TestCreateUser_JoinsAndRollsBackWithTheTransaction(t *testing.T) {
	ctx := context.Background()
	m := marker(t)
	boom := errors.New("boom")

	err := db.ExecTx(ctx, func(ctx context.Context, _ db.Querier) error {
		mustCreate(t, ctx, m+"Ada", "Lovelace")
		return boom
	})

	if !errors.Is(err, boom) {
		t.Fatalf("ExecTx err = %v, want boom", err)
	}

	total, err := repositories.CountUsers(ctx, repositories.UserFilter{Q: m})
	if err != nil {
		t.Fatalf("repositories.CountUsers: %v", err)
	}
	if total != 0 {
		t.Errorf("%d rows survived a rolled-back transaction, want 0", total)
	}
}
