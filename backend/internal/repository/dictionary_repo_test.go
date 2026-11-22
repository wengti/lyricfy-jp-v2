package repository_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/database/testdb"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
)

// These tests run against a real Postgres (TEST_DATABASE_URL); see the testdb package.

// word is the part of an entry these tests vary. Entries are created a minute apart in the
// order given, so the first is the oldest.
type word struct {
	japanese, hiragana, english string
	attempts, successes         int
	tags                        []string
}

// seed inserts the words and returns a repository over them. The repository has no insert yet
// (nothing writes entries but the v1 import), so the rows go in with SQL.
func seed(t *testing.T, words ...word) (*repository.PostgresDictionaryRepository, *pgxpool.Pool) {
	t.Helper()
	db := testdb.New(t)
	for i, w := range words {
		created := testNow.Add(time.Duration(i) * time.Minute)
		tags := w.tags
		if tags == nil {
			tags = []string{}
		}
		_, err := db.Exec(context.Background(), `
			INSERT INTO dictionary_entries
				(japanese_text, hiragana, english_translation, tags, attempt_count, success_count, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
			w.japanese, w.hiragana, w.english, tags, w.attempts, w.successes, created)
		if err != nil {
			t.Fatalf("insert %q: %v", w.japanese, err)
		}
	}
	return repository.NewPostgresDictionaryRepository(db), db
}

// list calls List and returns the Japanese of each entry, in order, and the total.
func list(t *testing.T, repo repository.DictionaryRepository, p repository.DictionaryListParams) ([]string, int) {
	t.Helper()
	if p.Sort == "" {
		p.Sort = repository.DefaultDictionarySort
	}
	if p.Limit == 0 {
		p.Limit = 100
	}
	entries, total, err := repo.List(context.Background(), p)
	if err != nil {
		t.Fatalf("List(%+v): %v", p, err)
	}
	got := make([]string, len(entries))
	for i, e := range entries {
		got[i] = e.JapaneseText
	}
	return got, total
}

func TestPostgresDictionaryRepository_List_Sorts(t *testing.T) {
	repo, _ := seed(t,
		word{japanese: "季節", hiragana: "きせつ", english: "season", attempts: 10, successes: 9},    // 90%
		word{japanese: "狂う", hiragana: "くるう", english: "to go mad", attempts: 10, successes: 2}, // 20%
		word{japanese: "中", hiragana: "なか", english: "Inside", attempts: 0, successes: 0},       // never quizzed
		word{japanese: "雨", hiragana: "あめ", english: "rain", attempts: 4, successes: 4},         // 100%, fewer attempts
		word{japanese: "空", hiragana: "そら", english: "sky", attempts: 8, successes: 8},          // 100%
	)

	tests := []struct {
		sort repository.DictionarySort
		want []string
	}{
		{repository.SortNewest, []string{"空", "雨", "中", "狂う", "季節"}},
		{repository.SortOldest, []string{"季節", "狂う", "中", "雨", "空"}},
		// By reading (あめ, きせつ, くるう, そら, なか), not by kanji code point
		{repository.SortJapanese, []string{"雨", "季節", "狂う", "空", "中"}},
		// Case-insensitive: "Inside" sorts among the lowercase words
		{repository.SortEnglish, []string{"中", "雨", "季節", "空", "狂う"}},
		// Never-quizzed words last in both directions; at equal rates, more attempts first
		{repository.SortLowestSuccess, []string{"狂う", "季節", "空", "雨", "中"}},
		{repository.SortHighestSuccess, []string{"空", "雨", "季節", "狂う", "中"}},
	}
	for _, tt := range tests {
		t.Run(string(tt.sort), func(t *testing.T) {
			got, total := list(t, repo, repository.DictionaryListParams{Sort: tt.sort})
			if !slices.Equal(got, tt.want) {
				t.Errorf("order = %v, want %v", got, tt.want)
			}
			if total != 5 {
				t.Errorf("total = %d, want 5", total)
			}
		})
	}
}

func TestPostgresDictionaryRepository_List_Pages(t *testing.T) {
	var words []word
	for i := range 45 {
		words = append(words, word{japanese: string(rune('A' + i)), hiragana: "あ", english: "x"})
	}
	repo, _ := seed(t, words...)

	// Walking the pages oldest first visits every entry exactly once, in order
	var all []string
	for offset := 0; ; offset += 20 {
		page, total := list(t, repo, repository.DictionaryListParams{Sort: repository.SortOldest, Limit: 20, Offset: offset})
		if total != 45 {
			t.Fatalf("total = %d, want 45", total)
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
	}
	want := make([]string, 45)
	for i := range want {
		want[i] = string(rune('A' + i))
	}
	if !slices.Equal(all, want) {
		t.Errorf("pages = %v, want %v", all, want)
	}

	// Every English is "x", so only id orders this sort; pages still neither overlap nor skip
	seen := map[string]bool{}
	for offset := 0; offset < 45; offset += 20 {
		page, _ := list(t, repo, repository.DictionaryListParams{Sort: repository.SortEnglish, Limit: 20, Offset: offset})
		for _, j := range page {
			if seen[j] {
				t.Fatalf("%q appears on two pages", j)
			}
			seen[j] = true
		}
	}
	if len(seen) != 45 {
		t.Errorf("saw %d entries across pages, want 45", len(seen))
	}
}

func TestPostgresDictionaryRepository_List_Filters(t *testing.T) {
	repo, _ := seed(t,
		word{japanese: "季節", hiragana: "きせつ", english: "season", tags: []string{"nature"}},
		word{japanese: "狂う", hiragana: "くるう", english: "to go MAD", tags: []string{"verb", "song"}},
		word{japanese: "百パーセント", hiragana: "ひゃくぱーせんと", english: "100% sure"},
		word{japanese: "雨", hiragana: "あめ", english: "rain", tags: []string{"nature", "song"}},
	)

	tests := []struct {
		name   string
		search string
		tag    string
		want   []string
	}{
		{"no filter", "", "", []string{"雨", "百パーセント", "狂う", "季節"}},
		{"search in Japanese", "節", "", []string{"季節"}},
		{"search in the reading", "くる", "", []string{"狂う"}},
		{"search in English ignores case", "mad", "", []string{"狂う"}},
		{"% is literal, not a wildcard", "0%", "", []string{"百パーセント"}},
		{"_ is literal, not a wildcard", "_", "", []string{}},
		{"tag", "", "nature", []string{"雨", "季節"}},
		{"tag must match exactly", "", "natur", []string{}},
		{"search and tag together", "rain", "song", []string{"雨"}},
		{"no match", "nothing", "", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total := list(t, repo, repository.DictionaryListParams{Search: tt.search, Tag: tt.tag})
			if !slices.Equal(got, tt.want) {
				t.Errorf("entries = %v, want %v", got, tt.want)
			}
			if total != len(tt.want) {
				t.Errorf("total = %d, want %d", total, len(tt.want))
			}
		})
	}

	t.Run("total counts every match, not just the page", func(t *testing.T) {
		got, total := list(t, repo, repository.DictionaryListParams{Tag: "song", Limit: 1})
		if len(got) != 1 || total != 2 {
			t.Errorf("got %v with total %d, want 1 entry with total 2", got, total)
		}
	})
}

func TestPostgresDictionaryRepository_List_ReadsEveryColumn(t *testing.T) {
	repo, db := seed(t)
	lastAttempt := testNow.Add(-time.Hour)
	_, err := db.Exec(context.Background(), `
		INSERT INTO dictionary_entries VALUES (
			'f2add063-2eb7-4062-b5ba-7f07b57f140c', '観光', 'かんこう', 'sightseeing',
			'京都は観光地です。', '[{"original": "京都", "reading": "きょうと"}, {"original": "は", "reading": null}]',
			'Kyoto is a tourist spot.', 'ヒッチコック', 'ヨルシカ', 'a line', '{travel}',
			5, 3, $1, $2, $3)`,
		lastAttempt, testNow, testNow.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	// A bare entry: every optional column NULL
	if _, err := db.Exec(context.Background(), `
		INSERT INTO dictionary_entries (japanese_text, hiragana, english_translation, created_at, updated_at)
		VALUES ('雨', 'あめ', 'rain', $1, $1)`, testNow.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	entries, _, err := repo.List(context.Background(), repository.DictionaryListParams{Sort: repository.SortNewest, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	full := entries[0]
	str := func(p *string) string {
		if p == nil {
			return "<nil>"
		}
		return *p
	}
	checks := []struct{ field, got, want string }{
		{"ID", full.ID, "f2add063-2eb7-4062-b5ba-7f07b57f140c"},
		{"JapaneseText", full.JapaneseText, "観光"},
		{"Hiragana", full.Hiragana, "かんこう"},
		{"EnglishTranslation", full.EnglishTranslation, "sightseeing"},
		{"ExampleJapanese", str(full.ExampleJapanese), "京都は観光地です。"},
		{"ExampleEnglish", str(full.ExampleEnglish), "Kyoto is a tourist spot."},
		{"SourceSong", str(full.SourceSong), "ヒッチコック"},
		{"SourceArtist", str(full.SourceArtist), "ヨルシカ"},
		{"SourceLyricsLine", str(full.SourceLyricsLine), "a line"},
		{"Tags", strings.Join(full.Tags, ","), "travel"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
	reading := "きょうと"
	wantFurigana := []model.FuriganaSegment{{Original: "京都", Reading: &reading}, {Original: "は"}}
	if len(full.ExampleFurigana) != 2 || full.ExampleFurigana[0].Original != "京都" ||
		str(full.ExampleFurigana[0].Reading) != "きょうと" || full.ExampleFurigana[1].Reading != nil {
		t.Errorf("ExampleFurigana = %+v, want %+v", full.ExampleFurigana, wantFurigana)
	}
	if full.AttemptCount != 5 || full.SuccessCount != 3 {
		t.Errorf("counts = %d/%d, want 3/5", full.SuccessCount, full.AttemptCount)
	}
	if full.LastAttemptedAt == nil || !full.LastAttemptedAt.Equal(lastAttempt) {
		t.Errorf("LastAttemptedAt = %v, want %v", full.LastAttemptedAt, lastAttempt)
	}
	if !full.CreatedAt.Equal(testNow) || !full.UpdatedAt.Equal(testNow.Add(time.Minute)) {
		t.Errorf("timestamps = %v / %v", full.CreatedAt, full.UpdatedAt)
	}

	bare := entries[1]
	if bare.ExampleJapanese != nil || bare.ExampleFurigana != nil || bare.ExampleEnglish != nil ||
		bare.SourceSong != nil || bare.SourceArtist != nil || bare.SourceLyricsLine != nil || bare.LastAttemptedAt != nil {
		t.Errorf("optional fields should be nil: %+v", bare)
	}
	if bare.Tags == nil || len(bare.Tags) != 0 {
		t.Errorf("Tags = %#v, want an empty, non-nil slice", bare.Tags)
	}
}

func TestPostgresDictionaryRepository_List_RejectsUnknownSort(t *testing.T) {
	repo, _ := seed(t)
	_, _, err := repo.List(context.Background(), repository.DictionaryListParams{Sort: "id; DROP TABLE x", Limit: 10})
	if err == nil {
		t.Fatal("expected an error")
	}
}
