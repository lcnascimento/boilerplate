package eventbus_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
)

func TestEnvelope_Headers(t *testing.T) {
	t.Parallel()

	envelope := eventbus.Envelope{
		ID:              "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
		Source:          "iam",
		Type:            "acme.iam.v1.UserCreated",
		Time:            time.Date(2026, 10, 17, 12, 30, 45, 123456789, time.UTC),
		Subject:         "user_0199f1a2-0000-7000-8000-000000000001",
		PartitionKey:    "user_0199f1a2-0000-7000-8000-000000000001",
		SpecVersion:     eventbus.SpecVersion,
		DataContentType: eventbus.ContentType,
		Origin: eventbus.Origin{
			ActorID:             "user_0199f1a2-0000-7000-8000-0000000000a1",
			ActorOrganizationID: "org_0199f1a2-0000-7000-8000-0000000000b2",
			IPHash:              "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		},
		CorrelationID: "784512",
	}

	require.Equal(t, map[string]string{
		"ce_id":                  "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
		"ce_source":              "iam",
		"ce_type":                "acme.iam.v1.UserCreated",
		"ce_time":                "2026-10-17T12:30:45.123456789Z",
		"ce_subject":             "user_0199f1a2-0000-7000-8000-000000000001",
		"ce_partitionkey":        "user_0199f1a2-0000-7000-8000-000000000001",
		"ce_specversion":         "1.0",
		"content-type":           "application/protobuf",
		"ce_actorid":             "user_0199f1a2-0000-7000-8000-0000000000a1",
		"ce_actororganizationid": "org_0199f1a2-0000-7000-8000-0000000000b2",
		"ce_iphash":              "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		"ce_correlationid":       "784512",
	}, envelope.Headers())
}

func TestParseEnvelope(t *testing.T) {
	t.Parallel()

	full := map[string]string{
		"ce_id":                  "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
		"ce_source":              "iam",
		"ce_type":                "acme.iam.v1.UserDeleted",
		"ce_time":                "2026-10-17T12:30:45.5-03:00",
		"ce_subject":             "user_1",
		"ce_partitionkey":        "user_1",
		"ce_specversion":         "1.0",
		"content-type":           "application/protobuf",
		"traceparent":            "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"ce_actorid":             "user_1",
		"ce_actororganizationid": "org_1",
		"ce_iphash":              "ip_1",
		"ce_correlationid":       "42",
	}

	without := func(key string) map[string]string {
		headers := map[string]string{}

		for k, v := range full {
			if k != key {
				headers[k] = v
			}
		}

		return headers
	}

	tests := []struct {
		name    string
		headers map[string]string
		want    eventbus.Envelope
		wantErr error
	}{
		{
			name:    "all headers",
			headers: full,
			want: eventbus.Envelope{
				ID:              "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
				Source:          "iam",
				Type:            "acme.iam.v1.UserDeleted",
				Time:            time.Date(2026, 10, 17, 15, 30, 45, 500000000, time.UTC),
				Subject:         "user_1",
				PartitionKey:    "user_1",
				SpecVersion:     "1.0",
				DataContentType: "application/protobuf",
				Origin: eventbus.Origin{
					ActorID:             "user_1",
					ActorOrganizationID: "org_1",
					IPHash:              "ip_1",
				},
				CorrelationID: "42",
			},
		},
		{
			name: "only required headers",
			headers: map[string]string{
				"ce_id":          "1",
				"ce_source":      "iam",
				"ce_type":        "acme.iam.v1.UserDeleted",
				"ce_specversion": "1.0",
			},
			want: eventbus.Envelope{ID: "1", Source: "iam", Type: "acme.iam.v1.UserDeleted", SpecVersion: "1.0"},
		},
		{name: "missing ce_id", headers: without("ce_id"), wantErr: eventbus.ErrMissingHeader},
		{name: "missing ce_source", headers: without("ce_source"), wantErr: eventbus.ErrMissingHeader},
		{name: "missing ce_type", headers: without("ce_type"), wantErr: eventbus.ErrMissingHeader},
		{name: "missing ce_specversion", headers: without("ce_specversion"), wantErr: eventbus.ErrMissingHeader},
		{
			name:    "empty ce_id",
			headers: map[string]string{"ce_id": "", "ce_source": "iam", "ce_type": "t", "ce_specversion": "1.0"},
			wantErr: eventbus.ErrMissingHeader,
		},
		{
			name:    "malformed ce_time",
			headers: map[string]string{"ce_id": "1", "ce_source": "iam", "ce_type": "t", "ce_specversion": "1.0", "ce_time": "yesterday"},
			wantErr: eventbus.ErrInvalidHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := eventbus.ParseEnvelope(tt.headers)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			require.True(t, tt.want.Time.Equal(got.Time), "time: want %s, got %s", tt.want.Time, got.Time)

			tt.want.Time, got.Time = time.Time{}, time.Time{}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseEnvelope_NamesTheMissingHeader(t *testing.T) {
	t.Parallel()

	_, err := eventbus.ParseEnvelope(map[string]string{"ce_id": "1", "ce_source": "iam", "ce_specversion": "1.0"})

	require.Equal(t, "ce_type", errors.Attributes(err)["header"])
}

func TestEnvelope_RoundTrip(t *testing.T) {
	t.Parallel()

	envelope := eventbus.Envelope{
		ID:              "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
		Source:          "iam",
		Type:            "acme.iam.v1.UserCreated",
		Time:            time.Date(2026, 10, 17, 12, 30, 45, 123456789, time.UTC),
		Subject:         "user_1",
		PartitionKey:    "user_1",
		SpecVersion:     eventbus.SpecVersion,
		DataContentType: eventbus.ContentType,
		Origin: eventbus.Origin{
			ActorID:             "user_1",
			ActorOrganizationID: "org_1",
			IPHash:              "ip_1",
		},
		CorrelationID: "42",
	}

	got, err := eventbus.ParseEnvelope(envelope.Headers())

	require.NoError(t, err)
	require.Equal(t, envelope, got)
}
