-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

CREATE TRIGGER session_identity_change
AFTER INSERT OR DELETE ON session
FOR EACH ROW EXECUTE FUNCTION record_identity_change('id', 'userId', 'activeOrganizationId');

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

DROP TRIGGER session_identity_change ON session;
