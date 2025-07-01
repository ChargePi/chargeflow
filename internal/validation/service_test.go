package validation

import (
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"

	mock_schema_registry "github.com/ChargePi/chargeflow/gen/mocks/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/ocpp"

	"github.com/stretchr/testify/suite"
)

var (
	validReq      = `[2, "1234", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]`
	validRes      = `[3, "1234", {"status": "Accepted"}]`
	invalidRes    = `[4, "1234", {"errorCode": "GenericError1", "errorDescription": "An error occurred"}]`
	invalidReq    = `[2, "1234", InvalidRequest", {"errorCode": "GenericError", "errorDescription": "An error occurred"}]`
	unparsableMsg = `{"invalid": "json"}`
)

type validationServiceTestSuite struct {
	suite.Suite
	filePaths map[string]string
	logger    *zap.Logger
}

func (s *validationServiceTestSuite) createFiles() {
	temp, err := os.CreateTemp(".", "valid_*.txt")
	s.Require().NoError(err)

	err = writeToFile(temp.Name(), strings.Join([]string{validReq, validRes, invalidRes, invalidReq, unparsableMsg}, "\n"))
	s.Require().NoError(err)

	s.filePath = temp.Name()
}

func writeToFile(filePath string, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

func (s *validationServiceTestSuite) SetupSuite() {
	s.logger = zap.NewExample()
	s.createFiles()
}

func (s *validationServiceTestSuite) TearDownSuite() {
	for _, path := range s.filePaths {
		_ = os.Remove(path)
	}
}

func (s *validationServiceTestSuite) TestValidateFile() {
	tests := []struct {
		name            string
		filepath        string
		version         ocpp.Version
		setExpectations func(*mock_schema_registry.MockSchemaRegistry)
		expectedErr     error
	}{
		{
			name: "Valid file with version 1.6",
		},
		{
			name: "Valid file with version 2.0",
		},
		{
			name: "Invalid version",
		},
		{
			name: "Non-existent file",
		},
		{
			name: "Invalid messages",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registry := mock_schema_registry.NewMockSchemaRegistry(s.T())
			if tt.setExpectations != nil {
				tt.setExpectations(registry)
			}

			service := NewService(s.logger, registry)
			err := service.ValidateFile(tt.filepath, tt.version)
			if tt.expectedErr != nil {
				s.ErrorContains(err, tt.expectedErr.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *validationServiceTestSuite) TestValidateMessage() {
	tests := []struct {
		name            string
		message         string
		version         ocpp.Version
		setExpectations func(*mock_schema_registry.MockSchemaRegistry)
		expectedErr     error
	}{
		{
			name: "Valid message with version 1.6",
		},
		{
			name: "Valid message with version 2.0",
		},
		{
			name: "Invalid version",
		},
		{
			name: "Unparsable message",
		},
		{
			name: "Invalid message",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registry := mock_schema_registry.NewMockSchemaRegistry(s.T())
			if tt.setExpectations != nil {
				tt.setExpectations(registry)
			}

			service := NewService(s.logger, registry)

			err := service.ValidateMessage(tt.message, tt.version)
			if tt.expectedErr != nil {
				s.ErrorContains(err, tt.expectedErr.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func TestValidationService(t *testing.T) {
	suite.Run(t, new(validationServiceTestSuite))
}
