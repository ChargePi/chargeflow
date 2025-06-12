package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChargePi/chargeflow/pkg/schema_registry"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

var validOcppRequest = "[2, \"1234567890\", \"Authorize\", {\"idTag\": \"1234567890\"}]"
var validOcppResponse = "[3, \"1234567890\", {\"idTagInfo\": {\"status\": \"Accepted\"}}]"

func Test_registerAdditionalSchemas(t *testing.T) {
	logger := zap.L()
	registry = schema_registry.NewInMemorySchemaRegistry(logger)

	tests := []struct {
		name               string
		schema             string
		fileName           string
		defaultOcppVersion string
		expected           error
	}{
		{
			name:               "Valid Schema for OCPP 1.6",
			schema:             "{\n \"$schema\": \"http://json-schema.org/draft-04/schema#\",\n \"id\": \"urn:OCPP:1.6:2019:12:AuthorizeRequest\",\n \"title\": \"AuthorizeRequest\",\n \"type\": \"object\",\n \"properties\": {\n \"idTag\": {\n \"type\": \"string\",\n \"maxLength\": 20\n }\n },\n \"additionalProperties\": false,\n \"required\": [\n \"idTag\"\n ]\n}\n",
			fileName:           "AuthorizeRequest.json",
			defaultOcppVersion: ocpp.V16.String(),
			expected:           nil,
		},
		{
			name:               "Invalid Schema for OCPP 1.6 (malformed JSON)",
			schema:             "\n \"$schema\": \"http://json-schema.org/draft-04/schema#\",\n \"id\": \"urn:OCPP:1.6:2019:12:AuthorizeRequest\",\n \"title\": \"AuthorizeRequest\",\n \"type\": \"object\",\n \"properties\": {\n \"idTag\": {\n \"type\": \"string\",\n \"maxLength\": 20\n }\n },\n \"additionalProperties\": false,\n \"required\": [\n \"idTag\"\n ]\n}\n",
			fileName:           "AuthorizeRequest.json",
			defaultOcppVersion: ocpp.V16.String(),
			expected:           errors.New("failed to register additional OCPP schemas"),
		},
		{
			name:               "Invalid file name for OCPP 1.6",
			schema:             "{\n \"$schema\": \"http://json-schema.org/draft-04/schema#\",\n \"id\": \"urn:OCPP:1.6:2019:12:AuthorizeRequest\",\n \"title\": \"AuthorizeRequest\",\n \"type\": \"object\",\n \"properties\": {\n \"idTag\": {\n \"type\": \"string\",\n \"maxLength\": 20\n }\n },\n \"additionalProperties\": false,\n \"required\": [\n \"idTag\"\n ]\n}\n",
			fileName:           "Authorize.json",
			defaultOcppVersion: ocpp.V16.String(),
			expected:           errors.New("action must end with 'Request' or 'Response'"),
		},
		{
			name:               "Invalid OCPP Version",
			schema:             "{\n \"$schema\": \"http://json-schema.org/draft-04/schema#\",\n \"id\": \"urn:OCPP:1.6:2019:12:AuthorizeRequest\",\n \"title\": \"AuthorizeRequest\",\n \"type\": \"object\",\n \"properties\": {\n \"idTag\": {\n \"type\": \"string\",\n \"maxLength\": 20\n }\n },\n \"additionalProperties\": false,\n \"required\": [\n \"idTag\"\n ]\n}\n",
			fileName:           "AuthorizeRequest.json",
			defaultOcppVersion: "invalid_version",
			expected:           errors.New("failed to register additional OCPP schemas"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			viper.Set("ocpp.version", test.defaultOcppVersion)

			r := require.New(t)

			// Setup: Create a temporary directory with files
			tempDir := t.TempDir()
			err := os.WriteFile(filepath.Join(tempDir, test.fileName), []byte(test.schema), 0644)
			r.NoError(err)

			// Call the function to register additional schemas
			err = registerAdditionalSchemas(logger, tempDir)
			if test.expected != nil {
				assert.ErrorContains(t, err, test.expected.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Validate(t *testing.T) {
	l, _ := zap.NewProduction()
	zap.ReplaceGlobals(l)

	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		expectedErr error
	}{
		{
			name:  "OCPP 1.6 Request",
			args:  []string{validOcppRequest},
			flags: map[string]string{},
		},
		{
			name: "OCPP 1.6 Response",
			args: []string{validOcppResponse},
			flags: map[string]string{
				"response-type": "Authorize",
			},
		},
		{
			name:  "Invalid OCPP message",
			args:  []string{"{\"invalid\": \"message\"}"},
			flags: map[string]string{},
		},
		{
			name: "Invalid OCPP version",
			args: []string{validOcppRequest},
			flags: map[string]string{
				"ocpp-version": "invalid_version",
			},
		},
		{
			name:  "Provided response without response-type",
			args:  []string{validOcppResponse},
			flags: map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup: Set command arguments and flags
			args := test.args
			for flag, value := range test.flags {
				args = append(args, fmt.Sprintf("--%s=%s", flag, value))
			}

			validate.SetArgs(args)

			// Execute the command
			err := validate.Execute()
			if test.expectedErr != nil {
				assert.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
