-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

DELETE FROM notes;

-- squawk-ignore adding-required-field
ALTER TABLE notes ADD COLUMN organization_id uuid NOT NULL;

-- squawk-ignore require-concurrent-index-creation
CREATE INDEX notes_organization_id_created_at_id_idx ON notes (organization_id, created_at DESC, id DESC);

ALTER TABLE notes ENABLE ROW LEVEL SECURITY;
ALTER TABLE notes FORCE ROW LEVEL SECURITY;

CREATE POLICY notes_organization_isolation ON notes
USING (organization_id = nullif(current_setting('app.organization_id', true), '')::uuid);

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

DROP POLICY notes_organization_isolation ON notes;

ALTER TABLE notes NO FORCE ROW LEVEL SECURITY;
ALTER TABLE notes DISABLE ROW LEVEL SECURITY;

-- squawk-ignore ban-drop-column
ALTER TABLE notes DROP COLUMN organization_id;
