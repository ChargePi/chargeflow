package validation

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	mock_schema_registry "github.com/ChargePi/chargeflow/gen/mocks/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

var (
	ocpp16validReq   = `[2, "1234", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`
	ocpp16validRes   = `[3, "1234", {"status": "Accepted"}]`
	ocpp201validReq  = `[2, "1234", "CostUpdated", {"totalCost": 2.01, "transactionId": "12345"}]`
	ocpp201validRes  = `[3, "1234", {}]`
	ocpp16invalidReq = `[2, "1234", Invalid", {"errorCode": "GenericError", "errorDescription": "An error occurred"}]`
	ocpp16invalidRes = `[4, "1234", {"errorCode": "GenericError1", "errorDescription": "An error occurred"}]`
	unparsableMsg    = `{"invalid": "json"}`

	bootNotificationSchema = json.RawMessage(`{
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
}`)

	bootNotificationResponseSchema = json.RawMessage(`{
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
}
`)
	costUpdatedSchema = json.RawMessage(`{
  "$schema": "http://json-schema.org/draft-06/schema#",
  "$id": "urn:OCPP:Cp:2:2020:3:CostUpdatedRequest",
  "comment": "OCPP 2.0.1 FINAL",
  "definitions": {
    "CustomDataType": {
      "description": "This class does not get 'AdditionalProperties = false' in the schema generation, so it can be extended with arbitrary JSON properties to allow adding custom data.",
      "javaType": "CustomData",
      "type": "object",
      "properties": {
        "vendorId": {
          "type": "string",
          "maxLength": 255
        }
      },
      "required": [
        "vendorId"
      ]
    }
  },
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "customData": {
      "$ref": "#/definitions/CustomDataType"
    },
    "totalCost": {
      "description": "Current total cost, based on the information known by the CSMS, of the transaction including taxes. In the currency configured with the configuration Variable: [&lt;&lt;configkey-currency, Currency&gt;&gt;]\r\n\r\n",
      "type": "number"
    },
    "transactionId": {
      "description": "Transaction Id of the transaction the current cost are asked for.\r\n\r\n",
      "type": "string",
      "maxLength": 36
    }
  },
  "required": [
    "totalCost",
    "transactionId"
  ]
}`)
	costUpdatedResponseSchema = json.RawMessage(`{
  "$schema": "http://json-schema.org/draft-06/schema#",
  "$id": "urn:OCPP:Cp:2:2020:3:CostUpdatedResponse",
  "comment": "OCPP 2.0.1 FINAL",
  "definitions": {
    "CustomDataType": {
      "description": "This class does not get 'AdditionalProperties = false' in the schema generation, so it can be extended with arbitrary JSON properties to allow adding custom data.",
      "javaType": "CustomData",
      "type": "object",
      "properties": {
        "vendorId": {
          "type": "string",
          "maxLength": 255
        }
      },
      "required": [
        "vendorId"
      ]
    }
  },
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "customData": {
      "$ref": "#/definitions/CustomDataType"
    }
  }
}`)
)

const dir = "./examples"

type file struct {
	content string
	path    string
}

type validationServiceTestSuite struct {
	suite.Suite
	files  map[string]file
	logger *zap.Logger
}

func (s *validationServiceTestSuite) SetupSuite() {
	s.logger = zap.NewExample()
	s.files = make(map[string]file)

	s.setupFiles()
	s.createFiles()
}

func (s *validationServiceTestSuite) setupFiles() {
	s.files["ocpp16_all_valid"] = file{
		content: strings.Join([]string{ocpp16validReq, ocpp16validRes}, "\n"),
	}

	s.files["ocpp201_all_valid"] = file{
		content: strings.Join([]string{ocpp201validReq, ocpp201validRes}, "\n"),
	}

	s.files["ocpp16_mixed"] = file{
		content: strings.Join([]string{ocpp16validReq, ocpp16invalidRes, unparsableMsg}, "\n"),
	}

	s.files["ocpp16_all_invalid"] = file{
		content: strings.Join([]string{ocpp16invalidReq, ocpp16invalidRes, unparsableMsg}, "\n"),
	}

	s.files["ocpp201_invalid"] = file{
		content: strings.Join([]string{ocpp201validRes, ocpp201validReq, unparsableMsg}, "\n"),
	}

	s.files["empty_file"] = file{
		content: " ",
	}

	s.files["invalid_format"] = file{
		content: "{bad json}",
	}
}

func (s *validationServiceTestSuite) createFiles() {
	err := os.Mkdir(dir, 0755)
	s.Require().NoError(err)

	// Operate on a copy of the file map to avoid modifying the original during iteration
	fileCopy := s.files
	for i, file := range fileCopy {
		// Write the content to a file in the specified directory and get the file path
		fileName, err := writeToFile(dir, file.content)
		s.Require().NoError(err)

		file.path = fileName

		s.files[i] = file
	}
}

