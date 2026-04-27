package schema_registry

import (
	"context"
	"encoding/json"

	"github.com/kaptinlin/jsonschema"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

type SchemaRegistry interface {
	RegisterSchema(ctx context.Context, ocppVersion ocpp.Version, action string, rawSchema json.RawMessage) error
	GetSchema(ctx context.Context, ocppVersion ocpp.Version, action string) (*jsonschema.Schema, bool)
	Type() string
}
