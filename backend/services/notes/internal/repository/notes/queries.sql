-- name: SetOrganization :exec
SELECT set_config('app.organization_id', sqlc.arg(organization_id)::text, true);

-- name: CreateNote :exec
INSERT INTO notes (id, organization_id, created_at, title, body)
VALUES ($1, $2, $3, $4, $5);

-- name: GetNote :one
SELECT id, created_at, title, body, organization_id
FROM notes
WHERE organization_id = $1 AND id = $2;

-- name: ListNotes :many
SELECT id, created_at, title, body, organization_id
FROM notes
WHERE organization_id = $1
ORDER BY created_at DESC, id DESC;

-- name: DeleteNote :execrows
DELETE FROM notes
WHERE organization_id = $1 AND id = $2;

-- name: DeleteOrganizationNotes :execrows
DELETE FROM notes
WHERE organization_id = $1;

-- name: ShareOrganizationLock :exec
SELECT pg_advisory_xact_lock_shared(hashtextextended(sqlc.arg(organization_id)::text, 0));

-- name: TakeOrganizationLock :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(organization_id)::text, 0));

-- name: OrganizationDeleted :one
SELECT EXISTS (SELECT 1 FROM deleted_organizations WHERE organization_id = $1) AS deleted;

-- name: MarkOrganizationDeleted :exec
INSERT INTO deleted_organizations (organization_id) VALUES ($1)
ON CONFLICT (organization_id) DO NOTHING;
