package remote_registry

import (
	"context"

	"github.com/kaptinlin/jsonschema"
)

// Cache stores compiled schemas fetched from the remote registry, keyed by subject name.
type Cache interface {
	Get(ctx context.Context, subject string) (*jsonschema.Schema, bool)
	Set(ctx context.Context, subject string, schema *jsonschema.Schema)
	Delete(ctx context.Context, subject string)
}
