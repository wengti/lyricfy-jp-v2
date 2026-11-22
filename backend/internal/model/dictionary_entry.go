package model

import "time"

// DictionaryEntry is one word in the vocabulary list, with its quiz results.
type DictionaryEntry struct {
	ID                 string            `json:"id"`
	JapaneseText       string            `json:"japaneseText"`
	Hiragana           string            `json:"hiragana"`
	EnglishTranslation string            `json:"englishTranslation"`
	ExampleJapanese    *string           `json:"exampleJapanese"`
	ExampleFurigana    []FuriganaSegment `json:"exampleFurigana"` // nil when there's no reading
	ExampleEnglish     *string           `json:"exampleEnglish"`
	SourceSong         *string           `json:"sourceSong"`
	SourceArtist       *string           `json:"sourceArtist"`
	SourceLyricsLine   *string           `json:"sourceLyricsLine"`
	Tags               []string          `json:"tags"`
	AttemptCount       int               `json:"attemptCount"`
	SuccessCount       int               `json:"successCount"`
	LastAttemptedAt    *time.Time        `json:"lastAttemptedAt"` // nil until first quizzed
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

// FuriganaSegment is one piece of an example sentence. Reading is nil for kana and
// punctuation, which need none. The JSON names match what v1 stored in example_furigana.
type FuriganaSegment struct {
	Original string  `json:"original"`
	Reading  *string `json:"reading"`
}
