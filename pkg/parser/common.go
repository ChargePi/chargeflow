package parser

import "encoding/json"

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
