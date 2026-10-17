-- +goose Up
CREATE TABLE audit_entries (
    source text NOT NULL,
    id text NOT NULL,
    type text NOT NULL,
    spec_version text NOT NULL,
    subject text,
    time timestamptz NOT NULL,
    data_content_type text NOT NULL,
    headers jsonb NOT NULL,
    payload bytea NOT NULL,
    kafka_topic text NOT NULL,
    -- Kafka partitions are int32 in the protocol.
    -- squawk-ignore prefer-bigint-over-int
    kafka_partition integer NOT NULL,
    kafka_offset bigint NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source, id)
);

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';
-- squawk-ignore ban-drop-table
DROP TABLE audit_entries;
