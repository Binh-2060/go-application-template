package requestbody

/*
Create a user.

Both fields are NOT NULL in the migration, and `name` / `surename` are
varchar(200) — the max=200 tags keep an oversized value from becoming a
database error the client cannot read.
*/
type CreateUser struct {
	Name     string `json:"name" form:"name" validate:"required,min=1,max=200"`
	Surename string `json:"surename" form:"surename" validate:"required,min=1,max=200"`
}

/*
Create many users in one request, committed as a single transaction.
*/
type CreateUsers struct {
	Users []CreateUser `json:"users" validate:"required,min=1,max=100,dive"`
}

/*
Partially update a user.

Pointers, not strings: they let the handler tell "field omitted" from "field set
to empty". A plain string cannot, since both arrive as "". The repository binds
a nil pointer as NULL and COALESCE leaves that column untouched.

`omitempty` means the min/max tags only fire when a value was actually supplied.
*/
type UpdateUser struct {
	Name     *string `json:"name" form:"name" validate:"omitempty,min=1,max=200"`
	Surename *string `json:"surename" form:"surename" validate:"omitempty,min=1,max=200"`
}

/*
Query string for the list endpoint.

Zero values mean "not supplied" and the service substitutes its defaults. Fiber
v3 binds these through `query` tags via c.Bind().Query — v2's QueryInt and
friends are gone.

Name is an optional case-insensitive substring filter. It is what makes the
list query dynamic, and therefore what the Squirrel builder in the repository
exists for.
*/
type ListUsers struct {
	Page    int    `query:"page" validate:"omitempty,min=1"`
	PerPage int    `query:"per_page" validate:"omitempty,min=1,max=100"`
	Name    string `query:"name" validate:"omitempty,max=200"`
}
