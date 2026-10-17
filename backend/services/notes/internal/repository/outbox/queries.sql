-- name: PurgeForwarded :execrows
DELETE FROM watermill_notes_outbox
WHERE (transaction_id, "offset") IN (
    SELECT m.transaction_id, m."offset"
    FROM watermill_notes_outbox m
    JOIN watermill_offsets_notes_outbox o ON o.consumer_group = @consumer_group
    WHERE m.created_at < now() - make_interval(secs => @retention_seconds::float8)
      AND (m.transaction_id, m."offset") <= (o.last_processed_transaction_id, o.offset_acked)
    LIMIT @batch_size
);

-- name: OldestPendingAgeSeconds :one
SELECT coalesce(extract(epoch FROM now() - min(m.created_at)), 0)::float8 AS age_seconds
FROM watermill_notes_outbox m
LEFT JOIN watermill_offsets_notes_outbox o ON o.consumer_group = @consumer_group
WHERE o.offset_acked IS NULL
   OR (m.transaction_id, m."offset") > (o.last_processed_transaction_id, o.offset_acked);
