package response

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func serve(t *testing.T, h echo.HandlerFunc) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var logs bytes.Buffer
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(slog.New(slog.NewTextHandler(&logs, nil)))
	e.GET("/", h)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec, logs.String()
}

func TestOK_WrapsDataInEnvelope(t *testing.T) {
	rec, _ := serve(t, func(c *echo.Context) error {
		return OK(c, map[string]string{"hello": "world"})
	})

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"data":{"hello":"world"}}` {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestErrorHandler_ClientErrorKeepsMessage(t *testing.T) {
	rec, _ := serve(t, func(c *echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	})

	if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != `{"error":"email is required"}` {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestErrorHandler_ServerErrorHidesDetailsAndLogs(t *testing.T) {
	rec, logs := serve(t, func(c *echo.Context) error {
		return errors.New("database password is hunter2")
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("internal error leaked to client: %s", rec.Body.String())
	}
	if !strings.Contains(logs, "hunter2") {
		t.Errorf("internal error was not logged: %q", logs)
	}
}

func TestErrorHandler_UnknownRouteIs404(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(slog.Default())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}
