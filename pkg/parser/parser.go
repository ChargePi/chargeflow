package parser

import (
	"fmt"
	"go.uber.org/zap"

	"github.com/pkg/errors"
)

type Parser struct {
	logger *zap.Logger
}

func NewParser(logger *zap.Logger) *Parser {
	return &Parser{
		logger: logger.Named("parser"),
	}
}

func (p *Parser) ParseMessage(data string) (Message, error) {
	p.logger.Debug("Parsing message from JSON", zap.String("data", data))

	message, err := ParseJsonMessage(data)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse message")
	}

	p.logger.Debug("Deconstructing the message", zap.Any("message", message))

	// Validate the message (action, unique ID)
	parse, err := p.parse(message)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse message")
	}

	return parse, nil
}

// Parses an OCPP-J message. The function expects an array of elements, as contained in the JSON message.
func (p *Parser) parse(arr []interface{}) (Message, error) {
	// Checking message fields
	if len(arr) < 3 {
		return nil, NewError(FormatErrorType(p), "Invalid message. Expected array length >= 3", "")
	}

	rawTypeId, ok := arr[0].(float64)
	if !ok {
		return nil, NewError(FormatErrorType(p), fmt.Sprintf("Invalid element %v at 0, expected message type (int)", arr[0]), "")
	}

	typeId := MessageType(rawTypeId)
	uniqueId, ok := arr[1].(string)
	if !ok {
		return nil, NewError(FormatErrorType(p), fmt.Sprintf("Invalid element %v at 1, expected unique ID (string)", arr[1]), uniqueId)
	}

	switch typeId {
	case CALL:
		p.logger.Debug("Message is of Request type")

		if len(arr) != 4 {
			return nil, NewError(FormatErrorType(p), "Invalid Call message. Expected array length 4", uniqueId)
		}
		action, ok := arr[2].(string)
		if !ok {
			return nil, NewError(FormatErrorType(p), fmt.Sprintf("Invalid element %v at 2, expected action (string)", arr[2]), uniqueId)
		}

		call := Call{
			MessageTypeId: CALL,
			UniqueId:      uniqueId,
			Action:        action,
			Payload:       arr[3],
		}
		return &call, nil
	case CALL_RESULT:
		p.logger.Debug("Message is of Response type")
		callResult := CallResult{
			MessageTypeId: CALL_RESULT,
			UniqueId:      uniqueId,
			Payload:       arr[3],
		}
		return &callResult, nil
	case CALL_ERROR:
		p.logger.Debug("Message is of Error response type")
		if len(arr) < 4 {
			return nil, NewError(FormatErrorType(p), "Invalid Call Error message. Expected array length >= 4", uniqueId)
		}

		var details interface{}
		if len(arr) > 4 {
			details = arr[4]
		}

		rawErrorCode, ok := arr[2].(string)
		if !ok {
			return nil, NewError(FormatErrorType(p), fmt.Sprintf("Invalid element %v at 2, expected rawErrorCode (string)", arr[2]), rawErrorCode)
		}

		errorCode := ErrorCode(rawErrorCode)
		errorDescription := ""
		if v, ok := arr[3].(string); ok {
			errorDescription = v
		}
		callError := CallError{
			MessageTypeId:    CALL_ERROR,
			UniqueId:         uniqueId,
			ErrorCode:        errorCode,
			ErrorDescription: errorDescription,
			ErrorDetails:     details,
		}
		return &callError, nil
	default:
		return nil, NewError(MessageTypeNotSupported, fmt.Sprintf("Invalid message type ID %v", typeId), uniqueId)
	}
}
