package schema_registry

import (
	"encoding/json"
	"go.uber.org/zap"
	"strings"
	"sync"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
)

var compiler = jsonschema.NewCompiler()

type SchemaRegistry struct {
	logger *zap.Logger
	mu     sync.RWMutex // Protects concurrent access to schemasPerOcppVersion map

	// Map of schema compilers registered per OCPP version
	schemasPerOcppVersion map[ocpp.Version]map[string]*jsonschema.Schema
}

func NewSchemaRegistry(logger *zap.Logger) *SchemaRegistry {
	return &SchemaRegistry{
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
func (sr *SchemaRegistry) RegisterSchema(ocppVersion ocpp.Version, action string, rawSchema json.RawMessage, opts ...Option) error {
	logger := sr.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
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
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	if _, exists := sr.schemasPerOcppVersion[ocppVersion]; !exists {
		sr.schemasPerOcppVersion[ocppVersion] = make(map[string]*jsonschema.Schema)
	}

	if !defaultOpts.overwrite {
		logger.Debug("Schema registry overwrite")
		// Check if the schema already exists for the given action
		if _, exists := sr.schemasPerOcppVersion[ocppVersion][action]; exists {
			return errors.Errorf("schema for action %s already exists for OCPP version %s", action, ocppVersion)
		}
	}

	// Register the schema for the specific action
	sr.schemasPerOcppVersion[ocppVersion][action] = schema

	return nil
}

// GetSchema retrieves a schema for a specific OCPP version and action.
func (sr *SchemaRegistry) GetSchema(ocppVersion ocpp.Version, action string) (*jsonschema.Schema, bool) {
	sr.logger.Info("Getting schema", zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))

	sr.mu.RLock()
	defer sr.mu.RUnlock()

	// Check if the OCPP version exists in the registry
	if schemas, exists := sr.schemasPerOcppVersion[ocppVersion]; exists {
		// Check if the action exists for the given OCPP version
		if schema, exists := schemas[action]; exists {
			return schema, true
		}
	}

	return nil, false
}
