package catalog

import (
	"slices"
	"strings"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
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
	RecordingRule
	LokiRecordingRule
	SlothRecording
	TempoSpanMetrics
)

type Metric struct {
	Type      Type
	Source    Source
	Labels    []string
	Presence  Presence
	EmittedBy Emitter
}

const (
	TargetInfo = "target_info"

	PgxpoolAcquiredConnections = "pgxpool_acquired_connections"
	PgxpoolMaxConnections      = "pgxpool_max_connections"

	PgUp                     = "pg_up"
	PgSettingsMaxConnections = "pg_settings_max_connections"
	PgStatActivityCount      = "pg_stat_activity_count"

	PgStatActivityMaxTxDuration = "pg_stat_activity_max_tx_duration"
	PgClientConnections         = "pg_client_connections_count"
	PgLocksCount                = "pg_locks_count"
	PgDatabaseSizeBytes         = "pg_database_size_bytes"
	PgStatDatabaseBlksHit       = "pg_stat_database_blks_hit"
	PgStatDatabaseBlksRead      = "pg_stat_database_blks_read"
	PgStatDatabaseXactCommit    = "pg_stat_database_xact_commit"
	PgStatDatabaseXactRollback  = "pg_stat_database_xact_rollback"
	PgStatDatabaseDeadlocks     = "pg_stat_database_deadlocks"
	PgStatStatementsSeconds     = "pg_stat_statements_seconds_total"
	PgStatStatementsCalls       = "pg_stat_statements_calls_total"
	PgStatStatementsRows        = "pg_stat_statements_rows_total"

	KafkaBrokers = "kafka_brokers"

	JobSeriesCount = "job:series:count"

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

	ServiceLogRecords         = "service:log_records:rate1m"
	ServiceLogRecordsCritical = "service:log_records_critical:rate1m"
	ServiceLogAlertRecords    = "service:log_alert_records:rate1m"

	GoGoroutineCount  = "go_goroutine_count"
	GoMemoryUsed      = "go_memory_used_bytes"
	GoMemoryAllocated = "go_memory_allocated_bytes_total"
	GoMemoryGCGoal    = "go_memory_gc_goal_bytes"

	ProcessCPUTime     = "process_cpu_time_seconds_total"
	ProcessMemoryUsage = "process_memory_usage_bytes"

	NodejsEventloopDelayP99    = "nodejs_eventloop_delay_p99_seconds"
	NodejsEventloopUtilization = "nodejs_eventloop_utilization_ratio"
	V8jsMemoryHeapUsed         = "v8js_memory_heap_used_bytes"
	V8jsGCDuration             = "v8js_gc_duration_seconds"

	TracesSpanMetricsCalls   = "traces_spanmetrics_calls_total"
	TracesSpanMetricsLatency = "traces_spanmetrics_latency"
)

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
	pgActivity         = []string{"datname", "state", "server"}
	pgDatabase         = []string{"datid", "datname"}
	pgStatement        = []string{"datname", "user", "queryid"}
	kafkaPartition     = []string{"topic", "partition"}
	kafkaConsumerGroup = []string{"consumergroup", "topic", "partition"}
)

// Sloth records a series only for the services that declare an SLO, and the error ratio only while the SLI has events.
var (
	sloErrorRatio = Metric{Type: Gauge, Source: SlothRecording, Labels: []string{"sloth_window"}, Presence: PresenceOnEvent}
	sloMeta       = Metric{Type: Gauge, Source: SlothRecording, Presence: PresenceOnEvent}
)

