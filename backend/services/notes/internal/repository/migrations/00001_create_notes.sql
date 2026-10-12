-- +goose Up
CREATE TABLE notes (
    id uuid PRIMARY KEY,
    created_at timestamptz NOT NULL,
    title text NOT NULL,
    body text NOT NULL
);
