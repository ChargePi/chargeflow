package schema_registry

type Options struct {
	// Whether to allow overwriting existing schemasPerOcppVersion in the registry or not.
	overwrite bool
}

type Option func(*Options)

// WithOverwrite allows overwriting existing schemasPerOcppVersion in the registry
func WithOverwrite(overwrite bool) Option {
	return func(o *Options) {
		o.overwrite = overwrite
	}
}
