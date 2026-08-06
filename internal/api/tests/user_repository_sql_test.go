package tests

import (
	"strings"
	"testing"

	"github.com/Binh-2060/go-application-template/internal/api/repositories"
	sq "github.com/Masterminds/squirrel"
)

/*
Tests for the SQL the builder produces. No database, no build tag: these run on
every `go test ./...` and in CI.

They exist because the two properties that matter most here — Postgres-style
placeholders, and filter values staying bound arguments rather than SQL text —
are properties of the generated string, and a query that gets them wrong fails
at runtime, in production, on the one input nobody tried.
*/

/*
Squirrel's zero value emits '?' placeholders (MySQL). Postgres rejects those, so
the package builder must be the $-numbered one.
*/
func TestPlaceholderFormatIsDollar(t *testing.T) {
	query, _, err := repositories.Builder.Select("id").From("users").Where(sq.Eq{"id": "x"}).ToSql()
	if err != nil {
		t.Fatalf("ToSql: %v", err)
	}

	if !strings.Contains(query, "$1") {
		t.Errorf("expected a $1 placeholder, got %q", query)
	}
	if strings.Contains(query, "?") {
		t.Errorf("query still uses MySQL '?' placeholders: %q", query)
	}
}

/*
An empty filter must not add a WHERE clause at all — not `WHERE true`, not
`WHERE name ILIKE '%%'`. An unfiltered list should be a plain table scan the
planner can reason about.
*/
func TestApplyUserFilter_EmptyAddsNoWhere(t *testing.T) {
	for _, name := range []string{"", "   ", "\t\n"} {
		builder := repositories.Builder.Select("id").From("users")

		query, args, err := repositories.ApplyUserFilter(builder, repositories.UserFilter{Q: name}).ToSql()
		if err != nil {
			t.Fatalf("ToSql: %v", err)
		}

		if strings.Contains(strings.ToUpper(query), "WHERE") {
			t.Errorf("filter %q produced a WHERE clause: %q", name, query)
		}
		if len(args) != 0 {
			t.Errorf("filter %q bound %d args, want 0", name, len(args))
		}
	}
}

/*
The important one: the filter value must arrive as a bound argument, never as
text spliced into the statement.

A value that is an argument cannot change the shape of the query no matter what
it contains, which is what makes the injection-shaped input in the integration
test match zero rows instead of every row.
*/
func TestApplyUserFilter_BindsValueAsArgument(t *testing.T) {
	const evil = "' OR '1'='1"

	builder := repositories.Builder.Select("id").From("users")

	query, args, err := repositories.ApplyUserFilter(builder, repositories.UserFilter{Q: evil}).ToSql()
	if err != nil {
		t.Fatalf("ToSql: %v", err)
	}

	if strings.Contains(query, evil) {
		t.Fatalf("filter value was inlined into the SQL: %q", query)
	}
	if want := "name ILIKE $1"; !strings.Contains(query, want) {
		t.Errorf("query = %q, want it to contain %q", query, want)
	}

	if len(args) != 1 {
		t.Fatalf("got %d args, want 1", len(args))
	}
	if got, want := args[0], "%"+evil+"%"; got != want {
		t.Errorf("arg = %q, want %q", got, want)
	}
}

/*
A filter of spaces around a value is still a filter; the value is trimmed but
the clause is kept.
*/
func TestApplyUserFilter_TrimsValue(t *testing.T) {
	builder := repositories.Builder.Select("id").From("users")

	_, args, err := repositories.ApplyUserFilter(builder, repositories.UserFilter{Q: "  ada  "}).ToSql()
	if err != nil {
		t.Fatalf("ToSql: %v", err)
	}

	if len(args) != 1 {
		t.Fatalf("got %d args, want 1", len(args))
	}
	if got, want := args[0], "%ada%"; got != want {
		t.Errorf("arg = %q, want %q", got, want)
	}
}

/*
created_at is not unique, so paging ordered by it alone can show the same row on
two pages or skip it entirely. id is the tiebreaker that makes the order total.

This asserts against BuildListUsersQuery — the function ListUsers actually
calls. Rebuilding the query inside the test instead would assert only that the
test agrees with itself, and would keep passing after the tiebreaker was removed.
*/
func TestListQuery_OrdersByUniqueTiebreaker(t *testing.T) {
	query, _, err := repositories.BuildListUsersQuery(repositories.UserFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("BuildListUsersQuery: %v", err)
	}

	if want := "ORDER BY created_at DESC, id DESC"; !strings.Contains(query, want) {
		t.Errorf("query = %q, want it to contain %q", query, want)
	}
}

/*
Paging is applied by the database, not by slicing in Go: an offset that never
reached the SQL would read the whole table and return the same first page every
time.
*/
func TestListQuery_AppliesLimitAndOffset(t *testing.T) {
	query, _, err := repositories.BuildListUsersQuery(repositories.UserFilter{}, 20, 40)
	if err != nil {
		t.Fatalf("BuildListUsersQuery: %v", err)
	}

	for _, want := range []string{"LIMIT 20", "OFFSET 40"} {
		if !strings.Contains(query, want) {
			t.Errorf("query = %q, want it to contain %q", query, want)
		}
	}
}

/*
The filter must reach the real list query, not just ApplyUserFilter in
isolation — and still as a bound argument once the paging is layered on.
*/
func TestListQuery_FilterIsBoundArgument(t *testing.T) {
	query, args, err := repositories.BuildListUsersQuery(repositories.UserFilter{Q: "ada"}, 10, 0)
	if err != nil {
		t.Fatalf("BuildListUsersQuery: %v", err)
	}

	if want := "name ILIKE $1"; !strings.Contains(query, want) {
		t.Errorf("query = %q, want it to contain %q", query, want)
	}
	if len(args) != 1 || args[0] != "%ada%" {
		t.Errorf("args = %v, want [%%ada%%]", args)
	}
}
