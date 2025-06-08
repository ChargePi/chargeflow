package schema_registry

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

type registryTestSuite struct {
	suite.Suite
	logger *zap.Logger
}

func (s *registryTestSuite) SetupSuite() {
	s.logger = zap.L()
}

func (s *registryTestSuite) TestRegisterSchema() {
	tests := []struct {
		name         string
		preconfigure func(registry SchemaRegistry)
		ocppVersion  ocpp.Version
		action       string
		schema       json.RawMessage
		opts         []Option
		expectedErr  error
	}{
		{
			name:         "Register schema for OCPP 1.6",
			preconfigure: func(registry SchemaRegistry) {},
			ocppVersion:  ocpp.V16,
			action:       "AuthorizeRequest",
			schema:       json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
			opts:         nil,
			expectedErr:  nil,
		},
		{
			name:         "Register schema for OCPP 2.0",
			preconfigure: func(registry SchemaRegistry) {},
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
			preconfigure: func(registry SchemaRegistry) {
				_ = registry.RegisterSchema(ocpp.V16, "AuthorizeRequest", json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`))
			},
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
			opts:        []Option{WithOverwrite(false)},
			expectedErr: errors.New("schema for action AuthorizeRequest already exists for OCPP version 1.6"),
		},
		{
			name:        "Schema already registered, overwrite enabled",
			ocppVersion: ocpp.V16,
			preconfigure: func(registry SchemaRegistry) {
				_ = registry.RegisterSchema(ocpp.V16, "AuthorizeRequest", json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`))
			},
			action:      "AuthorizeRequest",
			schema:      json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`),
			opts:        []Option{WithOverwrite(true)},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registry := NewInMemorySchemaRegistry(s.logger)

			if tt.preconfigure != nil {
				tt.preconfigure(registry)
			}

			if tt.opts == nil {
				tt.opts = []Option{}
			}

			err := registry.RegisterSchema(tt.ocppVersion, tt.action, tt.schema, tt.opts...)
			if tt.expectedErr != nil {
				s.ErrorContains(err, tt.expectedErr.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *registryTestSuite) TestGetSchema() {
	tests := []struct {
		name          string
		preconfigure  func(registry SchemaRegistry)
		ocppVersion   ocpp.Version
		action        string
		expectedFound bool
	}{
		{
			name: "Get schema for OCPP 1.6",
			preconfigure: func(registry SchemaRegistry) {
				_ = registry.RegisterSchema(ocpp.V16, "AuthorizeRequest", json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`))
			},
			ocppVersion:   ocpp.V16,
			action:        "AuthorizeRequest",
			expectedFound: true,
		},
		{
			name: "Get schema for OCPP 2.0",
			preconfigure: func(registry SchemaRegistry) {
				_ = registry.RegisterSchema(ocpp.V20, "AuthorizeRequest", json.RawMessage(`{ "$schema": "http://json-schema.org/draft-04/schema#", "id": "urn:OCPP:1.6:2019:12:AuthorizeRequest", "title": "AuthorizeRequest", "type": "object", "properties": { "idTag": { "type": "string", "maxLength": 20 } }, "additionalProperties": false, "required": [ "idTag" ]}`))
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
			registry := NewInMemorySchemaRegistry(s.logger)

			if test.preconfigure != nil {
				test.preconfigure(registry)
			}

			schema, found := registry.GetSchema(test.ocppVersion, test.action)
			s.Equal(test.expectedFound, found)
			if test.expectedFound {
				s.NotNil(schema)
			} else {
				s.Nil(schema)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	suite.Run(t, new(registryTestSuite))
}
