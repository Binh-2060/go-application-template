package routes

import "github.com/gofiber/fiber/v2"

func SetRoutes(router fiber.Router) {
	sampleRoutes := router.Group("/sample-routes")
	SetSampleRoute(sampleRoutes)

}
