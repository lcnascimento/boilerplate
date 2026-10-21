-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

ALTER TABLE audit_entries ADD COLUMN actor_id text, ADD COLUMN actor_organization_id text;

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

-- squawk-ignore ban-drop-column
ALTER TABLE audit_entries DROP COLUMN actor_id, DROP COLUMN actor_organization_id;
