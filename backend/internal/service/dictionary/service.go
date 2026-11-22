// Package dictionary holds the business logic for the vocabulary list: listing it a page at a
// time, sorted and filtered.
package dictionary

import (
	"context"
	"errors"
	"fmt"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
)

const (
	// DefaultPageSize is what the dictionary page shows at a time
	DefaultPageSize = 20
	// MaxPageSize bounds one request. It leaves room for selections like "best 30".
	MaxPageSize = 100
)

var (
	ErrInvalidPage     = errors.New("page must be at least 1")
	ErrInvalidPageSize = fmt.Errorf("pageSize must be between 1 and %d", MaxPageSize)
	ErrInvalidSort     = errors.New("unknown sort")
)

// ListQuery asks for one page of the dictionary. Page counts from 1.
type ListQuery struct {
	Page     int
	PageSize int
	Sort     repository.DictionarySort
	Search   string
	Tag      string
}

// Page is one page of entries, with what the frontend needs to draw the pager.
type Page struct {
	Entries    []model.DictionaryEntry `json:"entries"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"pageSize"`
	Total      int                     `json:"total"`      // entries matching the filters, on all pages
	TotalPages int                     `json:"totalPages"` // 0 when nothing matches
}

type Service struct {
	entries repository.DictionaryRepository
}

func NewService(entries repository.DictionaryRepository) *Service {
	return &Service{entries: entries}
}

// List returns one page of the dictionary. A page past the last one is not an error: it comes
// back empty, with the real total, so the frontend can step back.
func (s *Service) List(ctx context.Context, q ListQuery) (*Page, error) {
	if q.Page < 1 {
		return nil, ErrInvalidPage
	}
	if q.PageSize < 1 || q.PageSize > MaxPageSize {
		return nil, ErrInvalidPageSize
	}
	if !q.Sort.Valid() {
		return nil, ErrInvalidSort
	}

	entries, total, err := s.entries.List(ctx, repository.DictionaryListParams{
		Sort:   q.Sort,
		Search: q.Search,
		Tag:    q.Tag,
		Limit:  q.PageSize,
		Offset: (q.Page - 1) * q.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list dictionary entries: %w", err)
	}
	if entries == nil {
		entries = []model.DictionaryEntry{} // [] in the JSON, not null
	}
	return &Page{
		Entries:    entries,
		Page:       q.Page,
		PageSize:   q.PageSize,
		Total:      total,
		TotalPages: (total + q.PageSize - 1) / q.PageSize,
	}, nil
}
