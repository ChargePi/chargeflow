package parser

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type messageParsingTestSuite struct {
	suite.Suite
}

func (s *messageParsingTestSuite) TestParseJsonMessage() {
	// Example raw JSON message
	rawMessage := `[0, "123456", "BootNotification", {"key1": "value1", "key2": "value2"}]`

	// Call the function to parse the raw JSON message
	parsedMessage, err := ParseJsonMessage(rawMessage)
	s.NoError(err)

	// Check if the parsed message is as expected
	expectedMessage := []interface{}{float64(0), "123456", "BootNotification", map[string]interface{}{"key1": "value1", "key2": "value2"}}
	s.EqualValues(expectedMessage, parsedMessage)
}

func (s *messageParsingTestSuite) TestParseJsonMessage_Unhappy() {
	// Example raw JSON message
	rawMessage := `0, "123456", "BootNotification", {"key1": "value1", "key2": "value2"}]`

	// Call the function to parse the raw JSON message
	parsedMessage, err := ParseJsonMessage(rawMessage)
	s.Error(err)
	s.Nil(parsedMessage)
}

func (s *messageParsingTestSuite) TestParseRawJsonMessage() {
	// Example raw JSON message
	rawMessage := `[0, "123456", "BootNotification", {"key1": "value1", "key2": "value2"}]`

	// Call the function to parse the raw JSON message
	parsedMessage, err := ParseRawJsonMessage([]byte(rawMessage))
	s.NoError(err)

	// Check if the parsed message is as expected
	expectedMessage := []interface{}{float64(0), "123456", "BootNotification", map[string]interface{}{"key1": "value1", "key2": "value2"}}
	s.EqualValues(expectedMessage, parsedMessage)
}

func (s *messageParsingTestSuite) TestParseRawJsonMessage_Unhappy() {
	// Example raw JSON message
	rawMessage := `0, "123456", "BootNotification", {"key1": "value1", "key2": "value2"}]`

	// Call the function to parse the raw JSON message
	parsedMessage, err := ParseRawJsonMessage([]byte(rawMessage))
	s.Error(err)
	s.Nil(parsedMessage)
}

func TestMessageParsing(t *testing.T) {
	suite.Run(t, new(messageParsingTestSuite))
}
