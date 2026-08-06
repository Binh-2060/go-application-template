package routes

import "github.com/gofiber/fiber/v3"

func SetRoutes(router fiber.Router) {
	sampleRoutes := router.Group("/sample-routes")
	SetSampleRoute(sampleRoutes)

}
