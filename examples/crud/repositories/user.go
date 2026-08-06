package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Binh-2060/go-application-template/examples/crud/models"
	"github.com/Binh-2060/go-application-template/pkg/db"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

/*
ErrUserNotFound is returned when a query matched no row.

The repository stays HTTP-agnostic: the service layer re-exports this sentinel
and the controller turns it into a 404. Nothing below the service knows what an
HTTP status code is.
*/
var ErrUserNotFound = errors.New("user not found")

/*
ErrNoUpdateFields is returned when an update was asked to change nothing.

An UPDATE with an empty SET clause is not valid SQL, so this is caught before
the query is built rather than surfacing as a syntax error from the driver.
*/
var ErrNoUpdateFields = errors.New("no fields to update")

// One column list for every query, so scan order can never drift between them.
var userColumns = []string{"id", "name", "surename", "created_at"}

// Same list as a fragment, for the hand-written queries below.
var userColumnList = strings.Join(userColumns, ", ")

/*
Builder preconfigured for PostgreSQL.

Squirrel defaults to '?' placeholders (MySQL); Postgres needs $1, $2, … Setting
it once here means no individual query can forget.

Note what is *not* used: RunWith and Squirrel's Query/Scan helpers. Those are
built on database/sql, and this project talks to Postgres through native pgx.
Squirrel is used strictly as a string builder — ToSql() produces the SQL and
args, and db.Q(ctx) executes them.

Exported so the tests in examples/crud/tests can build the same statements the
repository does. Nothing outside this package and that one should need it.
*/
var Builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

/*
UserFilter narrows a list or count query. A zero value matches everything.

Both ListUsers and CountUsers apply it through the same helper, which is what
keeps a filtered page and its total from disagreeing.
*/
type UserFilter struct {
	// String column matches case-insensitively anywhere in the String. Empty means no
	// String or Q filter.
	Q string
}

// Every query goes through db.Q(ctx), never db.Pool(): Q hands back the
// in-flight transaction when one is on the context and the pool otherwise, so
// each of these functions works unchanged inside or outside db.ExecTx.

/*
Insert a user and return the stored row.

Hand-written: the statement is fixed, and building four known columns through a
builder would be longer than the SQL it replaced. RETURNING avoids a second
round trip for the server-generated id and created_at.
*/
func CreateUser(ctx context.Context, name, surename string) (models.User, error) {
	q := `INSERT INTO users (name, surename) VALUES ($1, $2) RETURNING ` + userColumnList

	return scanUser(db.Q(ctx).QueryRow(ctx, q, name, surename))
}

/*
Fetch a single user by id, or ErrUserNotFound.
*/
func GetUserByID(ctx context.Context, id string) (models.User, error) {
	q := `SELECT ` + userColumnList + ` FROM users WHERE id = $1`

	return scanUser(db.Q(ctx).QueryRow(ctx, q, id))
}

/*
Fetch one page of users, newest first, optionally filtered.

Built with Squirrel because the WHERE clause is not known until runtime. Doing
this by hand means concatenating clause fragments and hand-numbering $1, $2, $3
as conditions come and go — which is where both injection bugs and placeholder
off-by-ones come from. The builder numbers them.

id is the tiebreaker in ORDER BY: created_at alone is not unique, and without a
deterministic total order the same row can appear on two pages or on none.
*/
func ListUsers(ctx context.Context, filter UserFilter, limit, offset int) ([]models.User, error) {
	query, args, err := BuildListUsersQuery(filter, limit, offset)
	if err != nil {
		// Only reachable via a malformed builder call, i.e. a bug here rather
		// than anything the caller did.
		return nil, fmt.Errorf("build list users query: %w", err)
	}

	rows, err := db.Q(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	// Non-nil empty slice: encodes as [] rather than null.
	users := make([]models.User, 0, limit)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Surename, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	// rows.Err reports failures that happened mid-stream, which Next() reports
	// only as "no more rows".
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}

/*
Count users matching the same filter, for the page count.

Shares ApplyUserFilter with ListUsers on purpose: a count built from a different
WHERE clause than the page it describes is a bug waiting to happen.
*/
func CountUsers(ctx context.Context, filter UserFilter) (int, error) {
	query, args, err := ApplyUserFilter(Builder.Select("count(*)").From("users"), filter).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count users query: %w", err)
	}

	var total int
	if err := db.Q(ctx).QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return total, nil
}

/*
Partially update a user and return the stored row.

Also built with Squirrel: a nil pointer means "leave this column alone", so the
SET clause is only known at runtime. The builder emits exactly the columns being
changed, which reads as what it does — the hand-written alternative is
COALESCE($2, name), workable for two columns and unpleasant by eight.

Returns ErrNoUpdateFields when every field is nil, since an UPDATE with no SET
is not a statement.
*/
func UpdateUser(ctx context.Context, id string, name, surename *string) (models.User, error) {
	update := Builder.Update("users").Where(sq.Eq{"id": id})

	changed := false
	if name != nil {
		update = update.Set("name", *name)
		changed = true
	}
	if surename != nil {
		update = update.Set("surename", *surename)
		changed = true
	}
	if !changed {
		return models.User{}, ErrNoUpdateFields
	}

	query, args, err := update.Suffix("RETURNING " + userColumnList).ToSql()
	if err != nil {
		return models.User{}, fmt.Errorf("build update user query: %w", err)
	}

	return scanUser(db.Q(ctx).QueryRow(ctx, query, args...))
}

/*
Delete a user, or ErrUserNotFound if the id matched nothing.

DELETE does not return ErrNoRows, so absence has to be read off the command tag.
*/
func DeleteUser(ctx context.Context, id string) error {
	_, err := db.Q(ctx).Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

/*
Assemble the list query.

Split out from ListUsers, and exported, so the generated SQL can be asserted
without a database — the placeholder style and the ORDER BY tiebreaker are
properties of this string, and a test that rebuilt the query itself would only
be testing itself.
*/
func BuildListUsersQuery(filter UserFilter, limit, offset int) (string, []any, error) {
	return ApplyUserFilter(Builder.Select(userColumns...).From("users"), filter).
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
}

/*
Add the filter's conditions to a builder.

sq.ILike renders `name ILIKE $n` or string value `COLUMN ILIKE $n` with the pattern as a bound argument, so the
value is never spliced into the SQL text.
*/
func ApplyUserFilter(builder sq.SelectBuilder, filter UserFilter) sq.SelectBuilder {
	if q := strings.TrimSpace(filter.Q); q != "" {
		builder = builder.Where(sq.ILike{"name": "%" + q + "%"})
	}

	return builder
}

// Shared by every single-row query so the ErrNoRows translation lives in one place.
func scanUser(row pgx.Row) (models.User, error) {
	var u models.User

	err := row.Scan(&u.ID, &u.Name, &u.Surename, &u.CreatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return models.User{}, ErrUserNotFound
	case err != nil:
		return models.User{}, fmt.Errorf("scan user: %w", err)
	}

	return u, nil
}
