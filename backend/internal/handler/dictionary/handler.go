// Package dictionary holds the HTTP handlers for /api/dictionary: they read the request, call
// the dictionary service and write responses. URLs are mapped in the route package.
package dictionary

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
	dictservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/dictionary"
	"github.com/wengti/lyricfy-jp-v2/backend/pkg/response"
)

type Handler struct {
	svc *dictservice.Service
}

func NewHandler(svc *dictservice.Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/dictionary. Every query parameter is optional:
//
//	page      page number, from 1 (default 1)
//	pageSize  entries per page, 1 to 100 (default 20)
//	sort      created_at_desc (default), created_at_asc, japanese_asc, english_asc,
//	          success_rate_asc or success_rate_desc
//	search    keep entries whose Japanese, reading or English contains this, ignoring case
//	tag       keep entries with this tag
func (h *Handler) List(c *echo.Context) error {
	page, err := intParam(c, "page", 1)
	if err != nil {
		return err
	}
	pageSize, err := intParam(c, "pageSize", dictservice.DefaultPageSize)
	if err != nil {
		return err
	}
	sort := repository.DictionarySort(c.QueryParam("sort"))
	if sort == "" {
		sort = repository.DefaultDictionarySort
	}

	result, err := h.svc.List(c.Request().Context(), dictservice.ListQuery{
		Page:     page,
		PageSize: pageSize,
		Sort:     sort,
		Search:   strings.TrimSpace(c.QueryParam("search")),
		Tag:      strings.TrimSpace(c.QueryParam("tag")),
	})
	switch {
	case errors.Is(err, dictservice.ErrInvalidPage), errors.Is(err, dictservice.ErrInvalidPageSize):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, dictservice.ErrInvalidSort):
		return echo.NewHTTPError(http.StatusBadRequest, "unknown sort "+strconv.Quote(string(sort)))
	case err != nil:
		return err
	}
	return response.OK(c, result)
}

// intParam reads an integer query parameter, or returns def when it's absent.
func intParam(c *echo.Context, name string, def int) (int, error) {
	raw := c.QueryParam(name)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, name+" must be a whole number")
	}
	return n, nil
}
