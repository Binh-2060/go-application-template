//go:build integration

package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/Binh-2060/go-application-template/examples/crud/repositories"
	"github.com/Binh-2060/go-application-template/examples/crud/schemas/requestbody"
	"github.com/Binh-2060/go-application-template/examples/crud/services"
)

/*
Service-layer tests.

	go test -tags=integration ./examples/crud/...

Two things live only at this layer and are unreachable from either side of it:
the transaction that makes a bulk create atomic, and the pagination defaults.

The rollback test in particular cannot be written against the HTTP API — the
validate tags reject an over-long value before it ever reaches the database, so
the only way to fail mid-transaction is to enter below the controller.
*/

/*
All or nothing: the rows before the failure must not survive it.

The last user's surename exceeds varchar(200), so Postgres rejects that INSERT
after the first two have already succeeded inside the transaction. Without
db.ExecTx around the loop, two orphans would remain and a client retrying the
same request would create them twice.
*/
func TestCreateUsers_RollsBackEveryRowOnFailure(t *testing.T) {
	ctx := context.Background()
	m := marker(t)

	_, err := services.CreateUsers(ctx, requestbody.CreateUsers{Users: []requestbody.CreateUser{
		{Name: m + "Ada", Surename: "Lovelace"},
		{Name: m + "Grace", Surename: "Hopper"},
		{Name: m + "Toolong", Surename: strings.Repeat("x", 300)},
	}})
	if err == nil {
		t.Fatal("services.CreateUsers succeeded; expected the over-long surename to be rejected")
	}

	total, err := repositories.CountUsers(ctx, repositories.UserFilter{Q: m})
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if total != 0 {
		t.Errorf("%d rows survived a failed bulk create, want 0", total)
	}
}

func TestCreateUsers_CommitsEveryRowOnSuccess(t *testing.T) {
	ctx := context.Background()
	m := marker(t)

	created, err := services.CreateUsers(ctx, requestbody.CreateUsers{Users: []requestbody.CreateUser{
		{Name: m + "Ada", Surename: "Lovelace"},
		{Name: m + "Grace", Surename: "Hopper"},
	}})
	if err != nil {
		t.Fatalf("services.CreateUsers: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("returned %d users, want 2", len(created))
	}

	total, err := repositories.CountUsers(ctx, repositories.UserFilter{Q: m})
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if total != 2 {
		t.Errorf("%d rows committed, want 2", total)
	}
}

/*
A zero Page/PerPage means "not supplied" — the query binder cannot distinguish
that from an explicit zero, so the service substitutes the defaults rather than
asking the database for page 0 of 0 rows.

The expected values are written as literals because defaultPage / defaultPerPage
are unexported and this test lives outside the package. Change them in
services/user_service.go and this test is what fails.
*/
func TestListUsers_AppliesDefaultsForZeroValues(t *testing.T) {
	const wantPage = 1 // services.defaultPage

	page, err := services.ListUsers(context.Background(), requestbody.ListUsers{Q: marker(t)})
	if err != nil {
		t.Fatalf("services.ListUsers: %v", err)
	}

	if page.CurrentPage != wantPage {
		t.Errorf("CurrentPage = %d, want %d", page.CurrentPage, wantPage)
	}
	if page.Users == nil {
		t.Error("Users is nil; an empty page must still serialise as []")
	}
}

/*
Page count is a ceiling, computed with integer arithmetic. Plain division
truncates, so 3 rows at 2 per page would report 1 page and hide the last row.
*/
func TestListUsers_TotalPageRoundsUp(t *testing.T) {
	ctx := context.Background()
	m := marker(t)

	if _, err := services.CreateUsers(ctx, requestbody.CreateUsers{Users: []requestbody.CreateUser{
		{Name: m + "A", Surename: "S"},
		{Name: m + "B", Surename: "S"},
		{Name: m + "C", Surename: "S"},
	}}); err != nil {
		t.Fatalf("services.CreateUsers: %v", err)
	}

	page, err := services.ListUsers(ctx, requestbody.ListUsers{Q: m, Page: 1, PerPage: 2})
	if err != nil {
		t.Fatalf("services.ListUsers: %v", err)
	}

	if page.TotalPage != 2 {
		t.Errorf("TotalPage = %d, want 2 (3 rows at 2 per page)", page.TotalPage)
	}
	if page.CurrentPageTotalItem != 2 {
		t.Errorf("CurrentPageTotalItem = %d, want 2", page.CurrentPageTotalItem)
	}
}
