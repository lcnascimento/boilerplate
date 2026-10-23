package consumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	iamevents "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"

	"github.com/lcnascimento/go-kit/postgres/testdb"
	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"

	notesapp "github.com/lcnascimento/boilerplate/backend/services/notes/internal/application/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/consumer"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
)

const waitFor = 30 * time.Second

var (
	acme    = iamdomain.MustParseOrganizationID(acmeID)
	globex  = iamdomain.MustParseOrganizationID("org_4d7a1c3e-5b6f-4a8b-9c0d-1e2f3a4b5c6d")
	initech = iamdomain.MustParseOrganizationID("org_1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f")

	schemaIDs = eventbus.SchemaIDs{
		string((&events.NoteCreated{}).ProtoReflect().Descriptor().FullName()): 7,
	}
)

func TestComponentConsumer_OrganizationDeletedDeletesOnlyThatOrganizationsNotesOnEveryDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	env := eventbustest.New(t, iamevents.Protos)
	env.CreateTopic(t, iamevents.OrganizationTopic, 3)
	env.CreateTopic(t, eventbus.DLQTopic(iamevents.OrganizationTopic, consumer.Group), 1)

	repo := notes.New(testdb.New(t, migrations.FS), schemaIDs)
	createNote(t, repo, acme, "da Acme")
	createNote(t, repo, acme, "outra da Acme")
	createNote(t, repo, globex, "da Globex")
	createNote(t, repo, initech, "da Initech")

	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	c := consumer.New(notesapp.New(repo, v), env.Brokers, consumer.WithShutdown(func() { t.Error("consumer stopped unexpectedly") }))
	require.NoError(t, c.Start(t.Context()))
	t.Cleanup(func() { _ = c.Stop(context.WithoutCancel(t.Context())) })

	at := timestamppb.New(time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC))
	for _, msg := range []proto.Message{
		&iamevents.MemberRemoved{OrganizationId: globex.String(), MemberId: "member_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00"},
		&iamevents.OrganizationDeleted{OrganizationId: acme.String(), DeletedAt: at},
		&iamevents.OrganizationRenamed{OrganizationId: globex.String(), RenamedAt: at},
		&iamevents.OrganizationDeleted{OrganizationId: acme.String(), DeletedAt: at},
		&iamevents.OrganizationDeleted{OrganizationId: initech.String(), DeletedAt: at},
	} {
		env.PublishTo(t, iamevents.OrganizationTopic, "one-partition-keeps-the-order", msg)
	}

	require.Eventually(t, func() bool {
		list, err := repo.List(t.Context(), initech)

		return err == nil && len(list) == 0
	}, waitFor, 100*time.Millisecond)

	require.Empty(t, titlesOf(t, repo, acme))
	require.Equal(t, []string{"da Globex"}, titlesOf(t, repo, globex))
}

func createNote(t *testing.T, repo *notes.Repository, orgID iamdomain.OrganizationID, title string) {
	t.Helper()

	note := fixture.RandomNote()
	note.OrganizationID = orgID
	note.Title = title

	require.NoError(t, repo.Create(t.Context(), actortest.Owner(orgID), note))
}

func titlesOf(t *testing.T, repo *notes.Repository, orgID iamdomain.OrganizationID) []string {
	t.Helper()

	list, err := repo.List(t.Context(), orgID)
	require.NoError(t, err)

	titles := make([]string, 0, len(list))
	for _, note := range list {
		titles = append(titles, note.Title)
	}

	return titles
}
