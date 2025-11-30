package validator

import (
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
)

var ErrCannotCastToCallError = errors.New("cannot cast message to CallError")

type Validator struct {
	logger   *zap.Logger
	registry schema_registry.SchemaRegistry
}

func NewValidator(logger *zap.Logger, registry schema_registry.SchemaRegistry) *Validator {
	return &Validator{
		logger:   logger.Named("validator"),
		registry: registry,
	}
}

// ValidateMessage validates the message. It checks if the message has an action, a payload, and a unique ID.
// It also validates the payload against the schema for the given action and OCPP version.
func (v *Validator) ValidateMessage(ocppVersion ocpp.Version, message ocpp.Message) (*ValidationResult, error) {
	v.logger.Info("Validating message", zap.String("action", message.GetAction()))
	result := NewValidationResult()

	// Check if a message has a unique ID
	uniqueId := message.GetUniqueId()
	if uniqueId == "" {
		result.AddError(uniqueIdEmptyErr)
	}

	payload := message.GetPayload()

	switch message.GetMessageTypeId() {
	case ocpp.CALL:
		// Check if a message has an action
		action := message.GetAction()
		if action == "" {
			result.AddError(actionEmptyErr)
			break
		}

		// For CALL messages, the action must end with "Request"
		action = action + "Request"

		err := v.validatePayload(ocppVersion, payload, action, result)
		if err != nil {
			return result, errors.Wrap(err, "unable to validate message payload")
		}

	case ocpp.SEND:
		if ocppVersion != ocpp.V21 {
			result.AddError("SEND messages are only supported in OCPP 2.1")
			return result, nil
		}

		// Check if a message has an action
		action := message.GetAction()
		if action == "" {
			result.AddError(actionEmptyErr)
			break
		}

		// For CALL messages, the action must end with "Request"
		action = action + "Request"

		err := v.validatePayload(ocppVersion, payload, action, result)
		if err != nil {
			return result, errors.Wrap(err, "unable to validate message payload")
		}
	case ocpp.CALL_RESULT:
		// Check if a message has an action
		action := message.GetAction()
		if action == "" {
			result.AddError(actionEmptyErr)
		}

		// For CALL_RESULT messages, the action must end with "Response"
		action = action + "Response"

		err := v.validatePayload(ocppVersion, payload, action, result)
		if err != nil {
			return result, errors.Wrap(err, "unable to validate message payload")
		}
	case ocpp.CALL_ERROR:
		// errors are not validated against schemas, so we skip validation for CALL_ERROR messages
		// We will however validate the contents of the error message
		callError, ok := message.(*ocpp.CallError)
		if !ok {
			return result, ErrCannotCastToCallError
		}

		// Validate the error code
		if !ocpp.IsErrorCodeValid(callError.ErrorCode) {
			result.AddError(fmt.Sprintf("invalid error code: %s", callError.ErrorCode))
		}
	case ocpp.CALL_RESULT_ERROR:
		if ocppVersion != ocpp.V21 {
			result.AddError("CALL_RESULT_ERROR messages are only supported in OCPP 2.1")
			return result, nil
		}

		// errors are not validated against schemas, so we skip validation for CALL_ERROR messages
		// We will however validate the contents of the error message
		callError, ok := message.(*ocpp.CallResultError)
		if !ok {
			return result, ErrCannotCastToCallError
		}

		// Validate the error code
		if !ocpp.IsErrorCodeValid(callError.ErrorCode) {
			result.AddError(fmt.Sprintf("invalid error code: %s", callError.ErrorCode))
		}
	}

	return result, nil
}

func (v *Validator) validatePayload(ocppVersion ocpp.Version, payload interface{}, action string, validationResults *ValidationResult) error {
	// Check if a message has a payload
	if payload == nil {
		validationResults.AddError(payloadEmptyErr)
		return nil
	}

	switch v.registry.Type() {
	case "remote":
		// For remote, we can validate the payload against the schema directly
	case "local":

	}

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
