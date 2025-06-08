package cmd

import (
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func Test_registerAdditionalSchemas(t *testing.T) {
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
			tmpDefault := defaultOcppVersion
			defaultOcppVersion = test.defaultOcppVersion

			t.Cleanup(func() {
				defaultOcppVersion = tmpDefault
			})

			r := require.New(t)

			// Setup: Create a temporary directory with files
			tempDir := t.TempDir()
			err := os.WriteFile(filepath.Join(tempDir, test.fileName), []byte(test.schema), 0644)
			r.NoError(err)

			// Call the function to register additional schemas
			err = registerAdditionalSchemas(tempDir)
			if test.expected != nil {
				assert.ErrorContains(t, err, test.expected.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_registerSchemas(t *testing.T) {

}

func Test_Validate(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		expectedErr error
	}{
		{
			name: "",
		},
		{
			name: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := require.New(t)

			// Setup: Set command arguments and flags
			validate.SetArgs(test.args)

			for flag, value := range test.flags {
				err := validate.Flags().Set(flag, value)
				r.NoError(err)
			}

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
