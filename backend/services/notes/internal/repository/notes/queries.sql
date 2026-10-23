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
