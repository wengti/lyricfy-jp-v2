package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
)

// DictionarySort is an order the dictionary can be listed in. The values are the ones v1 used,
// plus success_rate_desc.
type DictionarySort string

const (
	SortNewest            DictionarySort = "created_at_desc"
	SortOldest            DictionarySort = "created_at_asc"
	SortJapanese          DictionarySort = "japanese_asc"
	SortEnglish           DictionarySort = "english_asc"
	SortLowestSuccess     DictionarySort = "success_rate_asc"
	SortHighestSuccess    DictionarySort = "success_rate_desc"
	DefaultDictionarySort                = SortNewest
)

// successRate is NULL for words never quizzed, so they go last in both directions: with no
// results they're neither strong nor weak
const successRate = `success_count::float8 / NULLIF(attempt_count, 0)`

// orderBy is the ORDER BY for each sort. Every one ends in a unique column, so pages never
// overlap or skip rows when values tie. Text sorts use the C collation (code-point order), which
// puts hiragana in gojūon order and behaves the same on every Postgres. Japanese sorts by the
// reading, since code-point order of kanji means nothing to a reader.
var orderBy = map[DictionarySort]string{
	SortNewest:         `created_at DESC, id`,
	SortOldest:         `created_at, id`,
	SortJapanese:       `hiragana COLLATE "C", japanese_text COLLATE "C", id`,
	SortEnglish:        `lower(english_translation) COLLATE "C", id`,
	SortLowestSuccess:  successRate + ` ASC NULLS LAST, attempt_count DESC, created_at DESC, id`,
	SortHighestSuccess: successRate + ` DESC NULLS LAST, attempt_count DESC, created_at DESC, id`,
}

// Valid reports whether s is one of the sorts above.
func (s DictionarySort) Valid() bool {
	_, ok := orderBy[s]
	return ok
}

// DictionaryListParams selects one page of the dictionary.
type DictionaryListParams struct {
	Sort DictionarySort
	// Search, if set, keeps entries whose Japanese, reading or English contains it, ignoring case
	Search string
	// Tag, if set, keeps entries that have this tag
	Tag    string
	Limit  int
	Offset int
}

type DictionaryRepository interface {
	// List returns the entries on the page p selects, and how many entries match its filters
	// across all pages. An unknown sort is an error.
	List(ctx context.Context, p DictionaryListParams) ([]model.DictionaryEntry, int, error)
}

// PostgresDictionaryRepository reads the dictionary_entries table.
type PostgresDictionaryRepository struct {
	db *pgxpool.Pool
}

func NewPostgresDictionaryRepository(db *pgxpool.Pool) *PostgresDictionaryRepository {
	return &PostgresDictionaryRepository{db: db}
}

// dictionaryFilter is shared by the count and the page. $1 is the escaped search pattern
// (” for none), $2 the tag (” for none).
const dictionaryFilter = `
	WHERE ($1 = '' OR japanese_text ILIKE '%' || $1 || '%' ESCAPE '\'
	               OR hiragana ILIKE '%' || $1 || '%' ESCAPE '\'
	               OR english_translation ILIKE '%' || $1 || '%' ESCAPE '\')
	  AND ($2 = '' OR $2 = ANY (tags))`

func (r *PostgresDictionaryRepository) List(ctx context.Context, p DictionaryListParams) ([]model.DictionaryEntry, int, error) {
	order, ok := orderBy[p.Sort]
	if !ok {
		return nil, 0, fmt.Errorf("unknown dictionary sort %q", p.Sort)
	}
	search := escapeLike(p.Search)

	var total int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM dictionary_entries`+dictionaryFilter, search, p.Tag).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count dictionary entries: %w", err)
	}

	// order comes from the map above, never from the request
	rows, err := r.db.Query(ctx, `
		SELECT id, japanese_text, hiragana, english_translation,
		       example_japanese, example_furigana, example_english,
		       source_song, source_artist, source_lyrics_line, tags,
		       attempt_count, success_count, last_attempted_at, created_at, updated_at
		FROM dictionary_entries`+dictionaryFilter+`
		ORDER BY `+order+`
		LIMIT $3 OFFSET $4`,
		search, p.Tag, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("select dictionary entries: %w", err)
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.DictionaryEntry, error) {
		var e model.DictionaryEntry
		err := row.Scan(&e.ID, &e.JapaneseText, &e.Hiragana, &e.EnglishTranslation,
			&e.ExampleJapanese, &e.ExampleFurigana, &e.ExampleEnglish,
			&e.SourceSong, &e.SourceArtist, &e.SourceLyricsLine, &e.Tags,
			&e.AttemptCount, &e.SuccessCount, &e.LastAttemptedAt, &e.CreatedAt, &e.UpdatedAt)
		return e, err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("read dictionary entries: %w", err)
	}
	return entries, total, nil
}

// escapeLike makes s match literally inside an ILIKE pattern, so a search for "100%" or "a_b"
// doesn't act as a wildcard.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
