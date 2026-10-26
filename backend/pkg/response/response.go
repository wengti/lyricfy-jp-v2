// Package response standardises JSON responses:
//
//	success: {"data": ...}
//	error:   {"error": "message"}
package response

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

type successBody struct {
	Data any `json:"data"`
}

type errorBody struct {
	Error string `json:"error"`
}

func OK(c *echo.Context, data any) error {
	return c.JSON(http.StatusOK, successBody{Data: data})
}

func Error(c *echo.Context, status int, message string) error {
	return c.JSON(status, errorBody{Error: message})
}

// ErrorHandler renders every error returned by handlers and middleware in the standard shape.
// Server errors are logged and replaced by a generic message so internals never leak.
func ErrorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if resp, _ := echo.UnwrapResponse(c.Response()); resp != nil && resp.Committed {
			return
		}

		status := echo.StatusCode(err)
		if status == 0 {
			status = http.StatusInternalServerError
		}

		message := http.StatusText(status)
		if he, ok := err.(*echo.HTTPError); ok && he.Message != "" && status < 500 {
			message = he.Message
		}
		if status >= 500 {
			logger.Error("request failed", "method", c.Request().Method, "path", c.Path(), "error", err)
		}

		if err := Error(c, status, message); err != nil {
			logger.Error("failed to write error response", "error", err)
		}
	}
}
