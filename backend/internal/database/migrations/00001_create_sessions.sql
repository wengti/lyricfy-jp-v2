-- +goose Up

-- One row per login. The browser holds a random token; only its SHA-256 hash is stored, so a
-- leaked table can't be used to log in. A session is valid while its row exists and hasn't
-- expired; logging out deletes the row. See docs/session-tokens.md.
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    -- The single user still lives in the config; this becomes a user ID once users move into
    -- the database
    user_email text NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);

-- Every login deletes the expired rows
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- +goose Down

DROP TABLE sessions;
