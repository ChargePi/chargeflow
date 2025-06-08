package schema_registry

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

var compiler = jsonschema.NewCompiler()

type SchemaRegistry interface {
	RegisterSchema(ocppVersion ocpp.Version, action string, rawSchema json.RawMessage, opts ...Option) error
	GetSchema(ocppVersion ocpp.Version, action string) (*jsonschema.Schema, bool)
}

type InMemorySchemaRegistry struct {
	logger *zap.Logger
	mu     sync.RWMutex // Protects concurrent access to schemasPerOcppVersion map

	// Map of schema compilers registered per OCPP version
	schemasPerOcppVersion map[ocpp.Version]map[string]*jsonschema.Schema
}

func NewInMemorySchemaRegistry(logger *zap.Logger) *InMemorySchemaRegistry {
	return &InMemorySchemaRegistry{
		logger:                logger.Named("schema_registry"),
		schemasPerOcppVersion: make(map[ocpp.Version]map[string]*jsonschema.Schema),
	}
}

// RegisterSchema registers a new schema for a specific OCPP version and action.
// Example: you would register a schema for the action "BootNotification" in OCPP 1.6 like this:
//
//	err := schemaRegistry.RegisterSchema(ocpp.V16, "BootNotificationRequest", "{...}")
//
// The rawSchema should be a valid JSON schema in raw format.
// The action is the name of the OCPP action that this schema applies to. Must be suffixed with either "Request" or "Response".
func (fsr *InMemorySchemaRegistry) RegisterSchema(ocppVersion ocpp.Version, action string, rawSchema json.RawMessage, opts ...Option) error {
	logger := fsr.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
	logger.Info("Registering schema")

	// Validate the OCPP version
	if !ocpp.IsValidProtocolVersion(ocppVersion) {
		return errors.Errorf("invalid OCPP version: %s", ocppVersion)
	}

	// Must be a valid action name ending with "Request" or "Response"
	if !(strings.HasSuffix(action, "Request") || strings.HasSuffix(action, "Response")) {
		return errors.Errorf("action must end with 'Request' or 'Response': %s", action)
	}

	logger.Debug("Compiling schema")
	// Compile the schema using the jsonschema compiler
	schema, err := compiler.Compile(rawSchema)
	if err != nil {
		return errors.Wrap(err, "failed to compile schema")
	}

	// Default to not overwriting existing schemas
	defaultOpts := &Options{
		overwrite: false,
	}
	for _, opt := range opts {
		opt(defaultOpts)
	}

	// Acquire write lock to modify the schemasPerOcppVersion map
	fsr.mu.RLock()
	defer fsr.mu.RUnlock()

	if _, exists := fsr.schemasPerOcppVersion[ocppVersion]; !exists {
		fsr.schemasPerOcppVersion[ocppVersion] = make(map[string]*jsonschema.Schema)
	}

	if !defaultOpts.overwrite {
		logger.Debug("Schema registry overwrite")
		// Check if the schema already exists for the given action
		if _, exists := fsr.schemasPerOcppVersion[ocppVersion][action]; exists {
			return errors.Errorf("schema for action %s already exists for OCPP version %s", action, ocppVersion)
		}
	}

	// Register the schema for the specific action
	fsr.schemasPerOcppVersion[ocppVersion][action] = schema

	return nil
}

// GetSchema retrieves a schema for a specific OCPP version and action.
func (fsr *InMemorySchemaRegistry) GetSchema(ocppVersion ocpp.Version, action string) (*jsonschema.Schema, bool) {
	fsr.logger.Info("Getting schema", zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))

	fsr.mu.RLock()
	defer fsr.mu.RUnlock()

	// Check if the OCPP version exists in the registry
	if schemas, exists := fsr.schemasPerOcppVersion[ocppVersion]; exists {
		// Check if the action exists for the given OCPP version
		if schema, exists := schemas[action]; exists {
			return schema, true
		}
	}

	return nil, false
}
