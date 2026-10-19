-- name: InsertAuditEntry :exec
INSERT INTO audit_entries (
    source, id, type, spec_version, subject, time, data_content_type,
    headers, payload, kafka_topic, kafka_partition, kafka_offset
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (source, id) DO NOTHING;
