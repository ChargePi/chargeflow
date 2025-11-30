package schema_registry

import (
	"encoding/json"

	"github.com/kaptinlin/jsonschema"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

type SchemaRegistry interface {
	RegisterSchema(ocppVersion ocpp.Version, action string, rawSchema json.RawMessage) error
	GetSchema(ocppVersion ocpp.Version, action string) (*jsonschema.Schema, bool)
	Type() string
}
