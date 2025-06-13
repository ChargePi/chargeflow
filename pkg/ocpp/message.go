package ocpp

// MessageType identifies the type of message exchanged between two OCPP endpoints.
type MessageType int

const (
	CALL        MessageType = 2
	CALL_RESULT MessageType = 3
	CALL_ERROR  MessageType = 4
)

// An OCPP-J message.
type Message interface {
	// Returns the message type identifier of the message.
	GetMessageTypeId() MessageType
	GetPayload() interface{}
	GetAction() string
	GetUniqueId() string
}

// An OCPP-J Call message, containing an OCPP Request.
type Call struct {
	MessageTypeId MessageType `json:"messageTypeId" validate:"required,eq=2"`
	UniqueId      string      `json:"uniqueId" validate:"required,max=36"`
	Action        string      `json:"action" validate:"required,max=36"`
	Payload       interface{} `json:"payload" validate:"required"`
}

func (call *Call) GetMessageTypeId() MessageType {
	return call.MessageTypeId
}

func (call *Call) GetUniqueId() string {
	return call.UniqueId
}

func (call *Call) GetAction() string {
	return call.Action
}

func (call *Call) GetPayload() interface{} {
	return call.Payload
}

// -------------------- Call Result --------------------

// An OCPP-J CallResult message, containing an OCPP Response.
type CallResult struct {
	MessageTypeId MessageType `json:"messageTypeId"`
	UniqueId      string      `json:"uniqueId"`
	Payload       interface{} `json:"payload"`
	Action        string      `json:"action"`
}

func (callResult *CallResult) GetMessageTypeId() MessageType {
	return callResult.MessageTypeId
}

func (callResult *CallResult) GetUniqueId() string {
	return callResult.UniqueId
}

func (callResult *CallResult) GetAction() string {
	return callResult.Action // CallResult does not have an action, so we return a placeholder.
}

func (callResult *CallResult) GetPayload() interface{} {
	return callResult.Payload
}

// -------------------- Call Error --------------------

// An OCPP-J CallError message, containing an OCPP Error.
type CallError struct {
	MessageTypeId    MessageType `json:"messageTypeId"`
	UniqueId         string      `json:"uniqueId"`
	ErrorCode        ErrorCode   `json:"errorCode"`
	ErrorDescription string      `json:"errorDescription"`
	ErrorDetails     interface{} `json:"errorDetails"`
}

func (callError *CallError) GetMessageTypeId() MessageType {
	return callError.MessageTypeId
}

func (callError *CallError) GetUniqueId() string {
	return callError.UniqueId
}

func (callError *CallError) GetPayload() interface{} {
	return callError.ErrorDetails
}

func (callError *CallError) GetAction() string {
	return string(callError.ErrorCode)
}

type ErrorCode string

const (
	NotImplemented                   ErrorCode = "NotImplemented"                // Requested Action is not known by receiver.
	NotSupported                     ErrorCode = "NotSupported"                  // Requested Action is recognized but not supported by the receiver.
	InternalError                    ErrorCode = "InternalError"                 // An internal error occurred and the receiver was not able to process the requested Action successfully.
	MessageTypeNotSupported          ErrorCode = "MessageTypeNotSupported"       // A message with a Message Type Number received that is not supported by this implementation.
	ProtocolError                    ErrorCode = "ProtocolError"                 // Payload for Action is incomplete.
	SecurityError                    ErrorCode = "SecurityError"                 // During the processing of Action a security issue occurred preventing receiver from completing the Action successfully.
	PropertyConstraintViolation      ErrorCode = "PropertyConstraintViolation"   // Payload is syntactically correct but at least one field contains an invalid value.
	OccurrenceConstraintViolationV2  ErrorCode = "OccurrenceConstraintViolation" // Payload for Action is syntactically correct but at least one of the fields violates occurrence constraints.
	OccurrenceConstraintViolationV16 ErrorCode = "OccurenceConstraintViolation"  // Payload for Action is syntactically correct but at least one of the fields violates occurrence constraints. Contains a typo in OCPP 1.6
	TypeConstraintViolation          ErrorCode = "TypeConstraintViolation"       // Payload for Action is syntactically correct but at least one of the fields violates data type constraints (e.g. “somestring”: 12).
	GenericError                     ErrorCode = "GenericError"                  // Any other error not covered by the previous ones.
	FormatViolationV2                ErrorCode = "FormatViolation"               // Payload for Action is syntactically incorrect. This is only valid for OCPP 2.0.1
	FormatViolationV16               ErrorCode = "FormationViolation"            // Payload for Action is syntactically incorrect or not conform the PDU structure for Action. This is only valid for OCPP 1.6
)

func FormatErrorType(version Version) ErrorCode {
	switch version {
	case V16:
		return FormatViolationV16
	case V20:
		return FormatViolationV2
	default:
		panic("invalid dialect")
	}
}

func OccurrenceConstraintErrorType(version Version) ErrorCode {
	switch version {
	case V16:
		return OccurrenceConstraintViolationV16
	case V20:
		return OccurrenceConstraintViolationV2
	default:
		panic("invalid dialect")
	}
}

func IsErrorCodeValid(code ErrorCode) bool {
	switch code {
	case NotImplemented, NotSupported, InternalError, MessageTypeNotSupported,
		ProtocolError, SecurityError, FormatViolationV16,
		FormatViolationV2, PropertyConstraintViolation, OccurrenceConstraintViolationV16,
		OccurrenceConstraintViolationV2, TypeConstraintViolation, GenericError:
		return true
	}
	return false
}
