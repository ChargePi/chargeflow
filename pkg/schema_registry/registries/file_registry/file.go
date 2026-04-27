package file_registry

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

const (
	RequestSuffix  = "Request"
	ResponseSuffix = "Response"
)

var compiler *jsonschema.Compiler

func init() {
	compiler = jsonschema.NewCompiler()
}

type SchemaRegistry struct {
	logger *zap.Logger
	config fileRegistryOptions

	mu sync.RWMutex // Protects concurrent access to schemasPerOcppVersion map
	// Map of schema compilers registered per OCPP version
	schemasPerOcppVersion map[ocpp.Version]map[string]*jsonschema.Schema
}

func NewFileSchemaRegistry(logger *zap.Logger, opts ...RegistryOption) *SchemaRegistry {
	// Default to not overwriting existing schemas
	defaultOpts := fileRegistryOptions{
		overwrite: false,
	}

	for _, opt := range opts {
		opt(&defaultOpts)
	}

	registry := &SchemaRegistry{
		logger:                logger.Named("file_schema_registry"),
		schemasPerOcppVersion: make(map[ocpp.Version]map[string]*jsonschema.Schema),
		config:                defaultOpts,
	}

	return registry
}

// RegisterSchema registers a new schema for a specific OCPP version and action.
// Example: you would register a schema for the action "BootNotification" in OCPP 1.6 like this:
//
//	err := schemaRegistry.RegisterSchema(ocpp.V16, "BootNotificationRequest", "{...}")
//
// The rawSchema should be a valid JSON schema in raw format.
// The action is the name of the OCPP action that this schema applies to. Must be suffixed with either "Request" or "Response".
func (fsr *SchemaRegistry) RegisterSchema(_ context.Context, ocppVersion ocpp.Version, action string, rawSchema json.RawMessage) error {
	logger := fsr.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
	logger.Debug("Registering schema")

	// Validate the OCPP version
	if !ocpp.IsValidProtocolVersion(ocppVersion) {
		return errors.Errorf("invalid OCPP version: %s", ocppVersion)
	}

	// Must be a valid action name ending with "Request" or "Response"
	if !(strings.HasSuffix(action, RequestSuffix) || strings.HasSuffix(action, ResponseSuffix)) {
		return errors.Errorf("action must end with 'Request' or 'Response': %s", action)
	}

	logger.Debug("Compiling schema")
	// Compile the schema using the jsonschema compiler
	schema, err := compiler.Compile(rawSchema)
	if err != nil {
		return errors.Wrap(err, "failed to compile schema")
	}

	// Acquire write lock to modify the schemasPerOcppVersion map
	fsr.mu.Lock()
	defer fsr.mu.Unlock()

	if _, exists := fsr.schemasPerOcppVersion[ocppVersion]; !exists {
		fsr.schemasPerOcppVersion[ocppVersion] = make(map[string]*jsonschema.Schema)
	}

	if !fsr.config.overwrite {
		logger.Debug("Overwriting previous schema")
		// Check if the schema already exists for the given action
		if _, exists := fsr.schemasPerOcppVersion[ocppVersion][action]; exists {
			return errors.Errorf("schema for action %s already exists for OCPP version %s", action, ocppVersion)
		}
	}

	// Register the schema for the specific action
	fsr.schemasPerOcppVersion[ocppVersion][action] = schema

	return nil
}

// DeleteSchema removes a schema for a specific OCPP version and action.
func (fsr *SchemaRegistry) DeleteSchema(_ context.Context, ocppVersion ocpp.Version, action string) error {
	logger := fsr.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
	logger.Debug("Deleting schema")

	if !ocpp.IsValidProtocolVersion(ocppVersion) {
		return errors.Errorf("invalid OCPP version: %s", ocppVersion)
	}

	fsr.mu.Lock()
	defer fsr.mu.Unlock()

	schemas, exists := fsr.schemasPerOcppVersion[ocppVersion]
	if !exists {
		return errors.Errorf("no schemas registered for OCPP version %s", ocppVersion)
	}

	if _, exists := schemas[action]; !exists {
		return errors.Errorf("schema for action %s not found for OCPP version %s", action, ocppVersion)
	}

	delete(schemas, action)
	return nil
}

// GetSchema retrieves a schema for a specific OCPP version and action.
func (fsr *SchemaRegistry) GetSchema(_ context.Context, ocppVersion ocpp.Version, action string) (*jsonschema.Schema, bool) {
	fsr.logger.Debug("Getting schema", zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))

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

func (fsr *SchemaRegistry) Type() string {
	return "file"
}
