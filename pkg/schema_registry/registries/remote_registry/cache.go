package remote_registry

import (
	"context"

	"github.com/kaptinlin/jsonschema"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

// Cache stores compiled schemas fetched from the remote registry.
type Cache interface {
	Get(ctx context.Context, version ocpp.Version, action string) (*jsonschema.Schema, bool)
	Set(ctx context.Context, version ocpp.Version, action string, schema *jsonschema.Schema)
	Delete(ctx context.Context, version ocpp.Version, action string)
}
