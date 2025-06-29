package parser

import (
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"testing"

	"github.com/stretchr/testify/suite"
)

type resultTestSuite struct {
	suite.Suite
}

func (s *resultTestSuite) TestAddError() {
	result := NewResult()
	result.AddError("test error")
	s.Contains(result.Errors(), "test error")

	result.AddError("another error")
	s.Contains(result.Errors(), "another error")
}

func (s *resultTestSuite) TestIsValid() {
	result := NewResult()
	s.True(result.IsValid())

	result.AddError("test error")
	s.False(result.IsValid())

	result.AddError("test 2 error")
	s.False(result.IsValid())
}

func (s *resultTestSuite) TestErrors() {
	result := NewResult()
	s.Empty(result.Errors())

	result.AddError("test error")
	result.AddError("another error")
	s.Equal([]string{"test error", "another error"}, result.Errors())
}

func (s *resultTestSuite) TestSetAndGetMessage() {
	result := NewResult()
	s.Nil(result.Message())

	msg := &ocpp.Call{
		MessageTypeId: ocpp.CALL,
		UniqueId:      "1234",
		Action:        "BootNotification",
		Payload:       map[string]interface{}{"chargePointVendor": "Vendor", "chargePointModel": "Model"},
	}
	result.SetMessage(msg)

	s.Equal(msg, result.Message())
}

type requestResponseResultTestSuite struct {
	suite.Suite
}

func (s *requestResponseResultTestSuite) TestAddRequestError() {
	result := NewRequestResponseResult()
	result.AddRequestError("test request error")
	s.Contains(result.Request.errors, "test request error")

	result.AddRequestError("another request error")
	s.Contains(result.Request.errors, "another request error")
}

func (s *requestResponseResultTestSuite) TestAddResponseError() {
	result := NewRequestResponseResult()
	result.AddResponseError("test response error")
	s.Contains(result.Response.errors, "test response error")

	result.AddResponseError("another response error")
	s.Contains(result.Response.errors, "another response error")
}

func (s *requestResponseResultTestSuite) TestAddRequest() {
	result := NewRequestResponseResult()
	request := &ocpp.Call{
		MessageTypeId: ocpp.CALL,
		UniqueId:      "1234",
		Action:        "BootNotification",
		Payload:       map[string]interface{}{"chargePointVendor": "Vendor", "chargePointModel": "Model"},
	}
	result.AddRequest(request)

	req, found := result.GetRequest()
	s.True(found)
	s.NotNil(req)
	s.Equal(request, req)
}

func (s *requestResponseResultTestSuite) TestAddResponse() {
	result := NewRequestResponseResult()
	response := &ocpp.CallResult{
		MessageTypeId: ocpp.CALL_RESULT,
		UniqueId:      "1234",
		Payload:       map[string]interface{}{"status": "Accepted"},
	}
	result.AddResponse(response)

	resp, found := result.GetResponse()
	s.True(found)
	s.NotNil(resp)
	s.Equal(response, resp)
}

func (s *requestResponseResultTestSuite) TestAddResponse_CallError() {
	result := NewRequestResponseResult()
	response := &ocpp.CallError{
		MessageTypeId:    ocpp.CALL_ERROR,
		UniqueId:         "1234",
		ErrorCode:        "GenericError",
		ErrorDescription: "An error occurred",
	}
	result.AddResponse(response)

	resp, found := result.GetResponse()
	s.True(found)
	s.NotNil(resp)
	s.Equal(response, resp)
}

func (s *requestResponseResultTestSuite) TestIsValid() {
	tests := []struct {
		name     string
		result   *RequestResponseResult
		expected bool
	}{
		{
			name:     "Valid result with no errors",
			result:   NewRequestResponseResult(),
			expected: true,
		},
		{
			name: "Invalid result with request error",
			result: &RequestResponseResult{
				Request: Result{
					isValid: false,
					errors:  []string{"Request error"},
				},
				Response: *NewResult(),
			},
		},
		{
			name: "Invalid result with response error",
			result: &RequestResponseResult{
				Request: *NewResult(),
				Response: Result{
					isValid: false,
					errors:  []string{"Response error"},
				},
			},
		},
		{
			name: "Both invalid request and response",
			result: &RequestResponseResult{
				Request: Result{
					isValid: false,
					errors:  []string{"Request error"},
				},
				Response: Result{
					isValid: false,
					errors:  []string{"Response error"},
				},
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expected, tt.result.IsValid())
		})
	}
}

func (s *requestResponseResultTestSuite) TestGetRequest() {
	s.T().Run("Request present", func(t *testing.T) {
		result := NewRequestResponseResult()
		request := &ocpp.Call{
			MessageTypeId: ocpp.CALL,
			UniqueId:      "1234",
			Action:        "BootNotification",
			Payload:       map[string]interface{}{"chargePointVendor": "Vendor", "chargePointModel": "Model"},
		}
		result.AddRequest(request)

		req, found := result.GetRequest()
		s.True(found)
		s.NotNil(req)
		s.Equal(request, req)
	})

	s.T().Run("Request not present", func(t *testing.T) {
		result := NewRequestResponseResult()
		req, found := result.GetRequest()
		s.False(found)
		s.Nil(req)
	})
}

func (s *requestResponseResultTestSuite) TestGetResponse() {
	s.T().Run("Response present", func(t *testing.T) {
		result := NewRequestResponseResult()
		response := &ocpp.CallResult{
			MessageTypeId: ocpp.CALL_RESULT,
			UniqueId:      "1234",
			Action:        "BootNotification",
			Payload:       map[string]interface{}{"status": "Accepted"},
		}
		result.AddResponse(response)

		resp, found := result.GetResponse()
		s.True(found)
		s.NotNil(resp)
		s.Equal(response, resp)
	})

	s.T().Run("Response not present", func(t *testing.T) {
		result := NewRequestResponseResult()
		resp, found := result.GetResponse()
		s.False(found)
		s.Nil(resp)
	})
}

func TestParserResult(t *testing.T) {
	suite.Run(t, new(resultTestSuite))
	suite.Run(t, new(requestResponseResultTestSuite))
}
