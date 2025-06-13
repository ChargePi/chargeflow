package parser

import (
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"testing"

	"github.com/stretchr/testify/suite"
)

type parserTestSuite struct {
	suite.Suite
}

func (s *parserTestSuite) TestParseMessage() {
	logger, _ := zap.NewDevelopment()
	tests := []struct {
		name            string
		data            string
		expectedMessage ocpp.Message
		expectedResult  *Result
		expectedError   error
	}{
		{
			name: "Valid Request",
			data: `[2,"1234", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`,
			expectedMessage: &ocpp.Call{
				MessageTypeId: ocpp.CALL,
				UniqueId:      "1234",
				Payload:       map[string]interface{}{"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"},
				Action:        "BootNotification",
			},
			expectedResult: NewResult(),
			expectedError:  nil,
		},
		{
			name: "Valid Response",
			data: `[3,"1234", {"status": "Accepted"}]`,
			expectedMessage: &ocpp.CallResult{
				MessageTypeId: ocpp.CALL_RESULT,
				UniqueId:      "1234",
				Payload:       map[string]interface{}{"status": "Accepted"},
				Action:        "",
			},
			expectedResult: NewResult(),
			expectedError:  nil,
		},
		{
			name: "Valid Error",
			data: `[4,"1234", "GenericError", "An error occurred"]`,
			expectedMessage: &ocpp.CallError{
				MessageTypeId:    ocpp.CALL_ERROR,
				UniqueId:         "1234",
				ErrorCode:        "GenericError",
				ErrorDescription: "An error occurred",
			},
			expectedResult: NewResult(),
			expectedError:  nil,
		},
		{
			name:            "Invalid Message",
			data:            `[5,"1234", "InvalidMessage"]`,
			expectedMessage: nil,
			expectedResult: &Result{
				errors: []string{"Unknown message type: 5"},
			},
			expectedError: errors.New("Unknown message type: 5"),
		},
		{
			name:            "Invalid JSON",
			data:            `{"invalid": "json"}`,
			expectedMessage: nil,
			expectedResult: &Result{
				errors: []string{"cannot parse message"},
			},
			expectedError: errors.New("cannot parse message"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			parser := NewParser(logger)
			message, result, err := parser.ParseMessage(tt.data)
			if tt.expectedError != nil {
				s.ErrorContains(err, tt.expectedError.Error())
			} else {
				s.NoError(err)
			}
			s.Equal(tt.expectedMessage, message)
			s.Equal(tt.expectedResult, result)
		})
	}
}

func TestParser(t *testing.T) {
	suite.Run(t, new(parserTestSuite))
}
