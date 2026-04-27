package validator

import (
	"context"
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
// It also validates the payload against the schema for the given OCPP version.
// When OcppContext.Vendor and/or OcppContext.Model are non-empty the registry attempts a
// vendor/model-specific schema first, falling back to the base OCPP spec schema.
func (v *Validator) ValidateMessage(octx ocpp.OcppContext, message ocpp.Message) (*ValidationResult, error) {
	logger := v.logger.With(zap.String("vendor", octx.Vendor), zap.String("model", octx.Model), zap.String("action", message.GetAction()))
	logger.Info("Validating message")

	result := NewValidationResult()

	// Check if a message has a unique ID
	uniqueId := message.GetUniqueId()
	if uniqueId == "" {
		result.AddError(uniqueIdEmptyErr)
	}

	payload := message.GetPayload()

	switch message.GetMessageTypeId() {
	case ocpp.CALL:
		action := message.GetAction()
		if action == "" {
			result.AddError(actionEmptyErr)
			break
		}

		err := v.validatePayload(octx, payload, action+"Request", result)
		if err != nil {
			return result, errors.Wrap(err, "unable to validate message payload")
		}

	case ocpp.SEND:
		if octx.Version != ocpp.V21 {
			result.AddError("SEND messages are only supported in OCPP 2.1")
			return result, nil
		}

		action := message.GetAction()
		if action == "" {
			result.AddError(actionEmptyErr)
			break
		}

		err := v.validatePayload(octx, payload, action+"Request", result)
		if err != nil {
			return result, errors.Wrap(err, "unable to validate message payload")
		}

	case ocpp.CALL_RESULT:
		action := message.GetAction()
		if action == "" {
			result.AddError(actionEmptyErr)
		}

		err := v.validatePayload(octx, payload, action+"Response", result)
		if err != nil {
			return result, errors.Wrap(err, "unable to validate message payload")
		}

	case ocpp.CALL_ERROR:
		callError, ok := message.(*ocpp.CallError)
		if !ok {
			return result, ErrCannotCastToCallError
		}

		if !ocpp.IsErrorCodeValid(callError.ErrorCode) {
			result.AddError(fmt.Sprintf("invalid error code: %s", callError.ErrorCode))
		}

	case ocpp.CALL_RESULT_ERROR:
		if octx.Version != ocpp.V21 {
			result.AddError("CALL_RESULT_ERROR messages are only supported in OCPP 2.1")
			return result, nil
		}

		callError, ok := message.(*ocpp.CallResultError)
		if !ok {
			return result, ErrCannotCastToCallError
		}

		if !ocpp.IsErrorCodeValid(callError.ErrorCode) {
			result.AddError(fmt.Sprintf("invalid error code: %s", callError.ErrorCode))
		}
	}

	return result, nil
}

func (v *Validator) validatePayload(
	octx ocpp.OcppContext,
	payload interface{},
	action string,
	validationResults *ValidationResult,
) error {
	if payload == nil {
		validationResults.AddError(payloadEmptyErr)
		return nil
	}

	schema, found := v.registry.GetSchema(context.Background(), schema_registry.GetSchemaRequest{
		OcppContext: octx,
		Action:      action,
	})
	if !found {
		return errors.Errorf("no schema found for action %s in OCPP version %s", action, octx.Version)
	}

	evaluationResult := schema.Validate(payload)
	if !evaluationResult.IsValid() {
		for _, evaluationError := range evaluationResult.Errors {
			validationResults.AddError(evaluationError.Error())
		}
	}

	return nil
}
