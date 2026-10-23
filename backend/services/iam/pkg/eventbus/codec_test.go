package eventbus_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/internal/testevents"
)

func TestSchemaIDs_Encode(t *testing.T) {
	t.Parallel()

	ids := eventbus.SchemaIDs{fullName(&testevents.ThingDeleted{}): 0x01020304}

	got, err := ids.Encode(&testevents.ThingDeleted{Id: "n1"})

	require.NoError(t, err)
	require.Equal(t, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x00, 0x0a, 0x02, 'n', '1'}, got)
}

func TestSchemaIDs_Encode_UnknownSchema(t *testing.T) {
	t.Parallel()

	ids := eventbus.SchemaIDs{fullName(&testevents.ThingDeleted{}): 7}

	_, err := ids.Encode(&testevents.ThingCreated{Id: "n1"})

	require.ErrorIs(t, err, eventbus.ErrUnknownSchema)
}

func TestDecode_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message proto.Message
	}{
		{
			name: "ThingCreated",
			message: &testevents.ThingCreated{
				Id:        "thing_1",
				Title:     "Lista de compras",
				CreatedAt: timestamppb.New(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)),
			},
		},
		{name: "ThingDeleted", message: &testevents.ThingDeleted{Id: "thing_1"}},
	}

	ids := eventbus.SchemaIDs{fullName(&testevents.ThingCreated{}): 11, fullName(&testevents.ThingDeleted{}): 12}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value, err := ids.Encode(tt.message)
			require.NoError(t, err)

			schemaID, got, err := eventbus.Decode(fullName(tt.message), value)

			require.NoError(t, err)
			require.Equal(t, ids[fullName(tt.message)], schemaID)
			require.True(t, proto.Equal(tt.message, got), "want %v, got %v", tt.message, got)
		})
	}
}

func TestDecode_UnknownTypeKeepsSchemaID(t *testing.T) {
	t.Parallel()

	schemaID, got, err := eventbus.Decode("acme.billing.v1.InvoicePaid", []byte{0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x0a, 0x01, 'x'})

	require.NoError(t, err)
	require.Equal(t, 9, schemaID)
	require.Nil(t, got)
}

func TestDecode_Errors(t *testing.T) {
	t.Parallel()

	thingCreated := fullName(&testevents.ThingCreated{})

	tests := []struct {
		name    string
		value   []byte
		wantErr error
	}{
		{name: "invalid magic byte", value: []byte{0x01, 0x00, 0x00, 0x00, 0x09, 0x00}, wantErr: eventbus.ErrInvalidFraming},
		{name: "schema ID truncated", value: []byte{0x00, 0x00, 0x09}, wantErr: eventbus.ErrInvalidFraming},
		{name: "empty value", value: nil, wantErr: eventbus.ErrInvalidFraming},
		{name: "message index missing", value: []byte{0x00, 0x00, 0x00, 0x00, 0x09}, wantErr: eventbus.ErrInvalidFraming},
		{name: "more than one message index", value: []byte{0x00, 0x00, 0x00, 0x00, 0x09, 0x04, 0x02, 0x02}, wantErr: eventbus.ErrInvalidFraming},
		{name: "protobuf does not decode", value: []byte{0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x0a, 0x05, 'x'}, wantErr: eventbus.ErrInvalidPayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := eventbus.Decode(thingCreated, tt.value)

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
