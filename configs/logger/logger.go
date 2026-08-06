package logger

import (
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

/*
My Custom logger tags
*/
var myCustomLoggerTags = map[string]logger.LogFunc{
	/// Custom request body logger tag
	"customReqBody": func(output logger.Buffer, c fiber.Ctx, data *logger.Data, extraParam string) (int, error) {
		// If request body is empty, return empty string
		if c.Request().Body() == nil {
			return output.WriteString("")
		}

		// If request body is in form-data
		if contentType := strings.Split(c.Get("Content-Type"), ";")[0]; contentType == "multipart/form-data" {
			form, err := c.MultipartForm()
			if err != nil {
				return output.WriteString("ERROR_GETTING_FORM_DATA")
			}

			var builder strings.Builder
			for key, value := range form.Value {
				if len(value) > 0 && value[0] != "" {
					builder.WriteString(key + "=" + value[0] + "&")
				}
			}
			return output.WriteString(builder.String())
		}

		// If request body is in JSON
		msg := strings.ReplaceAll(string(c.Request().Body()), "\n", "")
		return output.WriteString(msg)
	},
}

/*
Set logger middleware for REST API for JSON data
*/
func SetLoggerMiddlewareJSON(app fiber.Router) {
	app.Use(logger.New(logger.Config{
		Format:     `{"time": "${time}", "status": "${status}", "method": "${method}", "latency": "${latency}", "ip": "${ip}", "path": "${path}", "query_param": "${queryParams}", "user": "${locals:user}", "request_id": "${requestid}", "request_body": "${customReqBody}", "request_headers": "-", "response_body": "-", "error": "${error}"}` + "\n",
		CustomTags: myCustomLoggerTags,
		Stream:     os.Stdout,
		TimeFormat: time.RFC3339,
		TimeZone:   "UTC",
	}))
}
