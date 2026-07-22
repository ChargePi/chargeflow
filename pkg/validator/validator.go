package validator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocmf"
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
)

const (
	// meterValuesAction is the OCPP action name for MeterValues.req.
	meterValuesAction = "MeterValues"
	// transactionEventAction is the OCPP 2.0.1/2.1 action name for TransactionEvent.req,
	// which carries an optional top-level "meterValue" array of the same shape as
	// MeterValues.req rather than always requiring a dedicated MeterValues.req.
	transactionEventAction = "TransactionEvent"
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

		switch {
		case octx.Version == ocpp.V16 && action == meterValuesAction:
			v.validateOCMFSampledValues(payload, result)
		case octx.Version == ocpp.V20 && (action == meterValuesAction || action == transactionEventAction):
			v.validateOCMFSignedMeterValues(payload, result)
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

// decodeMeterValuesPayload re-decodes a generically-parsed payload carrying a top-level
// "meterValue" array (MeterValues.req, and OCPP 2.0.1/2.1's TransactionEvent.req) into
// OCPP's own MeterValue/SampledValue/SignedMeterValue types so callers can work with
// named fields instead of raw map lookups.
func decodeMeterValuesPayload(payload interface{}) (*ocpp.MeterValuesRequest, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal MeterValues payload")
	}

	var decoded ocpp.MeterValuesRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, errors.Wrap(err, "unable to decode MeterValues payload")
	}

	return &decoded, nil
}

// validateOCMFSampledValues scans an OCPP 1.6 MeterValues.req payload for
// sampledValue.value entries carrying an OCMF-formatted signed meter reading and
// validates each one found against the OCMF JSON Schema, in addition to the regular
// OCPP schema validation.
func (v *Validator) validateOCMFSampledValues(payload interface{}, validationResults *ValidationResult) {
	logger := v.logger.Named("ocmf")

	decoded, err := decodeMeterValuesPayload(payload)
	if err != nil {
		logger.Debug("unable to decode MeterValues payload for OCMF detection", zap.Error(err))
		return
	}

	for _, meterValue := range decoded.MeterValue {
		for _, sampledValue := range meterValue.SampledValue {
			value, ok := sampledValue.Value.(string)
			if !ok || !ocmf.LooksLikeOCMF(value) {
				continue
			}

			logger.Debug("found OCMF-formatted sampledValue.value")
			v.validateOCMFRecord(logger, value, validationResults)
		}
	}
}

// validateOCMFSignedMeterValues scans an OCPP 2.0.1 MeterValues.req or TransactionEvent.req
// payload for sampledValue.signedMeterValue entries whose encodingMethod is "OCMF".
// signedMeterData is base64-encoded per the OCPP spec (Figure 2 / Table 12) and must be
// decoded before it can be validated as an OCMF record.
func (v *Validator) validateOCMFSignedMeterValues(payload interface{}, validationResults *ValidationResult) {
	logger := v.logger.Named("ocmf")

	decoded, err := decodeMeterValuesPayload(payload)
	if err != nil {
		logger.Debug("unable to decode MeterValues payload for OCMF detection", zap.Error(err))
		return
	}

	for _, meterValue := range decoded.MeterValue {
		for _, sampledValue := range meterValue.SampledValue {
			signed := sampledValue.SignedMeterValue
			if signed == nil || signed.EncodingMethod != ocmf.Header {
				continue
			}

			logger.Debug("found OCMF-encoded signedMeterValue", zap.String("encodingMethod", signed.EncodingMethod))

			raw, err := base64.StdEncoding.DecodeString(signed.SignedMeterData)
			if err != nil {
				logger.Warn("signedMeterValue.signedMeterData is declared as OCMF but is not valid base64", zap.Error(err))
				validationResults.AddError(fmt.Sprintf("signedMeterValue.signedMeterData is declared as OCMF (encodingMethod) but is not valid base64: %s", err))
				continue
			}

			v.validateOCMFRecord(logger, string(raw), validationResults)
		}
	}
}

// validateOCMFRecord validates a single raw OCMF record against the OCMF JSON Schema and
// appends any failures to validationResults.
func (v *Validator) validateOCMFRecord(logger *zap.Logger, record string, validationResults *ValidationResult) {
	evaluationResult, err := ocmf.Validate(record)
	if err != nil {
		logger.Warn("OCMF record could not be parsed", zap.Error(err))
		validationResults.AddError(fmt.Sprintf("sampledValue contains an OCMF record that could not be parsed: %s", err))
		return
	}

	if !evaluationResult.IsValid() {
		logger.Debug("OCMF record failed schema validation", zap.Int("errors", len(evaluationResult.Errors)))
		for _, evaluationError := range evaluationResult.Errors {
			validationResults.AddError(fmt.Sprintf("OCMF: %s", evaluationError.Error()))
		}
		return
	}

	logger.Debug("OCMF record is valid")
}
