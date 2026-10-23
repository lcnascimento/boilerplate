-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

CREATE TABLE identity_change (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transaction_id xid8 NOT NULL DEFAULT pg_current_xact_id(),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    table_name text NOT NULL,
    operation text NOT NULL,
    fields jsonb NOT NULL
);

-- +goose StatementBegin
CREATE FUNCTION record_identity_change() RETURNS trigger
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

CREATE TRIGGER user_identity_change
AFTER INSERT OR DELETE ON "user"
FOR EACH ROW EXECUTE FUNCTION record_identity_change('id');

CREATE TRIGGER account_password_change
AFTER UPDATE OF password ON account
FOR EACH ROW WHEN (OLD.password IS DISTINCT FROM NEW.password)
EXECUTE FUNCTION record_identity_change('userId');

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

DROP TRIGGER account_password_change ON account;
DROP TRIGGER user_identity_change ON "user";
-- squawk-ignore ban-drop-function
DROP FUNCTION record_identity_change();
-- squawk-ignore ban-drop-table
DROP TABLE identity_change;