func (s *validationServiceTestSuite) TearDownSuite() {
	// Clean up the created file and directory
	_ = os.RemoveAll(dir)
}

func (s *validationServiceTestSuite) TestValidateFile() {
	tests := []struct {
		name            string
		filepath        string
		version         ocpp.Version
		setExpectations func(*mock_schema_registry.MockSchemaRegistry)
		expectedErr     error
	}{
		{
			name:     "Valid file with version 1.6",
			filepath: s.files["ocpp16_all_valid"].path,
			version:  ocpp.V16,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {
				compile, err := jsonschema.NewCompiler().Compile(bootNotificationSchema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V16, "BootNotificationRequest").Return(compile, true)

				compile, err = jsonschema.NewCompiler().Compile(bootNotificationResponseSchema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V16, "BootNotificationResponse").Return(compile, true)
			},
			expectedErr: nil,
		},
		{
			name:     "Valid file with version 2.0",
			filepath: s.files["ocpp201_all_valid"].path,
			version:  ocpp.V20,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {
				compile, err := jsonschema.NewCompiler().Compile(costUpdatedSchema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V20, "CostUpdatedRequest").Return(compile, true)

				compile, err = jsonschema.NewCompiler().Compile(costUpdatedResponseSchema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V20, "CostUpdatedResponse").Return(compile, true)
			},
			expectedErr: nil,
		},
		{
			name:     "Invalid version",
			filepath: s.files["ocpp201_all_valid"].path,
			version:  "ocpp.V99",
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {
				registry.EXPECT().GetSchema(ocpp.Version("ocpp.V99"), mock.Anything).Return(nil, false)
			},
			expectedErr: errors.New("no schema found for action CostUpdatedRequest in OCPP version ocpp.V99"),
		},
		{
			name:            "Non-existent file",
			filepath:        "./examples/non_existent_file.txt",
			version:         ocpp.V16,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			expectedErr:     errors.New("failed to open file"),
		},
		{
			name:     "Invalid messages",
			filepath: s.files["empty_file"].path,
			version:  ocpp.V16,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registry := mock_schema_registry.NewMockSchemaRegistry(s.T())
			if tt.setExpectations != nil {
				tt.setExpectations(registry)
			}

			service := NewService(s.logger, registry)
			err := service.ValidateFile(tt.filepath, tt.version)
			if tt.expectedErr != nil {
				s.ErrorContains(err, tt.expectedErr.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *validationServiceTestSuite) TestValidateMessage() {
	tests := []struct {
		name            string
		message         string
		version         ocpp.Version
		setExpectations func(*mock_schema_registry.MockSchemaRegistry)
		expectedErr     error
	}{
		{
			name:    "Valid message with version 1.6",
			message: ocpp16validReq,
			version: ocpp.V16,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {
				compile, err := jsonschema.NewCompiler().Compile(bootNotificationSchema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V16, "BootNotificationRequest").Return(compile, true)
			},
			expectedErr: nil,
		},
		{
			name:    "Valid message with version 2.0",
			message: ocpp201validReq,
			version: ocpp.V20,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {
				compile, err := jsonschema.NewCompiler().Compile(costUpdatedSchema)
				s.Require().NoError(err)
				registry.EXPECT().GetSchema(ocpp.V20, "CostUpdatedRequest").Return(compile, true)
			},
			expectedErr: nil,
		},
		{
			name:    "Invalid message with version 1.6",
			message: ocpp16invalidReq,
			version: ocpp.V16,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {
				//registry.EXPECT().GetSchema(ocpp.V16, "InvalidRequest").Return(nil, false)
			},
			expectedErr: nil,
		},
		{
			name:            "Unparsable message",
			message:         "{invalid: json}",
			version:         ocpp.V20,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			expectedErr:     nil,
		},
		{
			name:            "Invalid message",
			message:         "[5, \"1234\", \"Invalid\", {\"errorCode\": \"GenericError\", \"errorDescription\": \"An error occurred\"}]",
			version:         ocpp.V16,
			setExpectations: func(registry *mock_schema_registry.MockSchemaRegistry) {},
			expectedErr:     nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registry := mock_schema_registry.NewMockSchemaRegistry(s.T())
			if tt.setExpectations != nil {
				tt.setExpectations(registry)
			}

			// todo capture messages from logger and check if they match the expected output
			service := NewService(s.logger, registry)

			err := service.ValidateMessage(tt.message, tt.version)
			if tt.expectedErr != nil {
				s.ErrorContains(err, tt.expectedErr.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func TestValidationService(t *testing.T) {
	suite.Run(t, new(validationServiceTestSuite))
}

func writeToFile(filePath string, content string) (string, error) {
	// Create a temp file and write the content to it
	file, err := os.CreateTemp(filePath, "*.txt")
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return file.Name(), err
}
