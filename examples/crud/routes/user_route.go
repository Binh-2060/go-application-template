package routes

import (
	"github.com/Binh-2060/go-application-template/examples/crud/controllers"
	"github.com/gofiber/fiber/v3"
)

/*
Mount the user CRUD endpoints on router.

Mirrors internal/api/routes: routes.go groups by feature path and delegates to a
Set<Feature>Route function like this one. To wire the example into the running
app, add to internal/api/routes/routes.go:

	userRoutes := router.Group("/users")
	crudroutes.SetUserRoute(userRoutes)

/bulk is registered before /:id so the literal segment is not swallowed by the
param route.
*/
func SetUserRoute(router fiber.Router) {
	router.Post("/", controllers.CreateUser)
	router.Post("/bulk", controllers.CreateUsers)
	router.Get("/", controllers.ListUsers)
	router.Get("/:id", controllers.GetUser)
	router.Patch("/:id", controllers.UpdateUser)
	router.Delete("/:id", controllers.DeleteUser)
}
