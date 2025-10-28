package validation

// Option is a functional option for ValidateFile.
type Option func(*options)

type options struct {
	output string
}

// WithOutput sets the output path for the validation report.
func WithOutput(path string) Option {
	return func(o *options) { o.output = path }
}
