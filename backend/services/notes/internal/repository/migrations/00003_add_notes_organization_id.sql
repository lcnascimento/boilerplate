-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

DELETE FROM notes;

ALTER TABLE notes ADD COLUMN organization_id uuid;

ALTER TABLE notes ADD CONSTRAINT notes_organization_id_not_null CHECK (organization_id IS NOT NULL) NOT VALID;

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

-- squawk-ignore ban-drop-constraint
ALTER TABLE notes DROP CONSTRAINT notes_organization_id_not_null;

-- squawk-ignore ban-drop-column
ALTER TABLE notes DROP COLUMN organization_id;
