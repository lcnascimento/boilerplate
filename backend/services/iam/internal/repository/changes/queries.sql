-- name: LockChanges :many
SELECT id, transaction_id::text AS transaction_id, occurred_at, table_name, operation, fields
FROM identity_change
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: DeleteChanges :exec
DELETE FROM identity_change
WHERE id = ANY(sqlc.arg(ids)::bigint []);

-- name: OldestChangeAge :one
SELECT coalesce(extract(epoch FROM now() - min(occurred_at)), 0)::float8 AS seconds
FROM identity_change;
