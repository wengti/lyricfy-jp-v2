package repotest

import (
	"context"
	"sync"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
)

// DictionaryRepository is an in-memory repository.DictionaryRepository for testing the layers
// above it. It pages through Entries in the order given but doesn't sort, search or filter by
// tag: that's SQL, tested against a real database in the repository package. Calls records
// what each List was asked for, so tests can check what reached the repository.
type DictionaryRepository struct {
	// Err, when set, is returned by every method, to test what happens when the database fails
	Err     error
	Entries []model.DictionaryEntry

	mu    sync.Mutex
	calls []repository.DictionaryListParams
}

var _ repository.DictionaryRepository = (*DictionaryRepository)(nil)

func (r *DictionaryRepository) List(_ context.Context, p repository.DictionaryListParams) ([]model.DictionaryEntry, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, p)
	if r.Err != nil {
		return nil, 0, r.Err
	}
	start := min(p.Offset, len(r.Entries))
	end := min(start+p.Limit, len(r.Entries))
	return r.Entries[start:end], len(r.Entries), nil
}

// Calls returns the parameters of every List call so far.
func (r *DictionaryRepository) Calls() []repository.DictionaryListParams {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]repository.DictionaryListParams(nil), r.calls...)
}
