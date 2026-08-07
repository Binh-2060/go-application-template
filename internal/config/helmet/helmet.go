package helmet

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
)

func SetHelmetMiddleware(app *fiber.App) {
	app.Use(helmet.New())
}
