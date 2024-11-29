package helmet

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
)

func SetHelmetMiddleware(app *fiber.App) {
	app.Use(helmet.New())
}