var metrics = map[string]Metric{
	TargetInfo: {
		Type: Gauge, Source: OTLPResource, Labels: []string{"service_version", "deployment_environment"},
		EmittedBy: everyProcess,
	},

	PgxpoolAcquiredConnections: {Type: Gauge, Source: OTLP, Labels: pgxpool, EmittedBy: dbPool},
	PgxpoolMaxConnections:      {Type: Gauge, Source: OTLP, Labels: pgxpool, EmittedBy: dbPool},

	PgUp:                        {Type: Gauge, Source: Scrape},
	PgSettingsMaxConnections:    {Type: Gauge, Source: Scrape},
	PgStatActivityCount:         {Type: Gauge, Source: Scrape, Labels: pgActivity},
	PgStatActivityMaxTxDuration: {Type: Gauge, Source: Scrape, Labels: pgActivity},
	PgClientConnections: {
		Type: Gauge, Source: Scrape, Labels: append([]string{"usename", "application_name"}, pgActivity...),
	},
	PgLocksCount:               {Type: Gauge, Source: Scrape, Labels: []string{"datname", "mode"}},
	PgDatabaseSizeBytes:        {Type: Gauge, Source: Scrape, Labels: []string{"datname"}},
	PgStatDatabaseBlksHit:      {Type: Counter, Source: Scrape, Labels: pgDatabase},
	PgStatDatabaseBlksRead:     {Type: Counter, Source: Scrape, Labels: pgDatabase},
	PgStatDatabaseXactCommit:   {Type: Counter, Source: Scrape, Labels: pgDatabase},
	PgStatDatabaseXactRollback: {Type: Counter, Source: Scrape, Labels: pgDatabase},
	PgStatDatabaseDeadlocks:    {Type: Counter, Source: Scrape, Labels: pgDatabase},
	PgStatStatementsSeconds:    {Type: Counter, Source: Scrape, Labels: pgStatement},
	PgStatStatementsCalls:      {Type: Counter, Source: Scrape, Labels: pgStatement},
	PgStatStatementsRows:       {Type: Counter, Source: Scrape, Labels: pgStatement},

	KafkaBrokers: {Type: Gauge, Source: Scrape},

	JobSeriesCount: {Type: Gauge, Source: RecordingRule},

	HTTPServerRequestTotal: {
		Type: Counter, Source: OTLP, Labels: httpServerLabels, EmittedBy: on(service.Go, runs(service.API)),
	},
	HTTPServerRequestDuration: {
		Type: Histogram, Source: OTLP, Labels: append([]string{"url_scheme"}, httpServerLabels...),
		EmittedBy: on(service.Go, runs(service.API)),
	},
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

	MessagingClientConsumedMessages: {Type: Counter, Source: OTLP, Labels: messagingProcess, EmittedBy: runs(service.Consumer)},
	MessagingProcessDuration:        {Type: Histogram, Source: OTLP, Labels: messagingProcess, EmittedBy: runs(service.Consumer)},
	MessagingDLQSentMessages:        {Type: Counter, Source: OTLP, Labels: messagingProcess, Presence: PresenceOnEvent},
	MessagingClientSentMessages:     {Type: Counter, Source: OTLP, Labels: messagingSend, EmittedBy: runs(service.Worker)},
	MessagingClientOperationDuration: {
		Type: Histogram, Source: OTLP, Labels: messagingSend, EmittedBy: runs(service.Worker),
	},

	// The exporter lists only the topics a producer or a consumer created.
	KafkaTopicPartitionCurrentOffset: {
		Type: Gauge, Source: Scrape, Labels: kafkaPartition, EmittedBy: runs(service.Consumer, service.Worker),
	},
	KafkaConsumerGroupCurrentOffset: {
		Type: Gauge, Source: Scrape, Labels: kafkaConsumerGroup, EmittedBy: runs(service.Consumer),
	},
	KafkaConsumerGroupUncommittedOffsets: {
		Type: Gauge, Source: Scrape, Labels: kafkaConsumerGroup, EmittedBy: runs(service.Consumer),
	},

	ServiceLogRecords: {
		Type: Gauge, Source: LokiRecordingRule, Labels: []string{"service", "level"}, EmittedBy: everyProcess,
	},
	ServiceLogRecordsCritical: {Type: Gauge, Source: LokiRecordingRule, Labels: []string{"service"}, Presence: PresenceOnEvent},
	ServiceLogAlertRecords: {
		Type: Gauge, Source: LokiRecordingRule, Labels: []string{"service", "log_alert"}, Presence: PresenceOnEvent,
	},

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

	GoGoroutineCount:  {Type: Gauge, Source: OTLP, EmittedBy: on(service.Go, everyProcess)},
	GoMemoryUsed:      {Type: Gauge, Source: OTLP, Labels: []string{"go_memory_type"}, EmittedBy: on(service.Go, everyProcess)},
	GoMemoryAllocated: {Type: Counter, Source: OTLP, EmittedBy: on(service.Go, everyProcess)},
	GoMemoryGCGoal:    {Type: Gauge, Source: OTLP, EmittedBy: on(service.Go, everyProcess)},

	ProcessCPUTime: {
		Type: Counter, Source: OTLP, Labels: []string{"cpu_mode"}, EmittedBy: on(service.Go, everyProcess),
	},
	ProcessMemoryUsage: {Type: Gauge, Source: OTLP, EmittedBy: on(service.Go, everyProcess)},

	NodejsEventloopDelayP99:    {Type: Gauge, Source: OTLP, EmittedBy: on(service.Node, everyProcess)},
	NodejsEventloopUtilization: {Type: Gauge, Source: OTLP, EmittedBy: on(service.Node, everyProcess)},
	V8jsMemoryHeapUsed: {
		Type: Gauge, Source: OTLP, Labels: []string{"v8js_heap_space_name"}, EmittedBy: on(service.Node, everyProcess),
	},
	V8jsGCDuration: {
		Type: Histogram, Source: OTLP, Labels: []string{"v8js_gc_type"}, EmittedBy: on(service.Node, everyProcess),
	},

	TracesSpanMetricsCalls:   {Type: Counter, Source: TempoSpanMetrics, EmittedBy: runs(service.API)},
	TracesSpanMetricsLatency: {Type: Histogram, Source: TempoSpanMetrics, EmittedBy: runs(service.API)},
}

