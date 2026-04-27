package validator

const (
	payloadEmptyErr  = "payload is empty"
	actionEmptyErr   = "action is empty"
	uniqueIdEmptyErr = "unique id is empty"
)

type ValidationResult struct {
	isValid bool
	errors  []string
}

// NewValidationResult creates a new ValidationResult with the given validity and errors.
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		isValid: true,
		errors:  []string{},
	}
}

func (v *ValidationResult) AddError(err string) {
	v.isValid = false
	v.errors = append(v.errors, err)
}

// IsValid returns true if the validation result is valid, false otherwise.
func (v *ValidationResult) IsValid() bool {
	return v.isValid
}

// Errors returns a list of errors collected during validation.
func (v *ValidationResult) Errors() []string {
	return v.errors
}
