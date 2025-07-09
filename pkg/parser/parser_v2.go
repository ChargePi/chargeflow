package parser

import (
	"fmt"

	"github.com/spf13/viper"

	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

// ParserV2 is used for parsing multiple OCPP-J messages. It is stateful, as it will all results of the parsing process.
// Particularly needed for parsing files that contain request-response pairs, where the response is dependent on the request.
// It will return a map of unique IDs to RequestResponseResult, where RequestResponseResult is a struct that contains the parsed message and any errors that occurred during parsing.
type ParserV2 struct {
	logger *zap.Logger

	// results is a collection of valid parsed messages, indexed by their unique ID.
	// It contains both requests and responses (response can be a message response or an error response).
	results map[string]RequestResponseResult

	// nonParsable messages are messages that cannot proceed validation due to:
	// - Invalid JSON format
	// - Missing unique ID (responses only)
	// - Invalid message type (e.g. not CALL, CALL_RESULT, CALL_ERROR)
	nonParsable map[string]Result
}

func NewParserV2(logger *zap.Logger) *ParserV2 {
	return &ParserV2{
		logger:      logger.Named("file_parser"),
		results:     make(map[string]RequestResponseResult),
		nonParsable: make(map[string]Result),
	}
}

// Parse takes an array of OCPP-J messages and parses them. It returns a map of unique IDs to RequestResponseResult.
func (fp *ParserV2) Parse(data []string) (map[string]RequestResponseResult, map[string]Result, error) {
	if len(data) == 0 {
		fp.logger.Info("No data to parse")
		return fp.results, fp.nonParsable, nil
	}

	// Process each message, but dont return an error if one fails to be parsed
	for i, message := range data {
		logger := fp.logger.With(
			zap.String("message", message),
			zap.Int("line", i+1),
		)
		logger.Info("Parsing message")

		// Parse the message as JSON
		parsedMessage, err := ParseJsonMessage(message)
		if err != nil {
			logger.Error("Failed to parse message", zap.Error(err))
			result := NewResult()
			result.AddError("Message is not a valid OCPP message")
			key := fmt.Sprintf("line %d", i+1)
			fp.nonParsable[key] = *result
			continue
		}

		// Actually parse the message
		fp.parse(i+1, parsedMessage)
	}

	return fp.results, fp.nonParsable, nil
}

// Parses an OCPP-J message. The function expects an array of elements, as contained in the JSON message.
func (fp *ParserV2) parse(index int, arr []interface{}) {
	result := NewResult()
	line := fmt.Sprintf("line %d", index)

	// Checking message fields
	if len(arr) < 3 {
		// Add to non-parsable messages if the message is too short
		result.AddError(fmt.Sprintf("Expected at least 3 elements in the message, got %d", len(arr)))
		fp.nonParsable[line] = *result
		return
	}

	rawTypeId, ok := arr[0].(float64)
	if !ok {
		result.AddError("Expected first element to be a number (message type ID)")
		fp.nonParsable[line] = *result
		return
	}

	typeId := ocpp.MessageType(rawTypeId)
	uniqueId, ok := arr[1].(string)
	if !ok {
		result.AddError("Expected second element to be a string (unique ID)")
		fp.nonParsable[line] = *result
		return
	}

	if uniqueId == "" {
		// Add to non-parsable messages if the unique ID is missing
		result.AddError("Unique ID is missing in the message")
		// Replace the unique ID with the index of the message in the data array
		uniqueId = line
	}

	switch typeId {
	case ocpp.CALL:
		// Check if a result already exists for this message
		if _, exists := fp.results[uniqueId]; !exists {
			fp.results[uniqueId] = RequestResponseResult{
				Request:  *result,
				Response: *NewResult(),
			}
		}

		results := fp.results[uniqueId]

		fp.logger.Debug("Message is of Request type")

		if len(arr) != 4 {
			results.AddRequestError(fmt.Sprintf("Expected 4 elements in the message, got %d", len(arr)))
			break
		}

		action, ok := arr[2].(string)
		if !ok {
			results.AddRequestError("Expected third element to be a string (action)")
			break
		}

		call := ocpp.Call{
			MessageTypeId: ocpp.CALL,
			UniqueId:      uniqueId,
			Action:        action,
			Payload:       arr[3],
		}

		results.AddRequest(&call)
		// Store the results
		fp.results[uniqueId] = results
	case ocpp.CALL_RESULT:
		// Check if a result already exists for this message
		if _, exists := fp.results[uniqueId]; !exists {
			fp.results[uniqueId] = RequestResponseResult{
				Request:  *NewResult(),
				Response: *result,
			}
		}

		results := fp.results[uniqueId]
		fp.logger.Debug("Message is of Response type")

		// Check if response-type is set in global config
		// Note: This can only be used in single message parsing, or if you have responses with the same type
		action := viper.GetString("response-type")

		// Check if we have a request with the same unique ID to determine the response type
		existingResult, exist := fp.results[uniqueId]
		if !exist && action == "" {
			results.AddResponseError("Unable to determine response type for message")
			break
		}

		req, found := existingResult.GetRequest()
		if found {
			action = req.GetAction()
		}

		if action == "" {
			// Nothing to do here, we will use the action from the request
			break
		}

		callResult := ocpp.CallResult{
			MessageTypeId: ocpp.CALL_RESULT,
			UniqueId:      uniqueId,
			Action:        action,
			Payload:       arr[2],
		}

		results.AddResponse(&callResult)
		// Store the results
		fp.results[uniqueId] = results
	case ocpp.CALL_ERROR:
		// Check if a result already exists for this message
		if _, exists := fp.results[uniqueId]; !exists {
			fp.results[uniqueId] = RequestResponseResult{
				Request:  *NewResult(),
				Response: *result,
			}
		}

		results := fp.results[uniqueId]
		fp.logger.Debug("Message is of Error response type")

		if len(arr) < 4 {
			results.AddResponseError("Invalid Call Error message. Expected array length >= 4, got " + fmt.Sprintf("%d", len(arr)))
			break
		}

		var details interface{}
		if len(arr) > 4 {
			details = arr[4]
		}

		rawErrorCode, ok := arr[2].(string)
		if !ok {
			results.AddResponseError(fmt.Sprintf("Invalid element %v at 2, expected error code (string)", arr[2]))
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

		results.AddResponse(&callError)
		// Store the results
		fp.results[uniqueId] = results
	default:
		fp.logger.Error("Unknown message type", zap.Int("typeId", int(typeId)))
		result.AddError(fmt.Sprintf("Unknown message type: %d", typeId))
		fp.nonParsable[uniqueId] = *result
	}
}
