package validation

import "github.com/ChargePi/chargeflow/pkg/ocpp"

// Request carries all inputs for a single validation run.
type Request struct {
	OcppContext ocpp.OcppContext // OCPP version, vendor, and model for schema selection
	Messages    []string         // inline messages to validate (mutually exclusive with File)
	File        string           // path to a newline-delimited file of messages
	Output      string           // optional path to write the report (.json, .csv, .txt)
}

// Option is a functional option for ValidateFile (kept for backwards compat with callers
// that still build options separately before constructing a ValidationRequest).
type Option func(*options)

type options struct {
	output string
}

// WithOutput sets the output path for the validation report.
func WithOutput(path string) Option {
	return func(o *options) { o.output = path }
}
