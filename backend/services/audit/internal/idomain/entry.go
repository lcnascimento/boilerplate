package idomain

import "time"

type Entry struct {
	Source          string            `json:"source"          validate:"required"`
	ID              string            `json:"id"              validate:"required"`
	Type            string            `json:"type"            validate:"required"`
	SpecVersion     string            `json:"specversion"     validate:"required"`
	Subject         string            `json:"subject"`
	Time            time.Time         `json:"time"            validate:"required"`
	DataContentType string            `json:"datacontenttype" validate:"required"`
	Headers         map[string]string `json:"headers"`
	Payload         []byte            `json:"payload"`
	KafkaTopic      string            `json:"kafka_topic"`
	KafkaPartition  int32             `json:"kafka_partition"`
	KafkaOffset     int64             `json:"kafka_offset"`

	ActorID             string `json:"actor_id"`
	ActorOrganizationID string `json:"actor_organization_id"`
}
