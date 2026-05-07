package file_registry

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
)

type fileRegistryTestSuite struct {
	suite.Suite
	logger *zap.Logger
}

func (s *fileRegistryTestSuite) SetupSuite() {
	s.logger = zap.L()
}

func (s *fileRegistryTestSuite) TestRegisterSchema() {
	ctx := context.Background()

	const authorizeSchema = `{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`

	tests := []struct {
		name         string
		preconfigure func(registry *SchemaRegistry)
		ocppVersion  ocpp.Version
		action       string
		schema       json.RawMessage
		opts         []RegistryOption
		expectedErr  error
	}{
		{
			name:         "Register schema for OCPP 1.6",
			preconfigure: func(registry *SchemaRegistry) {},
			ocppVersion:  ocpp.V16,
			action:       "AuthorizeRequest",
			schema:       json.RawMessage(authorizeSchema),
			opts:         nil,
			expectedErr:  nil,
		},
		{
			name:         "Register schema for OCPP 2.0",
			preconfigure: func(registry *SchemaRegistry) {},
			ocppVersion:  ocpp.V20,
			action:       "AuthorizeRequest",
			schema:       json.RawMessage(authorizeSchema),
			opts:         nil,
			expectedErr:  nil,
		},
		{
			name:        "Unsupported OCPP version",
			ocppVersion: ocpp.Version("unsupported"),
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(authorizeSchema),
			opts:        nil,
			expectedErr: errors.New("invalid OCPP version: unsupported"),
		},
		{
			name:        "Unsupported action",
			ocppVersion: ocpp.V20,
			action:      "Authorize",
			schema:      json.RawMessage(authorizeSchema),
			opts:        nil,
			expectedErr: errors.New("action must end with 'Request' or 'Response': Authorize"),
		},
		{
			name:        "Invalid schema",
			ocppVersion: ocpp.V20,
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(`"invalid": "schema" }`),
			opts:        nil,
			expectedErr: errors.New("failed to compile schema"),
		},
		{
			name:        "Schema already registered, overwrite disabled",
			ocppVersion: ocpp.V16,
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
					OcppContext: ocpp.OcppContext{Version: ocpp.V16},
					Action:      "AuthorizeRequest",
					Schema:      json.RawMessage(authorizeSchema),
				})
			},
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(authorizeSchema),
			opts:        []RegistryOption{WithOverwrite(false)},
			expectedErr: errors.New("schema for action AuthorizeRequest already exists for OCPP version 1.6"),
		},
		{
			name:        "Schema already registered, overwrite enabled",
			ocppVersion: ocpp.V16,
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
					OcppContext: ocpp.OcppContext{Version: ocpp.V16},
					Action:      "AuthorizeRequest",
					Schema:      json.RawMessage(authorizeSchema),
				})
			},
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(authorizeSchema),
			opts:        []RegistryOption{WithOverwrite(true)},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.opts == nil {
				tt.opts = []RegistryOption{}
			}

			registry := NewFileSchemaRegistry(s.logger, tt.opts...)

			if tt.preconfigure != nil {
				tt.preconfigure(registry)
			}

			err := registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
				OcppContext: ocpp.OcppContext{Version: tt.ocppVersion},
				Action:      tt.action,
				Schema:      tt.schema,
			})
			if tt.expectedErr != nil {
				s.ErrorContains(err, tt.expectedErr.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *fileRegistryTestSuite) TestGetSchema() {
	ctx := context.Background()

	const authorizeSchema = `{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`

	tests := []struct {
		name          string
		preconfigure  func(registry *SchemaRegistry)
		req           schema_registry.GetSchemaRequest
		expectedFound bool
	}{
		{
			name: "Get base schema for OCPP 1.6",
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
					OcppContext: ocpp.OcppContext{Version: ocpp.V16},
					Action:      "AuthorizeRequest",
					Schema:      json.RawMessage(authorizeSchema),
				})
			},
			req:           schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16}, Action: "AuthorizeRequest"},
			expectedFound: true,
		},
		{
			name: "Get base schema for OCPP 2.0",
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
					OcppContext: ocpp.OcppContext{Version: ocpp.V20},
					Action:      "AuthorizeRequest",
					Schema:      json.RawMessage(authorizeSchema),
				})
			},
			req:           schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V20}, Action: "AuthorizeRequest"},
			expectedFound: true,
		},
		{
			name:          "Schema not found - unknown OCPP version",
			req:           schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.Version("unknown_version")}, Action: "BootNotificationRequest"},
			expectedFound: false,
		},
		{
			name:          "Schema not found - action missing",
			req:           schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V20}, Action: "BootNotificationRequest"},
			expectedFound: false,
		},
		{
			name: "Vendor/model-specific schema returned when registered",
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
					OcppContext: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme", Model: "X1"},
					Action:      "AuthorizeRequest",
					Schema:      json.RawMessage(authorizeSchema),
				})
			},
			req:           schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme", Model: "X1"}, Action: "AuthorizeRequest"},
			expectedFound: true,
		},
		{
			name: "Vendor/model request falls back to base schema when no specific schema exists",
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
					OcppContext: ocpp.OcppContext{Version: ocpp.V16},
					Action:      "AuthorizeRequest",
					Schema:      json.RawMessage(authorizeSchema),
				})
			},
			req:           schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme", Model: "X1"}, Action: "AuthorizeRequest"},
			expectedFound: true,
		},
		{
			name:          "Vendor/model request returns not-found when neither specific nor base schema exists",
			preconfigure:  func(registry *SchemaRegistry) {},
			req:           schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme", Model: "X1"}, Action: "AuthorizeRequest"},
			expectedFound: false,
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			registry := NewFileSchemaRegistry(s.logger)

			if test.preconfigure != nil {
				test.preconfigure(registry)
			}

			schema, found := registry.GetSchema(ctx, test.req)
			s.Equal(test.expectedFound, found)
			if test.expectedFound {
				s.NotNil(schema)
			} else {
				s.Nil(schema)
			}
		})
	}
}

