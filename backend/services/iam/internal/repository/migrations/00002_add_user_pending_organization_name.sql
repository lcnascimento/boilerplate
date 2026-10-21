-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

ALTER TABLE "user" ADD COLUMN "pendingOrganizationName" text;

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';
-- squawk-ignore ban-drop-column
ALTER TABLE "user" DROP COLUMN "pendingOrganizationName";
