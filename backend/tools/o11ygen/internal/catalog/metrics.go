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
	SlothRecording
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

// The series Sloth records for every SLO, read by the SLO panels of the dashboards.
const (
	SLOErrorRatioRate5m  = "slo:sli_error:ratio_rate5m"
	SLOErrorRatioRate30m = "slo:sli_error:ratio_rate30m"
	SLOErrorRatioRate1h  = "slo:sli_error:ratio_rate1h"
	SLOErrorRatioRate2h  = "slo:sli_error:ratio_rate2h"
	SLOErrorRatioRate6h  = "slo:sli_error:ratio_rate6h"
	SLOErrorRatioRate1d  = "slo:sli_error:ratio_rate1d"
	SLOErrorRatioRate3d  = "slo:sli_error:ratio_rate3d"
	SLOErrorRatioRate30d = "slo:sli_error:ratio_rate30d"

	SLOObjectiveRatio                  = "slo:objective:ratio"
	SLOErrorBudgetRatio                = "slo:error_budget:ratio"
	SLOTimePeriodDays                  = "slo:time_period:days"
	SLOCurrentBurnRateRatio            = "slo:current_burn_rate:ratio"
	SLOPeriodBurnRateRatio             = "slo:period_burn_rate:ratio"
	SLOPeriodErrorBudgetRemainingRatio = "slo:period_error_budget_remaining:ratio"
	SlothSLOInfo                       = "sloth_slo_info"
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

// Sloth records a series only for the services that declare an SLO, and the error ratio only while the SLI has events.
var (
	sloErrorRatio = Metric{Type: Gauge, Source: SlothRecording, Labels: []string{"sloth_window"}, Presence: PresenceOnEvent}
	sloMeta       = Metric{Type: Gauge, Source: SlothRecording, Presence: PresenceOnEvent}
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

	SLOErrorRatioRate5m:  sloErrorRatio,
	SLOErrorRatioRate30m: sloErrorRatio,
	SLOErrorRatioRate1h:  sloErrorRatio,
	SLOErrorRatioRate2h:  sloErrorRatio,
	SLOErrorRatioRate6h:  sloErrorRatio,
	SLOErrorRatioRate1d:  sloErrorRatio,
	SLOErrorRatioRate3d:  sloErrorRatio,
	SLOErrorRatioRate30d: sloErrorRatio,

	SLOObjectiveRatio:                  sloMeta,
	SLOErrorBudgetRatio:                sloMeta,
	SLOTimePeriodDays:                  sloMeta,
	SLOCurrentBurnRateRatio:            sloMeta,
	SLOPeriodBurnRateRatio:             sloMeta,
	SLOPeriodErrorBudgetRemainingRatio: sloMeta,
	SlothSLOInfo: {
		Type: Gauge, Source: SlothRecording, Presence: PresenceOnEvent,
		Labels: []string{"sloth_mode", "sloth_objective", "sloth_spec", "sloth_version"},
	},

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
	// service is the process that emits the SLI.
	SlothRecording: {"sloth_id", "sloth_service", "sloth_slo", "service"},
}

var (
	targetLabels = []string{"__name__", "job", "instance"}
	logLabels    = []string{"service_name", "service_namespace", "container", "detected_level", "severity_number"}
)

func OutboxOldestPendingAge(service string) string {
	return strings.ReplaceAll(service, "-", "_") + "_outbox_oldest_pending_age_seconds"
}
