package audit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application/audit"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application/internal/mocks"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
)

func TestRecord_InsertsTheEntry(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAuditEntriesRepository(gomock.NewController(t))
	repo.EXPECT().Insert(gomock.Any(), validEntry()).Return(nil)

	require.NoError(t, audit.New(repo, newValidator(t)).Record(t.Context(), validEntry()))
}

func TestRecord_RepositoryErrorIsTransient(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAuditEntriesRepository(gomock.NewController(t))
	repo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.ErrMock)

	err := audit.New(repo, newValidator(t)).Record(t.Context(), validEntry())

	require.ErrorIs(t, err, audit.ErrRecordEntry)
	require.ErrorIs(t, err, errors.ErrMock)
	require.NotErrorIs(t, err, eventbus.ErrPermanent)
}

func TestRecord_InvalidEnvelopeIsPermanentAndNotInserted(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		invalidate func(*idomain.Entry)
		reason     string
	}{
		"missing id":                {func(e *idomain.Entry) { e.ID = "" }, "[id] is a required field"},
		"missing source":            {func(e *idomain.Entry) { e.Source = "" }, "[source] is a required field"},
		"missing type":              {func(e *idomain.Entry) { e.Type = "" }, "[type] is a required field"},
		"missing spec version":      {func(e *idomain.Entry) { e.SpecVersion = "" }, "[specversion] is a required field"},
		"missing time":              {func(e *idomain.Entry) { e.Time = time.Time{} }, "[time] is a required field"},
		"missing data content type": {func(e *idomain.Entry) { e.DataContentType = "" }, "[datacontenttype] is a required field"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewMockAuditEntriesRepository(gomock.NewController(t))
			entry := validEntry()
			tc.invalidate(entry)

			err := audit.New(repo, newValidator(t)).Record(t.Context(), entry)

			require.ErrorIs(t, err, eventbus.ErrPermanent)
			require.ErrorIs(t, err, audit.ErrInvalidEnvelope)
			require.Contains(t, errors.SafeReasons(err), tc.reason)
		})
	}
}

func TestRecord_WithoutSubjectIsValid(t *testing.T) {
	t.Parallel()

	entry := validEntry()
	entry.Subject = ""

	repo := mocks.NewMockAuditEntriesRepository(gomock.NewController(t))
	repo.EXPECT().Insert(gomock.Any(), entry).Return(nil)

	require.NoError(t, audit.New(repo, newValidator(t)).Record(t.Context(), entry))
}

func newValidator(t *testing.T) *validator.Validator {
	t.Helper()

	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	return v
}

func validEntry() *idomain.Entry {
	return &idomain.Entry{
		Source:          "notes",
		ID:              "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
		Type:            "boilerplate.notes.v1.NoteDeleted",
		SpecVersion:     "1.0",
		Subject:         "note_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00",
		Time:            time.Date(2026, 10, 17, 12, 30, 0, 0, time.UTC),
		DataContentType: "application/protobuf",
		Headers:         map[string]string{"ce_id": "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e"},
		Payload:         []byte{0x00, 0x00, 0x00, 0x00, 0x02, 0x00},
		KafkaTopic:      "notes.note.v1",
		KafkaPartition:  1,
		KafkaOffset:     7,
	}
}
