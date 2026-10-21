-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

ALTER TABLE notes VALIDATE CONSTRAINT notes_organization_id_not_null;

ALTER TABLE notes ALTER COLUMN organization_id SET NOT NULL;

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

-- squawk-ignore ban-drop-not-null
ALTER TABLE notes ALTER COLUMN organization_id DROP NOT NULL;
