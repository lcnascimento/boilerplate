-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

CREATE TRIGGER organization_identity_change
AFTER INSERT OR DELETE ON organization
FOR EACH ROW EXECUTE FUNCTION record_identity_change('id');

CREATE TRIGGER organization_rename
AFTER UPDATE OF name ON organization
FOR EACH ROW WHEN (OLD.name IS DISTINCT FROM NEW.name)
EXECUTE FUNCTION record_identity_change('id');

CREATE TRIGGER member_identity_change
AFTER INSERT OR DELETE ON member
FOR EACH ROW EXECUTE FUNCTION record_identity_change('id', 'organizationId', 'userId', 'role');

CREATE TRIGGER member_role_change
AFTER UPDATE OF role ON member
FOR EACH ROW WHEN (OLD.role IS DISTINCT FROM NEW.role)
EXECUTE FUNCTION record_identity_change('id', 'organizationId', 'userId', 'role');

CREATE TRIGGER invitation_identity_change
AFTER INSERT ON invitation
FOR EACH ROW EXECUTE FUNCTION record_identity_change('id', 'organizationId', 'inviterId', 'role');

CREATE TRIGGER invitation_answer
AFTER UPDATE OF status ON invitation
FOR EACH ROW WHEN (
    OLD.status IS DISTINCT FROM NEW.status
    AND NEW.status IN ('accepted', 'rejected', 'canceled')
)
EXECUTE FUNCTION record_identity_change('id', 'organizationId', 'status');

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

DROP TRIGGER invitation_answer ON invitation;
DROP TRIGGER invitation_identity_change ON invitation;
DROP TRIGGER member_role_change ON member;
DROP TRIGGER member_identity_change ON member;
DROP TRIGGER organization_rename ON organization;
DROP TRIGGER organization_identity_change ON organization;
