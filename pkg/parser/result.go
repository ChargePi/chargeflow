package parser

import "github.com/ChargePi/chargeflow/pkg/ocpp"

type Result struct {
	isValid bool
	errors  []string
}

// NewResult creates a new ValidationResult with the given validity and errors.
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

type RequestResponsePairResult struct {
	// Request is the parsed OCPP request message.
	Request ocpp.Message

	// Response can be either ocpp.CallResult or ocpp.CallError (or nil if no request was provided).
	Response ocpp.Message

	// Result contains the parsing validation result.
	Result Result
}
