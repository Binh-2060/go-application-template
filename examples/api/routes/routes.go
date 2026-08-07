package routes

import "github.com/gofiber/fiber/v3"

func SetRoutes(router fiber.Router) {
	//sameple route
	sampleRoute := router.Group("/sameple")
	SetSampleRoute(sampleRoute)
}
