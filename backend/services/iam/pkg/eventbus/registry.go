package eventbus

import (
	"context"
	"io/fs"

	"github.com/twmb/franz-go/pkg/sr"
)

func RegisterSchemas(ctx context.Context, client *sr.Client, protos fs.FS) (SchemaIDs, error) {
	schemas, err := loadSchemas(protos)
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

func LookupSchemaIDs(ctx context.Context, client *sr.Client, protos fs.FS) (SchemaIDs, error) {
	schemas, err := loadSchemas(protos)
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
