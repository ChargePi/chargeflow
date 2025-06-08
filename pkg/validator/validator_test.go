package validator

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	mock_schema_registry "github.com/ChargePi/chargeflow/gen/mocks/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

var schema = []byte(`{
      "$schema": "http://json-schema.org/draft-04/schema#",
      "id": "urn:OCPP:1.6:2019:12:BootNotificationRequest",
      "title": "BootNotificationRequest",
      "type": "object",
      "properties": {
          "chargePointVendor": {
              "type": "string",
              "maxLength": 20
          },
          "chargePointModel": {
              "type": "string",
              "maxLength": 20
          },
          "chargePointSerialNumber": {
              "type": "string",
              "maxLength": 25
          },
          "chargeBoxSerialNumber": {
              "type": "string",
              "maxLength": 25
          },
          "firmwareVersion": {
              "type": "string",
              "maxLength": 50
          },
          "iccid": {
              "type": "string",
              "maxLength": 20
          },
          "imsi": {
              "type": "string",
              "maxLength": 20
          },
          "meterType": {
              "type": "string",
              "maxLength": 25
          },
          "meterSerialNumber": {
              "type": "string",
              "maxLength": 25
          }
      },
      "additionalProperties": false,
      "required": [
          "chargePointVendor",
          "chargePointModel"
      ]
  }
  `)

type validatorTestSuite struct {
	suite.Suite
	logger   *zap.Logger
	compiler *jsonschema.Compiler
}

func (s *validatorTestSuite) SetupSuite() {
	s.logger = zap.L()
	s.compiler = jsonschema.NewCompiler()
}

func (s *validatorTestSuite) TestValidateMessage_HappyPath() {
	tests := []struct {
		name          string
		setupRegistry func(*mock_schema_registry.MockSchemaRegistry)
		ocppVersion   ocpp.Version
		message       ocpp.Message
		expected      *ValidationResult
	}{
		{
			name: "Valid OCPP 2.0.1 request",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V20, "BootNotificationRequest").Return(schemaFromCompiler, true)
			},
			ocppVersion: ocpp.V20,
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       []byte("{\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
			},
			expected: NewValidationResult(),
		},
		{
			name: "Valid OCPP 1.6 request",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V16, "BootNotificationRequest").Return(schemaFromCompiler, true)
			},
			ocppVersion: ocpp.V16,
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       []byte("{\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
			},
			expected: NewValidationResult(),
		},
		{
			name: "Valid OCPP 1.6 response",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V20, "Response").Return(schemaFromCompiler, true)
			},
			ocppVersion: ocpp.V20,
			message: &ocpp.CallResult{
				MessageTypeId: ocpp.CALL_RESULT,
				UniqueId:      uuid.NewString(),
				Payload:       []byte("{\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
			},
			expected: NewValidationResult(),
		},
		{
			name:          "Valid OCPP 1.6 error",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			ocppVersion:   ocpp.V16,
			message: &ocpp.CallError{
				MessageTypeId:    ocpp.CALL_ERROR,
				UniqueId:         uuid.NewString(),
				ErrorCode:        ocpp.GenericError,
				ErrorDescription: "An error occurred",
				ErrorDetails:     nil,
			},
			expected: NewValidationResult(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registry := mock_schema_registry.NewMockSchemaRegistry(s.T())
			if tt.setupRegistry != nil {
				tt.setupRegistry(registry)
			}

			validator := NewValidator(s.logger, registry)

			result, err := validator.ValidateMessage(tt.ocppVersion, tt.message)
			s.NoError(err)
			s.Emptyf(result.Errors(), "expected no validation errors but got %v", result.Errors())
		})
	}
}

func (s *validatorTestSuite) TestValidateMessage_UnhappyPath() {
	tests := []struct {
		name          string
		setupRegistry func(*mock_schema_registry.MockSchemaRegistry)
		ocppVersion   ocpp.Version
		message       ocpp.Message
		expected      *ValidationResult
		expectedErr   error
	}{
		{
			name: "Missing uniqueId",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V16, "BootNotificationRequest").Return(schemaFromCompiler, true)
			},
			ocppVersion: ocpp.V16,
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      "",
				Action:        "BootNotification",
				Payload:       []byte("{\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{uniqueIdEmptyErr},
			},
			expectedErr: nil,
		},
		{
			name:          "Invalid request - missing payload",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			ocppVersion:   ocpp.V16,
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       nil,
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{payloadEmptyErr},
			},
			expectedErr: nil,
		},
		{
			name:          "Invalid response - missing payload",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			ocppVersion:   ocpp.V16,
			message: &ocpp.CallResult{
				MessageTypeId: ocpp.CALL_RESULT,
				UniqueId:      uuid.NewString(),
				Payload:       nil,
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{payloadEmptyErr},
			},
			expectedErr: nil,
		},
		{
			name:        "Invalid error - invalid error code",
			ocppVersion: ocpp.V16,
			message: &ocpp.CallError{
				MessageTypeId:    ocpp.CALL_ERROR,
				UniqueId:         uuid.NewString(),
				ErrorCode:        "",
				ErrorDescription: "An error occurred",
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{"invalid error code: "},
			},
			expectedErr: nil,
		},
		{
			name:        "Invalid error - cannot cast to CallError",
			ocppVersion: ocpp.V16,
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL_ERROR,
				UniqueId:      uuid.NewString(),
			},
			expected: &ValidationResult{
				isValid: false,
			},
			expectedErr: ErrCannotCastToCallError,
		},
		{
			name:          "Multiple errors in message - invalid payload, missing uniqueId, malformed payload",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			ocppVersion:   ocpp.V16,
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      "",
				Action:        "",
				Payload:       "notempty",
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{uniqueIdEmptyErr, actionEmptyErr},
			},
			expectedErr: nil,
		},
		{
			name: "No registered schema for request",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				registry.EXPECT().GetSchema(ocpp.V16, "BootNotificationRequest").Return(nil, false)
			},
			ocppVersion: ocpp.V16,
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       "{\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}",
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{},
			},
			expectedErr: errors.New("no schema found for action BootNotificationRequest in OCPP version 1.6"),
		},
		{
			name: "Request schema validation failed",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V16, "BootNotificationRequest").Return(schemaFromCompiler, true)
			},
			ocppVersion: ocpp.V16,
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       []byte("\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{"Invalid JSON format"},
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			registry := mock_schema_registry.NewMockSchemaRegistry(s.T())
			if test.setupRegistry != nil {
				test.setupRegistry(registry)
			}

			validator := NewValidator(s.logger, registry)

			result, err := validator.ValidateMessage(test.ocppVersion, test.message)
			if test.expectedErr != nil {
				s.ErrorContains(err, test.expectedErr.Error())
			} else {
				s.NoError(err)
				for _, e := range test.expected.Errors() {
					s.Contains(result.Errors(), e)
				}
			}
		})
	}
}

func TestValidator(t *testing.T) {
	suite.Run(t, new(validatorTestSuite))
}
