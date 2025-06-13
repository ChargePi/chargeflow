package parser

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type resultTestSuite struct {
	suite.Suite
}

func (s *resultTestSuite) TestAddError() {
	result := NewResult()
	result.AddError("test error")
	s.Contains(result.Errors(), "test error")

	result.AddError("another error")
	s.Contains(result.Errors(), "another error")
}

func (s *resultTestSuite) TestIsValid() {
	result := NewResult()
	s.True(result.IsValid())

	result.AddError("test error")
	s.False(result.IsValid())

	result.AddError("test 2 error")
	s.False(result.IsValid())
}

func (s *resultTestSuite) TestErrors() {
	result := NewResult()
	s.Empty(result.Errors())

	result.AddError("test error")
	result.AddError("another error")
	s.Equal([]string{"test error", "another error"}, result.Errors())
}

func TestParserResult(t *testing.T) {
	suite.Run(t, new(resultTestSuite))
}
