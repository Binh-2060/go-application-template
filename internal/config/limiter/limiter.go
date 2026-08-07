package limiter

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

func SetAppLimiter(app *fiber.App) {
	app.Use(limiter.New(limiter.Config{
		Max:               1000,
		Expiration:        60 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))

}
