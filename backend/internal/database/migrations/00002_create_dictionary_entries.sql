-- +goose Up

-- The vocabulary list, carried over from v1 (Supabase) minus user_id: there is one user, who
-- still lives in the config. Rows imported from v1 keep their IDs and timestamps.
CREATE TABLE dictionary_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    japanese_text text NOT NULL,
    hiragana text NOT NULL,
    english_translation text NOT NULL,
    -- An optional example sentence; example_furigana holds its readings as v1 generated them
    example_japanese text,
    example_furigana jsonb,
    example_english text,
    -- The lyric the word was found in, if any
    source_song text,
    source_artist text,
    source_lyrics_line text,
    tags text[] NOT NULL DEFAULT '{}',
    -- Quiz results. v1 kept these in a separate word_stats table, one row per user and word;
    -- with a single user that's one row per entry, so they live here
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    success_count integer NOT NULL DEFAULT 0 CHECK (success_count BETWEEN 0 AND attempt_count),
    last_attempted_at timestamptz, -- NULL until first quizzed
    -- Set by the application (no trigger, unlike v1)
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

-- The dictionary page lists newest or oldest first, a page at a time
CREATE INDEX dictionary_entries_created_at_idx ON dictionary_entries (created_at);

-- +goose Down

DROP TABLE dictionary_entries;
