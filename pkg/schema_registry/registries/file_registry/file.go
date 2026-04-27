package file_registry

import (
	"context"
	"strings"
	"sync"

	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
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
func (fsr *SchemaRegistry) RegisterSchema(_ context.Context, req schema_registry.CreateSchemaRequest) error {
	logger := fsr.logger.With(zap.String("ocppVersion", req.OcppContext.Version.String()), zap.String("action", req.Action))
	logger.Debug("Registering schema")

	// Validate the OCPP version
	if !ocpp.IsValidProtocolVersion(req.OcppContext.Version) {
		return errors.Errorf("invalid OCPP version: %s", req.OcppContext.Version)
	}

	// Must be a valid action name ending with "Request" or "Response"
	if !(strings.HasSuffix(req.Action, RequestSuffix) || strings.HasSuffix(req.Action, ResponseSuffix)) {
		return errors.Errorf("action must end with 'Request' or 'Response': %s", req.Action)
	}

	logger.Debug("Compiling schema")
	schema, err := compiler.Compile(req.Schema)
	if err != nil {
		return errors.Wrap(err, "failed to compile schema")
	}

	fsr.mu.Lock()
	defer fsr.mu.Unlock()

	if _, exists := fsr.schemasPerOcppVersion[req.OcppContext.Version]; !exists {
		fsr.schemasPerOcppVersion[req.OcppContext.Version] = make(map[string]*jsonschema.Schema)
	}

	key := buildStorageKey(req.OcppContext.Vendor, req.OcppContext.Model, req.Action)

	if !fsr.config.overwrite {
		logger.Debug("Overwriting previous schema")
		if _, exists := fsr.schemasPerOcppVersion[req.OcppContext.Version][key]; exists {
			return errors.Errorf("schema for action %s already exists for OCPP version %s", req.Action, req.OcppContext.Version)
		}
	}

	fsr.schemasPerOcppVersion[req.OcppContext.Version][key] = schema

	return nil
}

// DeleteSchema removes a schema for a specific OCPP version and action.
func (fsr *SchemaRegistry) DeleteSchema(_ context.Context, req schema_registry.DeleteSchemaRequest) error {
	logger := fsr.logger.With(zap.String("ocppVersion", req.OcppContext.Version.String()), zap.String("action", req.Action))
	logger.Debug("Deleting schema")

	if !ocpp.IsValidProtocolVersion(req.OcppContext.Version) {
		return errors.Errorf("invalid OCPP version: %s", req.OcppContext.Version)
	}

	fsr.mu.Lock()
	defer fsr.mu.Unlock()

	schemas, exists := fsr.schemasPerOcppVersion[req.OcppContext.Version]
	if !exists {
		return errors.Errorf("no schemas registered for OCPP version %s", req.OcppContext.Version)
	}

	key := buildStorageKey(req.OcppContext.Vendor, req.OcppContext.Model, req.Action)
	if _, exists := schemas[key]; !exists {
		return errors.Errorf("schema for action %s not found for OCPP version %s", req.Action, req.OcppContext.Version)
	}

	delete(schemas, key)
	return nil
}

// buildStorageKey returns a composite key that incorporates vendor and model when
// provided, keeping vendor/model-specific schemas separate from base schemas.
func buildStorageKey(vendor, model, action string) string {
	if vendor == "" && model == "" {
		return action
	}
	parts := make([]string, 0, 3)
	if vendor != "" {
		parts = append(parts, vendor)
	}
	if model != "" {
		parts = append(parts, model)
	}
	parts = append(parts, action)
	return strings.Join(parts, "|")
}

// GetSchema retrieves a schema for a specific OCPP version and action.
// When Vendor and/or Model are set it first tries the vendor/model-specific
// schema and falls back to the base OCPP spec schema.
func (fsr *SchemaRegistry) GetSchema(_ context.Context, req schema_registry.GetSchemaRequest) (*jsonschema.Schema, bool) {
	fsr.logger.Debug("Getting schema",
		zap.String("ocppVersion", req.OcppContext.Version.String()),
		zap.String("action", req.Action),
		zap.String("vendor", req.OcppContext.Vendor),
		zap.String("model", req.OcppContext.Model),
	)

	fsr.mu.RLock()
	defer fsr.mu.RUnlock()

	schemas, exists := fsr.schemasPerOcppVersion[req.OcppContext.Version]
	if !exists {
		return nil, false
	}

	// Try vendor/model-specific key first when either field is provided.
	if req.OcppContext.Vendor != "" || req.OcppContext.Model != "" {
		if schema, ok := schemas[buildStorageKey(req.OcppContext.Vendor, req.OcppContext.Model, req.Action)]; ok {
			return schema, true
		}
	}

	// Fall back to the base OCPP spec schema.
	if schema, ok := schemas[req.Action]; ok {
		return schema, true
	}

	return nil, false
}

func (fsr *SchemaRegistry) Type() string {
	return "file"
}
