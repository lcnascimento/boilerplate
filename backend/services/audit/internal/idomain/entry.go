package idomain

import "time"

// Entry is an Audit Entry: one received Integration Event, its envelope and payload stored uninterpreted.
// Source and ID identify it, so a redelivered event is the same Entry.
type Entry struct {
	Source          string
	ID              string
	Type            string
	SpecVersion     string
	Subject         string
	Time            time.Time
	DataContentType string
	Headers         map[string]string
	Payload         []byte
	KafkaTopic      string
	KafkaPartition  int32
	KafkaOffset     int64
}
