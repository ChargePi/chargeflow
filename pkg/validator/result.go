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
	if v.isValid != false {
		v.isValid = false
	}

	v.errors = append(v.errors, err)
}

func (v *ValidationResult) IsValid() bool {
	return v.isValid
}

func (v *ValidationResult) Errors() []string {
	return v.errors
}
