-- name: CreateNote :exec
INSERT INTO notes (id, created_at, title, body)
VALUES ($1, $2, $3, $4);

-- name: GetNote :one
SELECT id, created_at, title, body
FROM notes
WHERE id = $1;

-- name: ListNotes :many
SELECT id, created_at, title, body
FROM notes
ORDER BY created_at DESC, id DESC;

-- name: DeleteNote :execrows
DELETE FROM notes
WHERE id = $1;