const (
	outboxOldestPendingAge = "oldest_pending_age_seconds"
	outboxPurgedMessages   = "purged_messages_total"
	outboxPurgeRuns        = "purge_runs_total"
)

var outboxMetrics = map[string]Metric{
	outboxOldestPendingAge: {Type: Gauge, Source: OTLP},
	outboxPurgedMessages:   {Type: Counter, Source: OTLP},
	outboxPurgeRuns:        {Type: Counter, Source: OTLP, Labels: []string{"outcome"}},
}

func lookup(series string) (Metric, bool) {
	if metric, ok := metrics[series]; ok {
		return metric, true
	}

	_, name, ok := strings.Cut(series, "_outbox_")
	metric, known := outboxMetrics[name]

	return metric, ok && known
}

var sourceLabels = map[Source][]string{
	OTLP:             {"job", "instance", "service_name", "service_namespace"},
	OTLPResource:     {"job", "instance"},
	Scrape:           {"job", "instance"},
	RecordingRule:    {"job"},
	SlothRecording:   {"sloth_id", "sloth_service", "sloth_slo", "service"},
	TempoSpanMetrics: {"service", "span_name", "span_kind", "status_code", "http_status_code", "source"},
}

var (
	targetLabels = []string{"__name__", "job", "instance"}
	logLabels    = []string{"service_name", "service_namespace", "container_name", "detected_level", "severity_number"}
)

func OutboxOldestPendingAge(serviceName string) string {
	return outboxMetric(serviceName, outboxOldestPendingAge)
}

func OutboxPurgedMessages(serviceName string) string {
	return outboxMetric(serviceName, outboxPurgedMessages)
}

func OutboxPurgeRuns(serviceName string) string {
	return outboxMetric(serviceName, outboxPurgeRuns)
}

func outboxMetric(serviceName, name string) string {
	return strings.ReplaceAll(serviceName, "-", "_") + "_outbox_" + name
}
