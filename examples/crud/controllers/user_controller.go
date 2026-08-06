package controllers

import (
	"errors"

	"github.com/Binh-2060/go-application-template/examples/crud/schemas/requestbody"
	"github.com/Binh-2060/go-application-template/examples/crud/services"
	"github.com/Binh-2060/go-application-template/internal/api/presenters"
	"github.com/Binh-2060/go-application-template/internal/api/validators"
	"github.com/gofiber/fiber/v3"
)

// Handlers take fiber.Ctx by value — in v3 Ctx is an interface, not a struct
// pointer. They stay thin on purpose: validate input, call a service, shape the
// output through presenters. Anything else belongs a layer down.

/*
POST /users — create a user.
*/
func CreateUser(c fiber.Ctx) error {
	var body requestbody.CreateUser
	// Binds and runs the go-playground tags in one step. Fiber's own
	// StructValidator hook is not configured on this app, so tags are enforced
	// only when input goes through these helpers.
	if err := validators.ParseAndValidateBody(c, &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := services.CreateUser(c.Context(), body)
	if err != nil {
		return toHTTPError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(presenters.ResponseSuccess(user))
}

/*
POST /users/bulk — create many users in one transaction.
*/
func CreateUsers(c fiber.Ctx) error {
	var body requestbody.CreateUsers
	if err := validators.ParseAndValidateBody(c, &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	users, err := services.CreateUsers(c.Context(), body)
	if err != nil {
		return toHTTPError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(presenters.ResponseSuccess(users))
}

/*
GET /users?page=&per_page= — list users, paginated.
*/
func ListUsers(c fiber.Ctx) error {
	var query requestbody.ListUsers
	if err := validators.ParseAndValidateQueryParam(c, &query); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	page, err := services.ListUsers(c.Context(), query)
	if err != nil {
		return toHTTPError(err)
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccessListData(
		page.Users, page.CurrentPage, page.CurrentPageTotalItem, page.TotalPage,
	))
}

/*
GET /users/:id — fetch one user.
*/
func GetUser(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}

	user, err := services.GetUser(c.Context(), id)
	if err != nil {
		return toHTTPError(err)
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(user))
}

/*
PATCH /users/:id — partial update. Omitted fields keep their stored value.
*/
func UpdateUser(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}

	var body requestbody.UpdateUser
	if err := validators.ParseAndValidateBody(c, &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if body.Name == nil && body.Surename == nil {
		return fiber.NewError(fiber.StatusBadRequest, "no updatable field supplied")
	}

	user, err := services.UpdateUser(c.Context(), id, body)
	if err != nil {
		return toHTTPError(err)
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(user))
}

/*
DELETE /users/:id — delete one user.
*/
func DeleteUser(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}

	if err := services.DeleteUser(c.Context(), id); err != nil {
		return toHTTPError(err)
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(nil))
}

/*
Read and validate the :id path param.

Rejecting a malformed uuid here keeps a guaranteed-empty query off the database
and returns 400 (bad input) rather than 404 (valid input, no such row).

The alternative is c.Bind().URI(&out) with `uri:"id"` tags, which is worth it
once a route has several params; for a single uuid this is less ceremony.
*/
func userID(c fiber.Ctx) (string, error) {
	id := c.Params("id")
	if err := validators.ValidateUuid(id); err != nil {
		return "", fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return id, nil
}

/*
Translate service errors into HTTP errors.

Only the cases worth distinguishing are mapped; anything else is returned
untouched and main.go's ErrorHandler renders it as a 500 in the standard error
envelope.
*/
func toHTTPError(err error) error {
	switch {
	case errors.Is(err, services.ErrUserNotFound):
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	case errors.Is(err, services.ErrNoUpdateFields):
		return fiber.NewError(fiber.StatusBadRequest, "no updatable field supplied")
	}

	return err
}
