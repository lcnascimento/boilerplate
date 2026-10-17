package events

import (
	"context"

	"github.com/twmb/franz-go/pkg/sr"
)

// RegisterSchemas sets FULL_TRANSITIVE on every subject and registers its normalized .proto.
// Running it again with unchanged schemas returns the same IDs.
func RegisterSchemas(ctx context.Context, client *sr.Client) (SchemaIDs, error) {
	schemas, err := loadSchemas()
	if err != nil {
		return nil, err
	}

	ids := SchemaIDs{}

	for _, s := range schemas {
		compatibility := sr.SetCompatibility{Level: sr.CompatFullTransitive}
		for _, result := range client.SetCompatibility(ctx, compatibility, s.subject) {
			if result.Err != nil {
				return nil, ErrRegisterSchema.WithAttribute("subject", s.subject).WithCause(result.Err)
			}
		}

		registered, err := client.CreateSchema(sr.WithParams(ctx, sr.Normalize), s.subject, s.registrySchema())
		if err != nil {
			return nil, ErrRegisterSchema.WithAttribute("subject", s.subject).WithCause(err)
		}

		ids[s.subject] = registered.ID
	}

	return ids, nil
}

// LookupSchemaIDs finds the ID of every subject without registering anything.
func LookupSchemaIDs(ctx context.Context, client *sr.Client) (SchemaIDs, error) {
	schemas, err := loadSchemas()
	if err != nil {
		return nil, err
	}

	ids := SchemaIDs{}

	for _, s := range schemas {
		found, err := client.LookupSchema(sr.WithParams(ctx, sr.Normalize), s.subject, s.registrySchema())
		if err != nil {
			return nil, ErrLookupSchema.WithAttribute("subject", s.subject).WithCause(err)
		}

		ids[s.subject] = found.ID
	}

	return ids, nil
}

func (s schema) registrySchema() sr.Schema {
	return sr.Schema{Schema: s.text, Type: sr.TypeProtobuf}
}
