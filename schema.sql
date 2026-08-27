CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
    id                  text            PRIMARY KEY, 
    email               citext          NOT NULL UNIQUE,
    password_hash       text            NOT NULL, 
    created_at          timestamptz     NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS uploads (
    id                  text        PRIMARY KEY,
    original_filename   text        NOT NULL,
    content_type        text        NOT NULL,
    status              text        NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    user_id             text        NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_uploads_user_id ON uploads(user_id);