func (s *fileRegistryTestSuite) TestOptions() {
	tests := []struct {
		name     string
		opts     []RegistryOption
		expected fileRegistryOptions
	}{
		{
			name: "default options",
			opts: []RegistryOption{},
			expected: fileRegistryOptions{
				overwrite: false,
			},
		},
		{
			name: "WithOverwrite",
			opts: []RegistryOption{
				WithOverwrite(true),
			},
			expected: fileRegistryOptions{
				overwrite: true,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			options := &fileRegistryOptions{}
			for _, opt := range tt.opts {
				opt(options)
			}
			s.Equal(tt.expected, *options)
		})
	}
}

func (s *fileRegistryTestSuite) TestDeleteSchema() {
	ctx := context.Background()
	schema := json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`)

	tests := []struct {
		name         string
		preconfigure func(registry *SchemaRegistry)
		ocppVersion  ocpp.Version
		action       string
		expectedErr  error
	}{
		{
			name: "Delete existing schema",
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
					OcppContext: ocpp.OcppContext{Version: ocpp.V16},
					Action:      "AuthorizeRequest",
					Schema:      schema,
				})
			},
			ocppVersion: ocpp.V16,
			action:      "AuthorizeRequest",
			expectedErr: nil,
		},
		{
			name:        "Unsupported OCPP version",
			ocppVersion: ocpp.Version("unsupported"),
			action:      "AuthorizeRequest",
			expectedErr: errors.New("invalid OCPP version: unsupported"),
		},
		{
			name:        "No schemas registered for OCPP version",
			ocppVersion: ocpp.V16,
			action:      "AuthorizeRequest",
			expectedErr: errors.New("no schemas registered for OCPP version 1.6"),
		},
		{
			name: "Schema not found for action",
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
					OcppContext: ocpp.OcppContext{Version: ocpp.V16},
					Action:      "AuthorizeRequest",
					Schema:      schema,
				})
			},
			ocppVersion: ocpp.V16,
			action:      "BootNotificationRequest",
			expectedErr: errors.New("schema for action BootNotificationRequest not found for OCPP version 1.6"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registry := NewFileSchemaRegistry(s.logger)
			if tt.preconfigure != nil {
				tt.preconfigure(registry)
			}

			err := registry.DeleteSchema(ctx, schema_registry.DeleteSchemaRequest{
				OcppContext: ocpp.OcppContext{Version: tt.ocppVersion},
				Action:      tt.action,
			})
			if tt.expectedErr != nil {
				s.ErrorContains(err, tt.expectedErr.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *fileRegistryTestSuite) TestRegisterAndDeleteLifecycle() {
	ctx := context.Background()
	registry := NewFileSchemaRegistry(s.logger)
	schema := json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`)

	err := registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
		OcppContext: ocpp.OcppContext{Version: ocpp.V16},
		Action:      "AuthorizeRequest",
		Schema:      schema,
	})
	s.Require().NoError(err)

	_, found := registry.GetSchema(ctx, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16}, Action: "AuthorizeRequest"})
	s.True(found, "schema should be retrievable after registration")

	err = registry.DeleteSchema(ctx, schema_registry.DeleteSchemaRequest{
		OcppContext: ocpp.OcppContext{Version: ocpp.V16},
		Action:      "AuthorizeRequest",
	})
	s.Require().NoError(err)

	_, found = registry.GetSchema(ctx, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16}, Action: "AuthorizeRequest"})
	s.False(found, "schema should not be retrievable after deletion")

	err = registry.RegisterSchema(ctx, schema_registry.CreateSchemaRequest{
		OcppContext: ocpp.OcppContext{Version: ocpp.V16},
		Action:      "AuthorizeRequest",
		Schema:      schema,
	})
	s.NoError(err)
}

func TestInMemoryRegistry(t *testing.T) {
	suite.Run(t, new(fileRegistryTestSuite))
}
