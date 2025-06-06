package validator

import (
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/parser"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
	"github.com/pkg/errors"
)

type Validator struct {
	registry *schema_registry.SchemaRegistry
}

func NewValidator(registry *schema_registry.SchemaRegistry) *Validator {
	return &Validator{
		registry: registry,
	}
}

// ValidateMessage validates the message. It checks if the message has an action, a payload, and a unique ID.
// It also validates the payload against the schema for the given action and OCPP version.
func (v *Validator) ValidateMessage(ocppVersion ocpp.Version, message parser.Message) (*ValidationResult, error) {
	result := NewValidationResult()

	// Check if a message has a unique ID
	uniqueId := message.GetUniqueId()
	if uniqueId == "" {
		result.AddError(uniqueIdEmptyErr)
	}

	payload := message.GetPayload()

	switch message.GetMessageTypeId() {
	case parser.CALL:
		// Check if a message has an action
		action := message.GetAction()
		if action == "" {
			result.AddError(actionEmptyErr)
		}

		// Check if a message has a payload
		if payload == nil {
			result.AddError(payloadEmptyErr)
			break
		}

		// For CALL messages, the action must end with "Request"
		action = action + "Request"

		err := v.validatePayload(ocppVersion, payload, action, result)
		if err != nil {
			return result, errors.Wrap(err, "unable to validate message payload")
		}

	case parser.CALL_RESULT:
		// Check if a message has an action
		action := message.GetAction()
		if action == "" {
			result.AddError(actionEmptyErr)
		}

		// Todo try the brute force approach
		// Getting through all the schemas and checking if it matches one of them
		// Another approach would be to validate against a matching UniqueId action

		// For CALL_RESULT messages, the action must end with "Response"
		action = action + "Response"

		// Check if a message has a payload
		if payload == nil {
			result.AddError(payloadEmptyErr)
			break
		}

		err := v.validatePayload(ocppVersion, payload, action, result)
		if err != nil {
			return result, errors.Wrap(err, "unable to validate message payload")
		}

	case parser.CALL_ERROR:
		// errors are not validated against schemas, so we skip validation for CALL_ERROR messages
		// We will however validate the contents of the error message
		if payload != nil {
			callError := message.(parser.CallError)
		}
	}

	return result, nil
}

func (v *Validator) validatePayload(ocppVersion ocpp.Version, payload interface{}, action string, validationResults *ValidationResult) error {
	// Get the schema for the action and OCPP version
	schema, found := v.registry.GetSchema(ocppVersion, action)
	if !found {
		return errors.Errorf("no schema found for action %s in OCPP version %s", action, ocppVersion)
	}

	// Validate the payload against the schema
	evaluationResult := schema.Validate(payload)

	if !evaluationResult.IsValid() {
		// Append each validation error to the validation results
		for _, evaluationError := range evaluationResult.Errors {
			validationResults.AddError(evaluationError.Error())
		}
	}

	return nil
}
