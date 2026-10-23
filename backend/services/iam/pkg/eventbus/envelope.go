package eventbus

import "time"

const (
	SpecVersion = "1.0"
	ContentType = "application/protobuf"
)

const (
	headerID           = "ce_id"
	headerSource       = "ce_source"
	headerType         = "ce_type"
	headerTime         = "ce_time"
	headerSubject      = "ce_subject"
	headerPartitionKey = "ce_partitionkey"
	headerSpecVersion  = "ce_specversion"
	headerContentType  = "content-type"

	headerActorID             = "ce_actorid"
	headerActorOrganizationID = "ce_actororganizationid"
	headerIPHash              = "ce_iphash"
	headerCorrelationID       = "ce_correlationid"
)

type Envelope struct {
	Origin

	ID              string
	Source          string
	Type            string
	Time            time.Time
	Subject         string
	PartitionKey    string
	SpecVersion     string
	DataContentType string
	CorrelationID   string
}

type Origin struct {
	ActorID             string
	ActorOrganizationID string
	IPHash              string
}

func (e *Envelope) Headers() map[string]string {
	headers := map[string]string{}

	set := func(key, value string) {
		if value != "" {
			headers[key] = value
		}
	}

	set(headerID, e.ID)
	set(headerSource, e.Source)
	set(headerType, e.Type)
	set(headerSubject, e.Subject)
	set(headerPartitionKey, e.PartitionKey)
	set(headerSpecVersion, e.SpecVersion)
	set(headerContentType, e.DataContentType)
	set(headerActorID, e.ActorID)
	set(headerActorOrganizationID, e.ActorOrganizationID)
	set(headerIPHash, e.IPHash)
	set(headerCorrelationID, e.CorrelationID)

	if !e.Time.IsZero() {
		headers[headerTime] = e.Time.Format(time.RFC3339Nano)
	}

	return headers
}

func ParseEnvelope(headers map[string]string) (Envelope, error) {
	for _, key := range []string{headerID, headerSource, headerType, headerSpecVersion} {
		if headers[key] == "" {
			return Envelope{}, ErrMissingHeader.WithAttribute("header", key)
		}
	}

	envelope := Envelope{
		ID:              headers[headerID],
		Source:          headers[headerSource],
		Type:            headers[headerType],
		Subject:         headers[headerSubject],
		PartitionKey:    headers[headerPartitionKey],
		SpecVersion:     headers[headerSpecVersion],
		DataContentType: headers[headerContentType],

		Origin: Origin{
			ActorID:             headers[headerActorID],
			ActorOrganizationID: headers[headerActorOrganizationID],
			IPHash:              headers[headerIPHash],
		},
		CorrelationID: headers[headerCorrelationID],
	}

	if raw := headers[headerTime]; raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return Envelope{}, ErrInvalidHeader.WithAttribute("header", headerTime).WithCause(err)
		}

		envelope.Time = parsed
	}

	return envelope, nil
}
