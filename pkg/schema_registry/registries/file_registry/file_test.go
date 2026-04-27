package file_registry

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
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
			schema:       json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
			opts:         nil,
			expectedErr:  nil,
		},
		{
			name:         "Register schema for OCPP 2.0",
			preconfigure: func(registry *SchemaRegistry) {},
			ocppVersion:  ocpp.V20,
			action:       "AuthorizeRequest",
			schema:       json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
			opts:         nil,
			expectedErr:  nil,
		},
		{
			name:        "Unsupported OCPP version",
			ocppVersion: ocpp.Version("unsupported"),
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
			opts:        nil,
			expectedErr: errors.New("invalid OCPP version: unsupported"),
		},
		{
			name:        "Unsupported action",
			ocppVersion: ocpp.V20,
			action:      "Authorize",
			schema:      json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
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
				_ = registry.RegisterSchema(ctx, ocpp.V16, "AuthorizeRequest", json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`))
			},
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
			opts:        []RegistryOption{WithOverwrite(false)},
			expectedErr: errors.New("schema for action AuthorizeRequest already exists for OCPP version 1.6"),
		},
		{
			name:        "Schema already registered, overwrite enabled",
			ocppVersion: ocpp.V16,
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, ocpp.V16, "AuthorizeRequest", json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`))
			},
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
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

			err := registry.RegisterSchema(ctx, tt.ocppVersion, tt.action, tt.schema)
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
	tests := []struct {
		name          string
		preconfigure  func(registry *SchemaRegistry)
		ocppVersion   ocpp.Version
		action        string
		expectedFound bool
	}{
		{
			name: "Get schema for OCPP 1.6",
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, ocpp.V16, "AuthorizeRequest", json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`))
			},
			ocppVersion:   ocpp.V16,
			action:        "AuthorizeRequest",
			expectedFound: true,
		},
		{
			name: "Get schema for OCPP 2.0",
			preconfigure: func(registry *SchemaRegistry) {
				_ = registry.RegisterSchema(ctx, ocpp.V20, "AuthorizeRequest", json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`))
			},
			ocppVersion:   ocpp.V20,
			action:        "AuthorizeRequest",
			expectedFound: true,
		},
		{
			name:          "Schema not found for OCPP version",
			ocppVersion:   ocpp.Version("unknown_version"),
			action:        "BootNotificationRequest",
			expectedFound: false,
		},
		{
			name:          "Schema not found for OCPP version",
			ocppVersion:   ocpp.V20,
			action:        "BootNotificationRequest",
			expectedFound: false,
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			registry := NewFileSchemaRegistry(s.logger)

			if test.preconfigure != nil {
				test.preconfigure(registry)
			}

			schema, found := registry.GetSchema(ctx, test.ocppVersion, test.action)
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

func TestInMemoryRegistry(t *testing.T) {
	suite.Run(t, new(fileRegistryTestSuite))
}
