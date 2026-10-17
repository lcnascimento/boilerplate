package events_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

func TestEnvelope_Headers(t *testing.T) {
	t.Parallel()

	envelope := events.Envelope{
		ID:              "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
		Source:          "notes",
		Type:            "acme.notes.v1.NoteCreated",
		Time:            time.Date(2026, 10, 17, 12, 30, 45, 123456789, time.UTC),
		Subject:         "note_0199f1a2-0000-7000-8000-000000000001",
		PartitionKey:    "note_0199f1a2-0000-7000-8000-000000000001",
		SpecVersion:     events.SpecVersion,
		DataContentType: events.ContentType,
	}

	require.Equal(t, map[string]string{
		"ce_id":           "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
		"ce_source":       "notes",
		"ce_type":         "acme.notes.v1.NoteCreated",
		"ce_time":         "2026-10-17T12:30:45.123456789Z",
		"ce_subject":      "note_0199f1a2-0000-7000-8000-000000000001",
		"ce_partitionkey": "note_0199f1a2-0000-7000-8000-000000000001",
		"ce_specversion":  "1.0",
		"content-type":    "application/protobuf",
	}, envelope.Headers())
}

func TestParseEnvelope(t *testing.T) {
	t.Parallel()

	full := map[string]string{
		"ce_id":           "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
		"ce_source":       "notes",
		"ce_type":         "acme.notes.v1.NoteDeleted",
		"ce_time":         "2026-10-17T12:30:45.5-03:00",
		"ce_subject":      "note_1",
		"ce_partitionkey": "note_1",
		"ce_specversion":  "1.0",
		"content-type":    "application/protobuf",
		"traceparent":     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
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
		want    events.Envelope
		wantErr error
	}{
		{
			name:    "all headers",
			headers: full,
			want: events.Envelope{
				ID:              "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
				Source:          "notes",
				Type:            "acme.notes.v1.NoteDeleted",
				Time:            time.Date(2026, 10, 17, 15, 30, 45, 500000000, time.UTC),
				Subject:         "note_1",
				PartitionKey:    "note_1",
				SpecVersion:     "1.0",
				DataContentType: "application/protobuf",
			},
		},
		{
			name: "only required headers",
			headers: map[string]string{
				"ce_id":          "1",
				"ce_source":      "notes",
				"ce_type":        "acme.notes.v1.NoteDeleted",
				"ce_specversion": "1.0",
			},
			want: events.Envelope{ID: "1", Source: "notes", Type: "acme.notes.v1.NoteDeleted", SpecVersion: "1.0"},
		},
		{name: "missing ce_id", headers: without("ce_id"), wantErr: events.ErrMissingHeader},
		{name: "missing ce_source", headers: without("ce_source"), wantErr: events.ErrMissingHeader},
		{name: "missing ce_type", headers: without("ce_type"), wantErr: events.ErrMissingHeader},
		{name: "missing ce_specversion", headers: without("ce_specversion"), wantErr: events.ErrMissingHeader},
		{
			name:    "empty ce_id",
			headers: map[string]string{"ce_id": "", "ce_source": "notes", "ce_type": "t", "ce_specversion": "1.0"},
			wantErr: events.ErrMissingHeader,
		},
		{
			name:    "malformed ce_time",
			headers: map[string]string{"ce_id": "1", "ce_source": "notes", "ce_type": "t", "ce_specversion": "1.0", "ce_time": "yesterday"},
			wantErr: events.ErrInvalidHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := events.ParseEnvelope(tt.headers)

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

	_, err := events.ParseEnvelope(map[string]string{"ce_id": "1", "ce_source": "notes", "ce_specversion": "1.0"})

	require.Equal(t, "ce_type", errors.Attributes(err)["header"])
}

func TestEnvelope_RoundTrip(t *testing.T) {
	t.Parallel()

	envelope := events.Envelope{
		ID:              "0199f1a2-7c3e-7d4a-9b1c-2e3f4a5b6c7d",
		Source:          "notes",
		Type:            "acme.notes.v1.NoteCreated",
		Time:            time.Date(2026, 10, 17, 12, 30, 45, 123456789, time.UTC),
		Subject:         "note_1",
		PartitionKey:    "note_1",
		SpecVersion:     events.SpecVersion,
		DataContentType: events.ContentType,
	}

	got, err := events.ParseEnvelope(envelope.Headers())

	require.NoError(t, err)
	require.Equal(t, envelope, got)
}
