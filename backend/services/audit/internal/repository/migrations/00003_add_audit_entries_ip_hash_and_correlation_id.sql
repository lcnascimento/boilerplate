-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

ALTER TABLE audit_entries ADD COLUMN ip_hash text, ADD COLUMN correlation_id text;

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

-- squawk-ignore ban-drop-column
ALTER TABLE audit_entries DROP COLUMN ip_hash, DROP COLUMN correlation_id;
