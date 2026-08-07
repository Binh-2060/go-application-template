package healthcheck

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
)

func SetAppHealthCheck(app *fiber.App) {
	app.Use(healthcheck.New(healthcheck.Config{
		ResponseFormat: healthcheck.FormatJSON,
	}))
}
