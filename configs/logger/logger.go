package logger

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

const (
	// redacted replaces the value of any field considered sensitive.
	redacted = "***REDACTED***"

	// maxLoggedBodyBytes caps how much of a body reaches the log. Bodies are
	// attacker-influenced, so an uncapped tag is a cheap way to flood the log.
	maxLoggedBodyBytes = 2048
)

/*
JSON codec used to encode the log record.

sonic uses a JIT-assembled codec on amd64/arm64 and falls back to
encoding/json elsewhere, so this stays portable. ConfigDefault is chosen over
ConfigStd for speed; its two divergences are harmless here — object keys keep
input order instead of being alphabetised, and '<', '>', '&' are left literal.
*/
var jsonCodec = sonic.ConfigDefault

// streamMu serialises writes so concurrent requests cannot interleave halves
// of two records. A log line carrying two 2KB bodies can exceed the pipe-atomic
// write size, so a single Write call is not sufficient protection on its own.
var streamMu sync.Mutex

/*
accessLog is one access-log record.

Encoding a struct rather than interpolating a format string is what makes the
output valid JSON by construction: bodies nest as real objects and every value
is escaped by the encoder, so no field can break the surrounding record.
*/
type accessLog struct {
	Timestamp    string  `json:"timestamp"`
	Status       int     `json:"status"`
	Method       string  `json:"method"`
	LatencyMS    float64 `json:"latency_ms"`
	IP           string  `json:"ip"`
	Path         string  `json:"path"`
	QueryParam   string  `json:"query_param,omitempty"`
	User         string  `json:"user,omitempty"`
	RequestID    string  `json:"request_id,omitempty"`
	RequestBody  any     `json:"request_body"`
	ResponseBody any     `json:"response_body"`
	Error        *string `json:"error"`
}

/*
Field names whose values must never be logged.

sensitiveKeys holds names matched exactly — they are short enough that a
substring match would produce false positives ("pin" would redact "shipping").
sensitiveFragments holds names matched as substrings, so "user_password" and
"newPasswordConfirm" are caught alongside "password".

Matching is case-insensitive and ignores '-' and '_', so "API-Key", "api_key"
and "apiKey" all resolve to the same fragment.
*/
var sensitiveKeys = map[string]struct{}{
	"pin":  {},
	"otp":  {},
	"cvv":  {},
	"cvc":  {},
	"auth": {},
	"key":  {},
}

var sensitiveFragments = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"apikey",
	"authorization",
	"credential",
	"privatekey",
	"accesskey",
	"cardnumber",
	"creditcard",
	"ssn",
	"session",
	"signature",
}

var keyNormaliser = strings.NewReplacer("_", "", "-", "", " ", "")

/*
Report whether a field name should have its value redacted.
*/
func isSensitive(key string) bool {
	normalised := keyNormaliser.Replace(strings.ToLower(key))

	if _, ok := sensitiveKeys[normalised]; ok {
		return true
	}

	for _, fragment := range sensitiveFragments {
		if strings.Contains(normalised, fragment) {
			return true
		}
	}

	return false
}

/*
Recursively redact sensitive fields in a decoded JSON value.

A sensitive key is replaced wholesale rather than descended into, so nesting an
object under "credentials" cannot leak its children.
*/
func maskValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, val := range typed {
			if isSensitive(key) {
				out[key] = redacted
				continue
			}
			out[key] = maskValue(val)
		}
		return out

	case []any:
		out := make([]any, len(typed))
		for i, val := range typed {
			out[i] = maskValue(val)
		}
		return out

	default:
		return value
	}
}

/*
Mask form values into an object.
*/
func maskFormValues(values map[string][]string) map[string]string {
	out := make(map[string]string, len(values))

	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		if isSensitive(key) {
			out[key] = redacted
			continue
		}
		out[key] = vals[0]
	}

	return out
}

/*
Decode a body into a loggable value, redacting sensitive fields.

Returns nil for an empty body, a nested object/array for JSON and form bodies,
or a short string marker when the body cannot be represented safely. Callers
must not assume the result is an object.
*/
func maskBody(raw []byte, contentType string) any {
	if len(raw) == 0 {
		return nil
	}

	// Oversized bodies are described rather than parsed. Truncating structured
	// data would emit a fragment that no longer parses as what it claims to be.
	if len(raw) > maxLoggedBodyBytes {
		return "[body omitted: " + strconv.Itoa(len(raw)) + " bytes exceeds " +
			strconv.Itoa(maxLoggedBodyBytes) + " limit]"
	}

	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	switch {
	// Covers application/json plus vendor types such as application/vnd.api+json.
	case mediaType == fiber.MIMEApplicationJSON || strings.HasSuffix(mediaType, "+json"):
		var decoded any
		if err := jsonCodec.Unmarshal(raw, &decoded); err != nil {
			return "UNPARSEABLE_JSON_REDACTED"
		}
		return maskValue(decoded)

	case mediaType == fiber.MIMEApplicationForm:
		values, err := url.ParseQuery(string(raw))
		if err != nil {
			return "UNPARSEABLE_FORM_REDACTED"
		}
		return maskFormValues(values)

	case mediaType == "":
		return "[body omitted: no content-type]"

	// Anything this function cannot parse structurally, it cannot redact
	// safely, so the content is dropped and only its type is recorded.
	default:
		return "[body omitted: " + mediaType + "]"
	}
}

/*
Decode the request body, handling multipart separately.

The raw multipart payload carries part headers and file contents that must
never reach the log, so only the parsed text fields are considered.
*/
func maskRequestBody(c fiber.Ctx) any {
	body := c.Request().Body()
	if len(body) == 0 {
		return nil
	}

	contentType := c.Get(fiber.HeaderContentType)

	if strings.HasPrefix(strings.ToLower(contentType), fiber.MIMEMultipartForm) {
		form, err := c.MultipartForm()
		if err != nil {
			return "ERROR_GETTING_FORM_DATA"
		}
		return maskFormValues(form.Value)
	}

	return maskBody(body, contentType)
}

/*
Emit one request as a single JSON object.
*/
func jsonLoggerFunc(c fiber.Ctx, data *logger.Data, cfg *logger.Config) error {
	entry := accessLog{
		Timestamp:    data.Stop.UTC().Format(time.RFC3339),
		Status:       c.Response().StatusCode(),
		Method:       c.Method(),
		LatencyMS:    float64(data.Stop.Sub(data.Start).Nanoseconds()) / float64(time.Millisecond),
		IP:           c.IP(),
		Path:         c.Path(),
		QueryParam:   string(c.Request().URI().QueryString()),
		RequestID:    requestid.FromContext(c),
		RequestBody:  maskRequestBody(c),
		ResponseBody: maskBody(c.Response().Body(), string(c.Response().Header.ContentType())),
	}

	if user := c.Locals("user"); user != nil {
		entry.User = fmt.Sprint(user)
	}

	if data.ChainErr != nil {
		msg := data.ChainErr.Error()
		entry.Error = &msg
	}

	line, err := jsonCodec.Marshal(entry)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	streamMu.Lock()
	defer streamMu.Unlock()
	_, err = cfg.Stream.Write(line)

	return err
}

/*
Set logger middleware for REST API for JSON data
*/
func SetLoggerMiddlewareJSON(app fiber.Router) {
	app.Use(logger.New(logger.Config{
		LoggerFunc: jsonLoggerFunc,
		Stream:     os.Stdout,
		// Colours would wrap Stream in a terminal-aware writer and inject ANSI
		// escapes into the JSON. Format/CustomTags are unused: LoggerFunc
		// replaces the template renderer outright.
		DisableColors: true,
	}))
}
