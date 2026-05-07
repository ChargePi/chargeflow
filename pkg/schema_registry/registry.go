package schema_registry

import (
	"context"
	"encoding/json"

	"github.com/kaptinlin/jsonschema"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

type CreateSchemaRequest struct {
	OcppContext ocpp.OcppContext
	Action      string
	Schema      json.RawMessage
}

type DeleteSchemaRequest struct {
	OcppContext ocpp.OcppContext
	Action      string
}

type GetSchemaRequest struct {
	OcppContext ocpp.OcppContext
	Action      string
}

type SchemaRegistry interface {
	RegisterSchema(ctx context.Context, req CreateSchemaRequest) error
	DeleteSchema(ctx context.Context, req DeleteSchemaRequest) error
	// GetSchema retrieves a compiled schema for the given OCPP version and action.
	// When Vendor and/or Model are non-empty the registry first attempts to return a
	// vendor/model-specific schema and falls back to the base OCPP spec schema when
	// no specific schema is found.
	GetSchema(ctx context.Context, req GetSchemaRequest) (*jsonschema.Schema, bool)
	Type() string
}
