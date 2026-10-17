-- +goose Up
CREATE TABLE notes (
    id uuid PRIMARY KEY,
    created_at timestamptz NOT NULL,
    title text NOT NULL,
    body text NOT NULL
);

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';
-- squawk-ignore ban-drop-table
DROP TABLE notes;
