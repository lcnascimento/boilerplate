package catalog

import (
	"slices"
	"strings"
)

type Type int

const (
	Counter Type = iota
	Gauge
	Histogram
)

type Presence int

const (
	PresenceAlways Presence = iota
	PresenceOnEvent
)

type Source int

const (
	OTLP Source = iota
	OTLPResource
	Scrape
)

type Metric struct {
	Type     Type
	Source   Source
	Labels   []string
	Presence Presence
}

const (
	TargetInfo = "target_info"

	PgxpoolAcquiredConnections = "pgxpool_acquired_connections"
	PgxpoolMaxConnections      = "pgxpool_max_connections"

	PgUp                     = "pg_up"
	PgSettingsMaxConnections = "pg_settings_max_connections"
	PgStatActivityCount      = "pg_stat_activity_count"

	KafkaBrokers = "kafka_brokers"

	HTTPServerRequestTotal    = "http_server_request_total"
	HTTPServerRequestDuration = "http_server_request_duration_seconds"
	HTTPClientRequestTotal    = "http_client_request_total"
	HTTPClientRequestDuration = "http_client_request_duration_seconds"
	HTTPClientActiveRequests  = "http_client_active_requests"

	MessagingClientConsumedMessages  = "messaging_client_consumed_messages_total"
	MessagingProcessDuration         = "messaging_process_duration_seconds"
	MessagingDLQSentMessages         = "messaging_dlq_sent_messages_total"
	MessagingClientSentMessages      = "messaging_client_sent_messages_total"
	MessagingClientOperationDuration = "messaging_client_operation_duration_seconds"

	KafkaTopicPartitionCurrentOffset     = "kafka_topic_partition_current_offset"
	KafkaConsumerGroupCurrentOffset      = "kafka_consumergroup_current_offset"
	KafkaConsumerGroupUncommittedOffsets = "kafka_consumergroup_uncommitted_offsets"

	// example:begin

	NotesOutboxOldestPendingAge = "notes_outbox_oldest_pending_age_seconds"
	NotesOutboxPurgedMessages   = "notes_outbox_purged_messages_total"
	NotesOutboxPurgeRuns        = "notes_outbox_purge_runs_total"
	// example:end
)

var (
	httpServerLabels = []string{"http_request_method", "http_route", "http_response_status_code"}
	httpClientLabels = []string{"network_peer_address", "http_request_method"}
	httpClientServer = []string{"server_address", "server_port"}
	messagingSend    = []string{
		"messaging_system", "messaging_destination_name", "messaging_operation_name", "messaging_operation_type", "error_type",
	}
	messagingProcess   = append([]string{"messaging_consumer_group_name"}, messagingSend...)
	pgxpool            = []string{"db_system_name", "db_client_connection_pool_name"}
	kafkaPartition     = []string{"topic", "partition"}
	kafkaConsumerGroup = []string{"consumergroup", "topic", "partition"}
)

var metrics = map[string]Metric{
	TargetInfo: {Type: Gauge, Source: OTLPResource, Labels: []string{"service_version", "deployment_environment"}},

	PgxpoolAcquiredConnections: {Type: Gauge, Source: OTLP, Labels: pgxpool},
	PgxpoolMaxConnections:      {Type: Gauge, Source: OTLP, Labels: pgxpool},

	PgUp:                     {Type: Gauge, Source: Scrape},
	PgSettingsMaxConnections: {Type: Gauge, Source: Scrape},
	PgStatActivityCount: {Type: Gauge, Source: Scrape, Labels: []string{
		"datname", "state", "usename", "application_name", "backend_type",
	}},

	KafkaBrokers: {Type: Gauge, Source: Scrape},

	HTTPServerRequestTotal:    {Type: Counter, Source: OTLP, Labels: httpServerLabels},
	HTTPServerRequestDuration: {Type: Histogram, Source: OTLP, Labels: append([]string{"url_scheme"}, httpServerLabels...)},
	HTTPClientRequestTotal: {
		Type: Counter, Source: OTLP, Presence: PresenceOnEvent,
		Labels: append([]string{"http_response_status_code"}, httpClientLabels...),
	},
	HTTPClientRequestDuration: {
		Type: Histogram, Source: OTLP, Presence: PresenceOnEvent,
		Labels: slices.Concat(httpClientServer, httpClientLabels, []string{"http_response_status_code"}),
	},
	HTTPClientActiveRequests: {
		Type: Gauge, Source: OTLP, Presence: PresenceOnEvent,
		Labels: slices.Concat(httpClientServer, httpClientLabels),
	},

	MessagingClientConsumedMessages:  {Type: Counter, Source: OTLP, Labels: messagingProcess},
	MessagingProcessDuration:         {Type: Histogram, Source: OTLP, Labels: messagingProcess},
	MessagingDLQSentMessages:         {Type: Counter, Source: OTLP, Labels: messagingProcess, Presence: PresenceOnEvent},
	MessagingClientSentMessages:      {Type: Counter, Source: OTLP, Labels: messagingSend},
	MessagingClientOperationDuration: {Type: Histogram, Source: OTLP, Labels: messagingSend},

	KafkaTopicPartitionCurrentOffset:     {Type: Gauge, Source: Scrape, Labels: kafkaPartition},
	KafkaConsumerGroupCurrentOffset:      {Type: Gauge, Source: Scrape, Labels: kafkaConsumerGroup},
	KafkaConsumerGroupUncommittedOffsets: {Type: Gauge, Source: Scrape, Labels: kafkaConsumerGroup},

	// example:begin
	NotesOutboxOldestPendingAge: {Type: Gauge, Source: OTLP},
	NotesOutboxPurgedMessages:   {Type: Counter, Source: OTLP},
	NotesOutboxPurgeRuns:        {Type: Counter, Source: OTLP, Labels: []string{"outcome"}},
	// example:end
}

var sourceLabels = map[Source][]string{
	OTLP:         {"job", "instance", "service_name", "service_namespace"},
	OTLPResource: {"job", "instance"},
	Scrape:       {"job", "instance"},
}

var (
	targetLabels = []string{"__name__", "job", "instance"}
	logLabels    = []string{"service_name", "service_namespace", "container", "detected_level", "severity_number"}
)

func OutboxOldestPendingAge(service string) string {
	return strings.ReplaceAll(service, "-", "_") + "_outbox_oldest_pending_age_seconds"
}
