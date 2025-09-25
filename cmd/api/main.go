package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Binh-2060/go-application-template/api/v1/routes"
	"github.com/Binh-2060/go-application-template/api/validators"
	"github.com/Binh-2060/go-application-template/config/compress"
	"github.com/Binh-2060/go-application-template/config/cors"
	"github.com/Binh-2060/go-application-template/config/dotenv"
	"github.com/Binh-2060/go-application-template/config/helmet"
	"github.com/Binh-2060/go-application-template/config/logger"
	requestid "github.com/Binh-2060/go-application-template/config/requestId"
	dbpkg "github.com/Binh-2060/go-application-template/pkg/db-pkg"
	"github.com/gofiber/fiber/v2"
)

func init() {
	mode := os.Getenv("GO_ENV")
	if mode == "" {
		dotenv.SetDotenv()
		mode = os.Getenv("GO_ENV")
	}

	//logging MODE of app
	log.Println("------ Running in '" + mode + "' mode... ------")
}

func main() {
	var apiName = os.Getenv("API_NAME")
	var mode = os.Getenv("GO_ENV")
	var buildAt = os.Getenv("BUILD_DATE")
	var startRunAt = time.Now().Format("2006-01-02 15:04:05")

	myConfig := fiber.Config{
		AppName: apiName,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			// Status code defaults to 500
			code := fiber.StatusInternalServerError

			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			//response error
			err = ctx.Status(code).JSON(fiber.Map{
				"timestamp": time.Now().Format("2006-01-02-15-04-05"),
				"status":    0,
				"items":     nil,
				"error":     err.Error(),
			})
			return err
		},
	}

	app := fiber.New(myConfig)
	//CORS
	cors.SetCORSMiddleware(app)
	//requestId
	requestid.SetRequestIdMiddleware(app)
	//validators
	validators.Init()
	//compress
	compress.SetCompressMiddleware(app)
	//helmet
	helmet.SetHelmetMiddleware(app)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"API_NAME":     apiName,
			"MODE":         mode,
			"BUILD_AT":     buildAt,
			"START_RUN_AT": startRunAt,
		})
	})

	//check health status
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "OK",
		})
	})

	apiV1 := app.Group("/api/v1")
	routes.SetRoutes(apiV1)

	//logging
	logger.SetLoggerMiddlewareJSON(apiV1)

	dbpkg.CreateDatabaseConnection()

	// Run server in a separate goroutine so it doesn't block
	go func() {
		if err := app.Listen(":" + os.Getenv("PORT")); err != nil {
			log.Panic(err)
		}
	}()

	// Create channel to signify a signal being sent
	c := make(chan os.Signal, 1)

	// When an interrupt or termination signal is sent, notify the channel
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	_ = <-c // This blocks the main thread until an interrupt is received
	log.Println("Gracefully shutting down...")
	_ = app.Shutdown()

	log.Println("Running cleanup tasks...")
	// Your cleanup tasks go here ...

	log.Println("Fiber was successful shutdown.")
}
