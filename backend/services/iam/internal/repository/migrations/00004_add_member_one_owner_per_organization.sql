-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

ALTER TABLE member
    -- Deferred so a transfer can promote the new Owner before demoting the old one in the same transaction.
    -- An EXCLUDE constraint cannot be NOT VALID, and member holds a few rows per Organization.
    -- squawk-ignore constraint-missing-not-valid
    ADD CONSTRAINT member_one_owner_per_organization
    EXCLUDE USING btree ("organizationId" WITH =) WHERE (role = 'owner')
    DEFERRABLE INITIALLY DEFERRED;

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';
-- squawk-ignore ban-drop-constraint
ALTER TABLE member DROP CONSTRAINT member_one_owner_per_organization;
