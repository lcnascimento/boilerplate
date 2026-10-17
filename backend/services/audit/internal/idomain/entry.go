package idomain

import "time"

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
