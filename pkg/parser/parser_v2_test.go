package parser

import (
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"testing"
)

type parserSuite struct {
	suite.Suite
}

func (s *parserSuite) TestParse() {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name                      string
		data                      []string
		expectedResults           map[string]RequestResponsePairResult
		expectedNonParsedMessages map[string]Result
		expectedErr               error
	}{
		{
			name: "Valid Request",
			data: []string{`[2,"1234", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`},
			expectedResults: map[string]RequestResponsePairResult{
				"1234": {
					Request: &ocpp.Call{
						MessageTypeId: 2,
						UniqueId:      "1234",
						Action:        "BootNotification",
						Payload:       map[string]interface{}{"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"},
					},
					Result: *NewResult(),
				},
			},
			expectedNonParsedMessages: map[string]Result{},
			expectedErr:               nil,
		},
		{
			name: "Valid Response",
			data: []string{
				`[2,"1234", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`,
				`[3,"1234", {"status": "Accepted"}]`},
			expectedResults: map[string]RequestResponsePairResult{
				"1234": {
					Request: &ocpp.Call{
						MessageTypeId: 2,
						UniqueId:      "1234",
						Action:        "BootNotification",
						Payload:       map[string]interface{}{"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"},
					},
					Response: &ocpp.CallResult{
						MessageTypeId: 3,
						UniqueId:      "1234",
						Action:        "BootNotification",
						Payload:       map[string]interface{}{"status": "Accepted"},
					},
					Result: *NewResult(),
				},
			},
			expectedNonParsedMessages: map[string]Result{},
			expectedErr:               nil,
		},
		{
			name: "Valid Error",
			data: []string{
				`[2,"1234", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`,
				`[4,"1234", "GenericError", "An error occurred"]`},
			expectedResults: map[string]RequestResponsePairResult{
				"1234": {
					Request: &ocpp.Call{
						MessageTypeId: 2,
						UniqueId:      "1234",
						Action:        "BootNotification",
						Payload:       map[string]interface{}{"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"},
					},
					Response: &ocpp.CallError{
						MessageTypeId:    4,
						UniqueId:         "1234",
						ErrorCode:        "GenericError",
						ErrorDescription: "An error occurred",
					},
					Result: *NewResult(),
				},
			},
			expectedNonParsedMessages: map[string]Result{},
			expectedErr:               nil,
		},
		{
			name: "Invalid Message",
			data: []string{`[5,"1234", "InvalidMessage"]`},
			expectedResults: map[string]RequestResponsePairResult{
				"1234": {
					Result: Result{
						isValid: false,
						errors:  []string{"Unknown message type: 5"},
					},
				},
			},
			expectedNonParsedMessages: map[string]Result{},
			expectedErr:               nil,
		},
		{
			name:            "Invalid JSON",
			data:            []string{`{"invalid": "json"}`},
			expectedResults: map[string]RequestResponsePairResult{},
			expectedNonParsedMessages: map[string]Result{
				"line 1": {
					isValid: false,
					errors:  []string{"Message is not a valid OCPP message"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "No unique ID",
			data: []string{`[2, "", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`},
			expectedResults: map[string]RequestResponsePairResult{
				"line 1": {
					Request: &ocpp.Call{
						MessageTypeId: 2,
						UniqueId:      "line 1",
						Action:        "BootNotification",
						Payload:       map[string]interface{}{"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"},
					},
					Result: Result{
						errors: []string{"Unique ID is missing in the message"},
					},
				},
			},
			expectedNonParsedMessages: map[string]Result{},
			expectedErr:               nil,
		},
		{
			name: "Mixed Valid and Invalid Messages",
			data: []string{
				`[5,"12345", "InvalidMessage"]`,
				`[2,"1234", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`,
				`[3,"1234", {"status": "Accepted"}]`,
				`[2,"12344", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`,
				`[4,"12344", "GenericError", "An error occurred"]`,
				`{"invalid": "json"}`,
			},
			expectedResults: map[string]RequestResponsePairResult{
				"1234": {
					Request: &ocpp.Call{
						MessageTypeId: 2,
						UniqueId:      "1234",
						Action:        "BootNotification",
						Payload:       map[string]interface{}{"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"},
					},
					Response: &ocpp.CallResult{
						MessageTypeId: 3,
						UniqueId:      "1234",
						Action:        "BootNotification",
						Payload:       map[string]interface{}{"status": "Accepted"},
					},
					Result: *NewResult(),
				},
				"12344": {
					Request: &ocpp.Call{
						MessageTypeId: 2,
						UniqueId:      "12344",
						Action:        "BootNotification",
						Payload:       map[string]interface{}{"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"},
					},
					Response: &ocpp.CallError{
						MessageTypeId:    4,
						UniqueId:         "12344",
						ErrorCode:        "GenericError",
						ErrorDescription: "An error occurred",
					},
					Result: *NewResult(),
				},
				"12345": {
					Result: Result{
						errors: []string{"Unknown message type: 5"},
					},
				},
			},
			expectedNonParsedMessages: map[string]Result{
				"line 6": {
					isValid: false,
					errors: []string{
						"Message is not a valid OCPP message",
					},
				},
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			parser := NewParserV2(logger)

			results, nonParsedMessages, err := parser.Parse(test.data)
			s.Equal(test.expectedResults, results)
			s.Equal(test.expectedNonParsedMessages, nonParsedMessages)
			if test.expectedErr != nil {
				s.ErrorContains(test.expectedErr, err.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func TestParserV2(t *testing.T) {
	suite.Run(t, new(parserSuite))
}
