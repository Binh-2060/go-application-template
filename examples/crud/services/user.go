package services

import (
	"context"

	"github.com/Binh-2060/go-application-template/examples/crud/models"
	"github.com/Binh-2060/go-application-template/examples/crud/repositories"
	"github.com/Binh-2060/go-application-template/examples/crud/schemas/requestbody"
	"github.com/Binh-2060/go-application-template/examples/crud/schemas/responsebody"
	"github.com/Binh-2060/go-application-template/pkg/db"
)

/*
ErrUserNotFound is re-exported so controllers can map it to a 404 without
importing the repository layer. errors.Is against it still matches the error the
repository actually returns — it is the same value, not a copy.
*/
var ErrUserNotFound = repositories.ErrUserNotFound

/*
ErrNoUpdateFields is re-exported for the same reason: an update that would
change nothing is the client's mistake, not a server failure, so the controller
needs to recognise it as a 400.
*/
var ErrNoUpdateFields = repositories.ErrNoUpdateFields

const (
	defaultPage    = 1
	defaultPerPage = 20
)

/*
PagedUsers carries a page of users plus the counts presenters.ResponseSuccessListData
needs, so the service returns one value instead of four bare ints.
*/
type PagedUsers struct {
	Users                []responsebody.User
	CurrentPage          int
	CurrentPageTotalItem int
	TotalPage            int
}

/*
Create one user.
*/
func CreateUser(ctx context.Context, in requestbody.CreateUser) (responsebody.User, error) {
	var err error
	var user models.User
	err = db.ExecTx(ctx, func(ctx context.Context, _ db.Querier) error {
		userTx, err := repositories.CreateUser(ctx, in.Name, in.Surename)
		if err != nil {
			return err
		}

		user = userTx
		return nil
	})

	return responsebody.NewUser(user), err
}

/*
Create several users atomically.

db.ExecTx commits when fn returns nil and rolls back on any error, so a failure
on the last user leaves none of them behind. The repository calls inside fn need
no change: they resolve their Querier with db.Q(ctx), which returns this
transaction because ExecTx put it on the context it passes to fn.
*/
func CreateUsers(ctx context.Context, in requestbody.CreateUsers) ([]responsebody.User, error) {
	created := make([]responsebody.User, 0, len(in.Users))

	err := db.ExecTx(ctx, func(ctx context.Context, _ db.Querier) error {
		for _, u := range in.Users {
			user, err := repositories.CreateUser(ctx, u.Name, u.Surename)
			if err != nil {
				return err
			}
			created = append(created, responsebody.NewUser(user))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

/*
Fetch one user, or ErrUserNotFound.
*/
func GetUser(ctx context.Context, id string) (responsebody.User, error) {
	user, err := repositories.GetUserByID(ctx, id)
	if err != nil {
		return responsebody.User{}, err
	}

	return responsebody.NewUser(user), nil
}

/*
Fetch one page of users along with its pagination counts.

The count and the page are read in a single transaction so the total cannot
shift between the two queries and report a page count that disagrees with the
rows returned.
*/
func ListUsers(ctx context.Context, in requestbody.ListUsers) (PagedUsers, error) {
	page, perPage := in.Page, in.PerPage
	if page < 1 {
		page = defaultPage
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}

	var (
		total int
		users []responsebody.User
	)

	// One filter value for both queries, so the total always describes the same
	// set of rows the page was drawn from.
	filter := repositories.UserFilter{Q: in.Q}
	total, err := repositories.CountUsers(ctx, filter)
	if err != nil {
		return PagedUsers{}, err
	}

	offset := (page - 1) * perPage
	rows, err := repositories.ListUsers(ctx, filter, perPage, offset)
	if err != nil {
		return PagedUsers{}, err
	}

	users = responsebody.NewUsers(rows)
	totalPage := (total + perPage - 1) / perPage

	return PagedUsers{
		Users:                users,
		CurrentPage:          page,
		CurrentPageTotalItem: len(users),
		TotalPage:            totalPage,
	}, nil
}

/*
Apply a partial update and return the stored row.
*/
func UpdateUser(ctx context.Context, id string, in requestbody.UpdateUser) error {
	userInfo, err := repositories.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	err = db.ExecTx(ctx, func(ctx context.Context, _ db.Querier) error {
		_, err := repositories.UpdateUser(ctx, userInfo.ID, &in.Name, &in.Surename)
		if err != nil {
			return err
		}

		return nil
	})

	return nil
}

/*
Delete a user, or ErrUserNotFound.
*/
func DeleteUser(ctx context.Context, id string) error {
	err := db.ExecTx(ctx, func(ctx context.Context, _ db.Querier) error {
		err := repositories.DeleteUser(ctx, id)
		return err
	})
	return err
}
