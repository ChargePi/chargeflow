package file

type fileRegistryOptions struct {
	// Whether to allow overwriting existing schemasPerOcppVersion in the registry or not.
	overwrite bool
}

type RegistryOption func(*fileRegistryOptions)

// WithOverwrite allows overwriting existing schemasPerOcppVersion in the registry
func WithOverwrite(overwrite bool) RegistryOption {
	return func(o *fileRegistryOptions) {
		o.overwrite = overwrite
	}
}
