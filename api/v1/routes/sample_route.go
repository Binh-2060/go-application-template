package routes

import (
	"github.com/Binh-2060/go-application-template/api/v1/controllers"
	"github.com/gofiber/fiber/v2"
)

func SetSampleRoute(router fiber.Router) {
	router.Get("/", controllers.GetSampleController)
}
