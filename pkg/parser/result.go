package parser

import "github.com/ChargePi/chargeflow/pkg/ocpp"

type Result struct {
	message ocpp.Message
	isValid bool
	errors  []string
}

// NewResult creates a new Result with the given validity and errors.
func NewResult() *Result {
	return &Result{
		isValid: true,
		errors:  []string{},
	}
}

func (v *Result) AddError(err string) {
	if v.isValid != false {
		v.isValid = false
	}

	v.errors = append(v.errors, err)
}

func (v *Result) IsValid() bool {
	return v.isValid
}

func (v *Result) Errors() []string {
	return v.errors
}

func (v *Result) Message() ocpp.Message {
	return v.message
}

func (v *Result) SetMessage(message ocpp.Message) {
	v.message = message
}

type RequestResponseResult struct {
	// Request is the parsed OCPP request message.
	Request Result

	// Response can be either ocpp.CallResult or ocpp.CallError (or nil if no request was provided).
	Response Result

	// ResponseError
	ResponseError Result
}

func NewRequestResponseResult() *RequestResponseResult {
	return &RequestResponseResult{
		Request:  *NewResult(),
		Response: *NewResult(),
	}
}

func (r *RequestResponseResult) AddRequestError(err string) {
	r.Request.AddError(err)
}

func (r *RequestResponseResult) AddResponseError(err string) {
	r.Response.AddError(err)
}

func (r *RequestResponseResult) AddResponseErrorError(err string) {
	r.ResponseError.AddError(err)
}

func (r *RequestResponseResult) AddRequest(request ocpp.Message) {
	r.Request.SetMessage(request)
}

func (r *RequestResponseResult) AddResponse(response ocpp.Message) {
	r.Response.SetMessage(response)
}

func (r *RequestResponseResult) AddResponseErrorResult(response ocpp.Message) {
	r.ResponseError.SetMessage(response)
}

func (r *RequestResponseResult) IsValid() bool {
	return r.Request.IsValid() && r.Response.IsValid()
}

func (r *RequestResponseResult) GetRequest() (ocpp.Message, bool) {
	return r.Request.message, r.Request.message != nil
}

func (r *RequestResponseResult) GetResponse() (ocpp.Message, bool) {
	return r.Response.message, r.Response.message != nil
}
