package dictionary

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository/repotest"
	dictservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/dictionary"
	"github.com/wengti/lyricfy-jp-v2/backend/pkg/response"
)

// These tests cover the handler on its own. Which URL it's mounted on and which middleware
// guards it is tested in the route package.

func newHandler(entries ...model.DictionaryEntry) (*Handler, *repotest.DictionaryRepository) {
	repo := &repotest.DictionaryRepository{Entries: entries}
	return NewHandler(dictservice.NewService(repo)), repo
}

// get sends GET /?query to h and returns the status and decoded JSON body.
func get(t *testing.T, h *Handler, query string) (int, map[string]any) {
	t.Helper()
	e := echo.New()
	e.HTTPErrorHandler = response.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	e.GET("/", h.List)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?"+query, nil))

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response is not JSON: %q", rec.Body.String())
	}
	return rec.Code, decoded
}

func TestList_Defaults(t *testing.T) {
	h, repo := newHandler()

	status, body := get(t, h, "")

	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %v", status, body)
	}
	want := repository.DictionaryListParams{Sort: repository.SortNewest, Limit: 20, Offset: 0}
	if got := repo.Calls()[0]; got != want {
		t.Errorf("repository got %+v, want %+v", got, want)
	}
}

func TestList_ReadsQueryParameters(t *testing.T) {
	h, repo := newHandler()

	status, body := get(t, h, "page=3&pageSize=30&sort=success_rate_desc&search=+rain+&tag=song")

	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %v", status, body)
	}
	want := repository.DictionaryListParams{
		Sort: repository.SortHighestSuccess, Search: "rain", Tag: "song", Limit: 30, Offset: 60,
	}
	if got := repo.Calls()[0]; got != want {
		t.Errorf("repository got %+v, want %+v", got, want)
	}
}

func TestList_ResponseShape(t *testing.T) {
	example := "雨が降る。"
	h, _ := newHandler(model.DictionaryEntry{
		ID: "1", JapaneseText: "雨", Hiragana: "あめ", EnglishTranslation: "rain",
		ExampleJapanese: &example, Tags: []string{}, AttemptCount: 4, SuccessCount: 3,
		CreatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	})

	_, body := get(t, h, "")

	data := body["data"].(map[string]any)
	for key, want := range map[string]float64{"page": 1, "pageSize": 20, "total": 1, "totalPages": 1} {
		if data[key] != want {
			t.Errorf("%s = %v, want %v", key, data[key], want)
		}
	}
	entries := data["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	entry := entries[0].(map[string]any)
	wantFields := map[string]any{
		"id": "1", "japaneseText": "雨", "hiragana": "あめ", "englishTranslation": "rain",
		"exampleJapanese": example, "exampleFurigana": nil, "lastAttemptedAt": nil,
		"attemptCount": float64(4), "successCount": float64(3), "createdAt": "2026-04-01T00:00:00Z",
	}
	for key, want := range wantFields {
		if got, ok := entry[key]; !ok || got != want {
			t.Errorf("%s = %v (present: %v), want %v", key, got, ok, want)
		}
	}
}

func TestList_EmptyResultIsAnEmptyList(t *testing.T) {
	h, _ := newHandler()

	_, body := get(t, h, "")

	entries, ok := body["data"].(map[string]any)["entries"].([]any)
	if !ok || len(entries) != 0 {
		t.Errorf("entries = %v, want []", body["data"].(map[string]any)["entries"])
	}
}

func TestList_BadRequests(t *testing.T) {
	h, _ := newHandler()

	tests := []struct {
		query     string
		wantError string
	}{
		{"page=abc", "page must be a whole number"},
		{"page=1.5", "page must be a whole number"},
		{"pageSize=x", "pageSize must be a whole number"},
		{"page=0", "page must be at least 1"},
		{"pageSize=0", "pageSize must be between 1 and 100"},
		{"pageSize=101", "pageSize must be between 1 and 100"},
		{"sort=random", `unknown sort "random"`},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			status, body := get(t, h, tt.query)
			if status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %v)", status, body)
			}
			if body["error"] != tt.wantError {
				t.Errorf("error = %v, want %q", body["error"], tt.wantError)
			}
		})
	}
}

func TestList_DatabaseErrorIsHidden(t *testing.T) {
	h, repo := newHandler()
	repo.Err = errors.New("connection refused to db.internal")

	status, body := get(t, h, "")

	if status != http.StatusInternalServerError || body["error"] != "Internal Server Error" {
		t.Fatalf("got %d %v, want a generic 500", status, body)
	}
}
