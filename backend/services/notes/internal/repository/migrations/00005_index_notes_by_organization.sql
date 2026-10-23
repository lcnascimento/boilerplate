-- +goose NO TRANSACTION

-- +goose Up
SET lock_timeout = '1s';
SET statement_timeout = '5s';

CREATE INDEX CONCURRENTLY IF NOT EXISTS notes_organization_id_created_at_id_idx
ON notes (organization_id, created_at DESC, id DESC);

RESET statement_timeout;
RESET lock_timeout;

-- +goose Down
SET lock_timeout = '1s';
SET statement_timeout = '5s';

DROP INDEX CONCURRENTLY IF EXISTS notes_organization_id_created_at_id_idx;

RESET statement_timeout;
RESET lock_timeout;
