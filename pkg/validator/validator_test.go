package validator

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	mock_schema_registry "github.com/ChargePi/chargeflow/gen/mocks/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
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

var responseSchema = []byte(`{
    "$schema": "http://json-schema.org/draft-04/schema#",
    "id": "urn:OCPP:1.6:2019:12:BootNotificationResponse",
    "title": "BootNotificationResponse",
    "type": "object",
    "properties": {
        "status": {
            "type": "string",
            "additionalProperties": false,
            "enum": [
                "Accepted",
                "Pending",
                "Rejected"
            ]
        },
        "currentTime": {
            "type": "string",
            "format": "date-time"
        },
        "interval": {
            "type": "integer"
        }
    },
    "additionalProperties": false,
    "required": [
        "status",
        "currentTime",
        "interval"
    ]
}`)

// permissiveSchema accepts any object payload; used where the test cares about the
// OCMF hook rather than full OCPP structural validation.
var permissiveSchema = []byte(`{"type": "object"}`)

const validOCMFRecord = `OCMF|{"FV":"1.0","PG":"T1","RD":[{"TM":"2018-07-24T13:22:04,000+0200 S","ST":"G"}]}|{"SD":"AA"}`

// invalidOCMFRecord has an "RI" reading field without its required pair "RU".
const invalidOCMFRecord = `OCMF|{"FV":"1.0","PG":"T1","RD":[{"TM":"2018-07-24T13:22:04,000+0200 S","ST":"G","RI":"1-b:1.8.0"}]}|{"SD":"AA"}`

// meterValuesPayload16 builds an OCPP 1.6 MeterValues.req payload carrying value as the
// raw sampledValue, as used for signed (e.g. OCMF) readings in that version.
func meterValuesPayload16(value string) map[string]interface{} {
	return map[string]interface{}{
		"connectorId": float64(1),
		"meterValue": []interface{}{
			map[string]interface{}{
				"timestamp": "2023-10-01T12:00:00Z",
				"sampledValue": []interface{}{
					map[string]interface{}{
						"value":  value,
						"format": "SignedData",
					},
				},
			},
		},
	}
}

// meterValuesPayload20 builds an OCPP 2.0.1 MeterValues.req payload carrying value inside
// a sampledValue.signedMeterValue sub-object, base64-encoding it as the OCPP spec requires.
func meterValuesPayload20(encodingMethod, value string) map[string]interface{} {
	return map[string]interface{}{
		"evseId": float64(1),
		"meterValue": []interface{}{
			map[string]interface{}{
				"timestamp": "2023-10-01T12:00:00Z",
				"sampledValue": []interface{}{
					map[string]interface{}{
						"value": float64(0),
						"signedMeterValue": map[string]interface{}{
							"signedMeterData": base64.StdEncoding.EncodeToString([]byte(value)),
							"signingMethod":   "",
							"encodingMethod":  encodingMethod,
							"publicKey":       "",
						},
					},
				},
			},
		},
	}
}

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
		ocppCtx       ocpp.OcppContext
		message       ocpp.Message
		expected      *ValidationResult
	}{
		{
			name: "Valid OCPP 2.0.1 request",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V20}, Action: "BootNotificationRequest"}).Return(schemaFromCompiler, true)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V20},
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
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16}, Action: "BootNotificationRequest"}).Return(schemaFromCompiler, true)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16},
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
				schemaFromCompiler, err := s.compiler.Compile(responseSchema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V20}, Action: "BootNotificationResponse"}).Return(schemaFromCompiler, true)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V20},
			message: &ocpp.CallResult{
				MessageTypeId: ocpp.CALL_RESULT,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       []byte("{\"status\":\"Accepted\",\"currentTime\":\"2023-10-01T12:00:00Z\",\"interval\":10}"),
			},
			expected: NewValidationResult(),
		},
		{
			name:          "Valid OCPP 1.6 error",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			ocppCtx:       ocpp.OcppContext{Version: ocpp.V16},
			message: &ocpp.CallError{
				MessageTypeId:    ocpp.CALL_ERROR,
				UniqueId:         uuid.NewString(),
				ErrorCode:        ocpp.GenericError,
				ErrorDescription: "An error occurred",
				ErrorDetails:     nil,
			},
			expected: NewValidationResult(),
		},
		{
			name:          "Valid OCPP 2.1 SEND request",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			ocppCtx:       ocpp.OcppContext{Version: ocpp.V21},
			message: &ocpp.CallError{
				MessageTypeId:    ocpp.CALL_ERROR,
				UniqueId:         uuid.NewString(),
				ErrorCode:        ocpp.GenericError,
				ErrorDescription: "An error occurred",
				ErrorDetails:     nil,
			},
			expected: NewValidationResult(),
		},
		{
			name: "Valid request with vendor and model",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme", Model: "X1"}, Action: "BootNotificationRequest"}).Return(schemaFromCompiler, true)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme", Model: "X1"},
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       []byte("{\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
			},
			expected: NewValidationResult(),
		},
		{
			name: "Valid request with vendor only",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme"}, Action: "BootNotificationRequest"}).Return(schemaFromCompiler, true)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme"},
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       []byte("{\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
			},
			expected: NewValidationResult(),
		},
		{
			name: "Valid request with model only",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16, Model: "X1"}, Action: "BootNotificationRequest"}).Return(schemaFromCompiler, true)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16, Model: "X1"},
			message: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       []byte("{\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
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

			result, err := validator.ValidateMessage(tt.ocppCtx, tt.message)
			s.NoError(err)
			s.Emptyf(result.Errors(), "expected no validation errors but got %v", result.Errors())
		})
	}
}

