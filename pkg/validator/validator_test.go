package validator

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type validatorTestSuite struct {
	suite.Suite
}

func (s *validatorTestSuite) TestValidateMessage() {
	tests := []struct {
		name string
	}{}

	for _, tt := range tests {
		s.Run(tt.name, func() {

		})
	}
}

func TestValidator(t *testing.T) {
	suite.Run(t, new(validatorTestSuite))
}
