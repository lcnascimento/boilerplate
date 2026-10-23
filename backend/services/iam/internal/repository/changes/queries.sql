-- name: LockChanges :many
SELECT
    id,
    transaction_id::text AS transaction_id,
    occurred_at,
    table_name,
    operation,
    fields,
    actor_id,
    actor_organization_id,
    ip_hash
FROM identity_change
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: DeleteChanges :exec
DELETE FROM identity_change
WHERE id = ANY(sqlc.arg(ids)::bigint []);
