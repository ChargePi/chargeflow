package report

import (
	"github.com/ChargePi/chargeflow/pkg/parser"
	"github.com/ChargePi/chargeflow/pkg/validator"
)

type Report struct {
	// InvalidMessages contains all the errors per message (request or response)
	InvalidMessages     map[string]map[string][]string `json:"invalid_messages"`
	NonParsableMessages map[string][]string            `json:"non_parsable_messages"`
}

type Results struct {
	validator.ValidationResult
	parser.Result
}
