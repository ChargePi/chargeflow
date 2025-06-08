package parser

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

type Parser struct {
	logger *zap.Logger
}

func NewParser(logger *zap.Logger) *Parser {
	return &Parser{
		logger: logger.Named("parser"),
	}
}

func (p *Parser) ParseMessage(data string) (ocpp.Message, *Result, error) {
	p.logger.Debug("Parsing message from JSON", zap.String("data", data))

	result := NewResult()

	message, err := ParseJsonMessage(data)
	if err != nil {
		result.AddError("cannot parse message")
		return nil, result, errors.Wrap(err, "cannot parse message")
	}

	p.logger.Debug("Deconstructing the message", zap.Any("message", message))

	// Validate the message (action, unique ID)
	parse, err := p.parse(message, result)
	if err != nil {
		return nil, result, errors.Wrapf(err, "cannot parse message")
	}

	return parse, result, nil
}

// Parses an OCPP-J message. The function expects an array of elements, as contained in the JSON message.
func (p *Parser) parse(arr []interface{}, result *Result) (ocpp.Message, error) {
	// Checking message fields
	if len(arr) < 3 {
		result.AddError(fmt.Sprintf("Expected at least 3 elements in the message, got %d", len(arr)))
		return nil, nil
	}

	rawTypeId, ok := arr[0].(float64)
	if !ok {
		result.AddError("Expected first element to be a number (message type ID)")
	}

	typeId := ocpp.MessageType(rawTypeId)
	uniqueId, ok := arr[1].(string)
	if !ok {
		result.AddError("Expected second element to be a string (unique ID)")
	}

	switch typeId {
	case ocpp.CALL:
		p.logger.Debug("Message is of Request type")

		if len(arr) != 4 {
			result.AddError(fmt.Sprintf("Expected 4 elements in the message, got %d", len(arr)))
			return nil, errors.Errorf("Expected 4 elements in the message, got %d", len(arr))
		}

		action, ok := arr[2].(string)
		if !ok {
			result.AddError("Expected second element to be a string (action ID)")
			return nil, errors.Errorf("Expected second element to be a string (action ID), got %v", arr[2])
		}

		call := ocpp.Call{
			MessageTypeId: ocpp.CALL,
			UniqueId:      uniqueId,
			Action:        action,
			Payload:       arr[3],
		}
		return &call, nil
	case ocpp.CALL_RESULT:
		p.logger.Debug("Message is of Response type")
		callResult := ocpp.CallResult{
			MessageTypeId: ocpp.CALL_RESULT,
			UniqueId:      uniqueId,
			Payload:       arr[3],
		}
		return &callResult, nil
	case ocpp.CALL_ERROR:
		p.logger.Debug("Message is of Error response type")

		if len(arr) < 4 {
			result.AddError("Invalid Call Error message. Expected array length >= 4, got " + fmt.Sprintf("%d", len(arr)))
			return nil, errors.Errorf("Invalid Call Error message. Expected array length >= 4, got %v", arr[2])
		}

		var details interface{}
		if len(arr) > 4 {
			details = arr[4]
		}

		rawErrorCode, ok := arr[2].(string)
		if !ok {
			result.AddError(fmt.Sprintf("Invalid element %v at 2, expected error code (string)", arr[2]))
		}

		errorCode := ocpp.ErrorCode(rawErrorCode)
		errorDescription := ""
		if v, ok := arr[3].(string); ok {
			errorDescription = v
		}
		callError := ocpp.CallError{
			MessageTypeId:    ocpp.CALL_ERROR,
			UniqueId:         uniqueId,
			ErrorCode:        errorCode,
			ErrorDescription: errorDescription,
			ErrorDetails:     details,
		}
		return &callError, nil
	default:
		p.logger.Error("Unknown message type", zap.String("typeId", fmt.Sprintf("%v", typeId)))
		result.AddError("Unknown message type: " + fmt.Sprintf("%v", typeId))
		return nil, errors.Errorf("Unknown message type: %v ", typeId)
	}
}

// ParseRawJsonMessage Unmarshals an OCPP-J json object from a byte array.
// Returns the array of elements contained in the message.
func ParseRawJsonMessage(dataJson []byte) ([]interface{}, error) {
	var arr []interface{}
	err := json.Unmarshal(dataJson, &arr)
	if err != nil {
		return nil, err
	}
	return arr, nil
}

// ParseJsonMessage Unmarshals an OCPP-J json object from a JSON string.
// Returns the array of elements contained in the message.
func ParseJsonMessage(dataJson string) ([]interface{}, error) {
	rawJson := []byte(dataJson)
	return ParseRawJsonMessage(rawJson)
}
