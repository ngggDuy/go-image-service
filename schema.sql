CREATE TABLE IF NOT EXISTS uploads (
    id                  text        PRIMARY KEY, 
    original_filename   text        NOT NULL, 
    ext                 text        NOT NULL, 
    status              text        NOT NULL, 
    created_at          timestamptz NOT NULL DEFAULT now()
);