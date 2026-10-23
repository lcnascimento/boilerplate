-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

ALTER TABLE identity_change
ADD COLUMN actor_id uuid,
ADD COLUMN actor_organization_id uuid,
ADD COLUMN ip_hash text;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION record_identity_change() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    changed_row jsonb := CASE TG_OP WHEN 'DELETE' THEN to_jsonb(OLD) ELSE to_jsonb(NEW) END;
    kept jsonb := '{}';
    field text;
BEGIN
    FOREACH field IN ARRAY TG_ARGV LOOP
        kept := kept || jsonb_build_object(field, changed_row -> field);
    END LOOP;

    INSERT INTO identity_change (table_name, operation, fields, actor_id, actor_organization_id, ip_hash)
    VALUES (
        TG_TABLE_NAME,
        TG_OP,
        kept,
        NULLIF(current_setting('app.actor_id', true), '')::uuid,
        NULLIF(current_setting('app.actor_organization_id', true), '')::uuid,
        NULLIF(current_setting('app.ip_hash', true), '')
    );

    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION record_identity_change() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    changed_row jsonb := CASE TG_OP WHEN 'DELETE' THEN to_jsonb(OLD) ELSE to_jsonb(NEW) END;
    kept jsonb := '{}';
    field text;
BEGIN
    FOREACH field IN ARRAY TG_ARGV LOOP
        kept := kept || jsonb_build_object(field, changed_row -> field);
    END LOOP;

    INSERT INTO identity_change (table_name, operation, fields) VALUES (TG_TABLE_NAME, TG_OP, kept);

    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- squawk-ignore ban-drop-column
ALTER TABLE identity_change DROP COLUMN ip_hash;
-- squawk-ignore ban-drop-column
ALTER TABLE identity_change DROP COLUMN actor_organization_id;
-- squawk-ignore ban-drop-column
ALTER TABLE identity_change DROP COLUMN actor_id;
