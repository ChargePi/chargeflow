package validator

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type resultTestSuite struct {
	suite.Suite
}

func (s *resultTestSuite) TestAddError() {
	result := NewValidationResult()
	s.True(result.isValid)

	result.AddError("test error")
	s.False(result.IsValid())
	s.Contains(result.Errors(), "test error")
}

func (s *resultTestSuite) TestErrors() {
	result := NewValidationResult()
	s.True(result.isValid)

	result.AddError("first error")
	result.AddError("second error")

	errors := result.Errors()
	s.Len(errors, 2)
	s.Contains(errors, "first error")
	s.Contains(errors, "second error")
	s.NotContains(errors, "third error")
}

func TestResult(t *testing.T) {
	suite.Run(t, new(resultTestSuite))
}