func (s *validatorTestSuite) TestValidateMessage_UnhappyPath() {
	tests := []struct {
		name          string
		setupRegistry func(*mock_schema_registry.MockSchemaRegistry)
		ocppCtx       ocpp.OcppContext
		message       ocpp.Message
		expected      *ValidationResult
		expectedErr   error
	}{
		{
			name: "Missing uniqueId",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				schemaFromCompiler, err := s.compiler.Compile(schema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16}, Action: "BootNotificationRequest"}).Return(schemaFromCompiler, true)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16},
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
			ocppCtx:       ocpp.OcppContext{Version: ocpp.V16},
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
			ocppCtx:       ocpp.OcppContext{Version: ocpp.V16},
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
			name:    "Invalid error - invalid error code",
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16},
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
			name:    "Invalid error - cannot cast to CallError",
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16},
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
			ocppCtx:       ocpp.OcppContext{Version: ocpp.V16},
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
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16}, Action: "BootNotificationRequest"}).Return(nil, false)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16},
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
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16}, Action: "BootNotificationRequest"}).Return(schemaFromCompiler, true)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16},
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
		{
			name: "Unsupported message (SEND) for ocpp version",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16},
			message: &ocpp.Send{
				MessageTypeId: ocpp.SEND,
				UniqueId:      uuid.NewString(),
				Action:        "BootNotification",
				Payload:       []byte("\"chargePointVendor\":\"Vendor\",\"chargePointModel\":\"Model\"}"),
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{"SEND messages are only supported in OCPP 2.1"},
			},
			expectedErr: nil,
		},
		{
			name: "Unsupported message (CALL_RESULT_ERROR) for ocpp version",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16},
			message: &ocpp.CallResultError{
				MessageTypeId:    ocpp.CALL_RESULT_ERROR,
				UniqueId:         uuid.NewString(),
				ErrorCode:        "GenericError",
				ErrorDescription: "Error occurred",
			},
			expected: &ValidationResult{
				isValid: false,
				errors:  []string{"CALL_RESULT_ERROR messages are only supported in OCPP 2.1"},
			},
			expectedErr: nil,
		},
		{
			name: "No registered schema for vendor and model specific request",
			setupRegistry: func(registry *mock_schema_registry.MockSchemaRegistry) {
				registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme", Model: "X1"}, Action: "BootNotificationRequest"}).Return(nil, false)
			},
			ocppCtx: ocpp.OcppContext{Version: ocpp.V16, Vendor: "Acme", Model: "X1"},
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
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			registry := mock_schema_registry.NewMockSchemaRegistry(s.T())
			if test.setupRegistry != nil {
				test.setupRegistry(registry)
			}

			validator := NewValidator(s.logger, registry)

			result, err := validator.ValidateMessage(test.ocppCtx, test.message)
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

func (s *validatorTestSuite) TestValidateMessage_MeterValuesOCMF() {
	tests := []struct {
		name        string
		ocppCtx     ocpp.OcppContext
		action      string
		payload     interface{}
		expectOCMF  bool
		wantErrText string
	}{
		{
			name:       "OCPP 1.6 MeterValues with valid OCMF sampled value",
			ocppCtx:    ocpp.OcppContext{Version: ocpp.V16},
			action:     "MeterValues",
			payload:    meterValuesPayload16(validOCMFRecord),
			expectOCMF: false,
		},
		{
			name:        "OCPP 1.6 MeterValues with invalid OCMF sampled value",
			ocppCtx:     ocpp.OcppContext{Version: ocpp.V16},
			action:      "MeterValues",
			payload:     meterValuesPayload16(invalidOCMFRecord),
			expectOCMF:  true,
			wantErrText: "OCMF:",
		},
		{
			name:       "OCPP 1.6 MeterValues with non-OCMF sampled value",
			ocppCtx:    ocpp.OcppContext{Version: ocpp.V16},
			action:     "MeterValues",
			payload:    meterValuesPayload16("1234.5"),
			expectOCMF: false,
		},
		{
			name:       "OCPP 2.0 MeterValues with valid OCMF signedMeterValue",
			ocppCtx:    ocpp.OcppContext{Version: ocpp.V20},
			action:     "MeterValues",
			payload:    meterValuesPayload20("OCMF", validOCMFRecord),
			expectOCMF: false,
		},
		{
			name:        "OCPP 2.0 MeterValues with invalid OCMF signedMeterValue",
			ocppCtx:     ocpp.OcppContext{Version: ocpp.V20},
			action:      "MeterValues",
			payload:     meterValuesPayload20("OCMF", invalidOCMFRecord),
			expectOCMF:  true,
			wantErrText: "OCMF:",
		},
		{
			name:       "OCPP 2.0 MeterValues with non-OCMF encodingMethod is not checked",
			ocppCtx:    ocpp.OcppContext{Version: ocpp.V20},
			action:     "MeterValues",
			payload:    meterValuesPayload20("EDL", invalidOCMFRecord),
			expectOCMF: false,
		},
		{
			name:       "OCPP 1.6 MeterValues is not checked for OCPP 2.0's signedMeterValue shape",
			ocppCtx:    ocpp.OcppContext{Version: ocpp.V16},
			action:     "MeterValues",
			payload:    meterValuesPayload20("OCMF", invalidOCMFRecord),
			expectOCMF: false,
		},
		{
			name:        "OCPP 2.0 TransactionEvent with invalid OCMF signedMeterValue",
			ocppCtx:     ocpp.OcppContext{Version: ocpp.V20},
			action:      "TransactionEvent",
			payload:     meterValuesPayload20("OCMF", invalidOCMFRecord),
			expectOCMF:  true,
			wantErrText: "OCMF:",
		},
		{
			name:       "OCPP 2.0 TransactionEvent without meterValue is not checked",
			ocppCtx:    ocpp.OcppContext{Version: ocpp.V20},
			action:     "TransactionEvent",
			payload:    map[string]interface{}{"eventType": "Started"},
			expectOCMF: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registry := mock_schema_registry.NewMockSchemaRegistry(s.T())
			schemaFromCompiler, err := s.compiler.Compile(permissiveSchema)
			s.Require().NoError(err)
			registry.EXPECT().GetSchema(mock.Anything, schema_registry.GetSchemaRequest{OcppContext: tt.ocppCtx, Action: tt.action + "Request"}).Return(schemaFromCompiler, true)

			validator := NewValidator(s.logger, registry)

			result, err := validator.ValidateMessage(tt.ocppCtx, &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      uuid.NewString(),
				Action:        tt.action,
				Payload:       tt.payload,
			})
			s.Require().NoError(err)

			if !tt.expectOCMF {
				s.Emptyf(result.Errors(), "expected no validation errors but got %v", result.Errors())
				return
			}

			s.NotEmpty(result.Errors())
			found := false
			for _, e := range result.Errors() {
				if strings.Contains(e, tt.wantErrText) {
					found = true
					break
				}
			}
			s.Truef(found, "expected an error containing %q, got %v", tt.wantErrText, result.Errors())
		})
	}
}

func TestValidator(t *testing.T) {
	suite.Run(t, new(validatorTestSuite))
}
