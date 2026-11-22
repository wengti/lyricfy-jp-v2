package dictionary

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository/repotest"
)

func entries(n int) []model.DictionaryEntry {
	out := make([]model.DictionaryEntry, n)
	for i := range out {
		out[i] = model.DictionaryEntry{ID: fmt.Sprint(i)}
	}
	return out
}

func query(page, pageSize int) ListQuery {
	return ListQuery{Page: page, PageSize: pageSize, Sort: repository.DefaultDictionarySort}
}

func TestList_Pages(t *testing.T) {
	repo := &repotest.DictionaryRepository{Entries: entries(45)}
	svc := NewService(repo)

	tests := []struct {
		name                string
		page, pageSize      int
		wantFirst           string
		wantLen, wantOffset int
	}{
		{"first page", 1, 20, "0", 20, 0},
		{"middle page", 2, 20, "20", 20, 20},
		{"last, partial page", 3, 20, "40", 5, 40},
		{"past the end", 4, 20, "", 0, 60},
		{"a selection of 30", 1, 30, "0", 30, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.List(context.Background(), query(tt.page, tt.pageSize))
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(got.Entries) != tt.wantLen {
				t.Errorf("got %d entries, want %d", len(got.Entries), tt.wantLen)
			}
			if tt.wantLen > 0 && got.Entries[0].ID != tt.wantFirst {
				t.Errorf("first entry = %s, want %s", got.Entries[0].ID, tt.wantFirst)
			}
			if got.Entries == nil {
				t.Error("Entries is nil; it must encode as []")
			}
			if got.Page != tt.page || got.PageSize != tt.pageSize || got.Total != 45 {
				t.Errorf("page %d, size %d, total %d; want %d, %d, 45", got.Page, got.PageSize, got.Total, tt.page, tt.pageSize)
			}
			calls := repo.Calls()
			if last := calls[len(calls)-1]; last.Offset != tt.wantOffset || last.Limit != tt.pageSize {
				t.Errorf("asked the repository for offset %d limit %d, want %d and %d", last.Offset, last.Limit, tt.wantOffset, tt.pageSize)
			}
		})
	}
}

func TestList_TotalPages(t *testing.T) {
	tests := []struct{ total, pageSize, want int }{
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{100, 20, 5},
	}
	for _, tt := range tests {
		svc := NewService(&repotest.DictionaryRepository{Entries: entries(tt.total)})
		got, err := svc.List(context.Background(), query(1, tt.pageSize))
		if err != nil {
			t.Fatal(err)
		}
		if got.TotalPages != tt.want {
			t.Errorf("%d entries at %d a page: TotalPages = %d, want %d", tt.total, tt.pageSize, got.TotalPages, tt.want)
		}
	}
}

func TestList_PassesFiltersThrough(t *testing.T) {
	repo := &repotest.DictionaryRepository{}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), ListQuery{
		Page: 1, PageSize: 20, Sort: repository.SortLowestSuccess, Search: "rain", Tag: "song",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := repo.Calls()[0]
	if got.Sort != repository.SortLowestSuccess || got.Search != "rain" || got.Tag != "song" {
		t.Errorf("repository got %+v", got)
	}
}

func TestList_RejectsInvalidQueries(t *testing.T) {
	tests := []struct {
		name string
		q    ListQuery
		want error
	}{
		{"page 0", ListQuery{Page: 0, PageSize: 20, Sort: repository.SortNewest}, ErrInvalidPage},
		{"negative page", ListQuery{Page: -1, PageSize: 20, Sort: repository.SortNewest}, ErrInvalidPage},
		{"page size 0", ListQuery{Page: 1, PageSize: 0, Sort: repository.SortNewest}, ErrInvalidPageSize},
		{"page size over the max", ListQuery{Page: 1, PageSize: MaxPageSize + 1, Sort: repository.SortNewest}, ErrInvalidPageSize},
		{"unknown sort", ListQuery{Page: 1, PageSize: 20, Sort: "random"}, ErrInvalidSort},
		{"empty sort", ListQuery{Page: 1, PageSize: 20}, ErrInvalidSort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &repotest.DictionaryRepository{}
			_, err := NewService(repo).List(context.Background(), tt.q)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if len(repo.Calls()) != 0 {
				t.Error("an invalid query reached the repository")
			}
		})
	}
}

func TestList_RepositoryError(t *testing.T) {
	dbErr := errors.New("connection refused")
	svc := NewService(&repotest.DictionaryRepository{Err: dbErr})

	if _, err := svc.List(context.Background(), query(1, 20)); !errors.Is(err, dbErr) {
		t.Fatalf("err = %v, want it to wrap %v", err, dbErr)
	}
}
