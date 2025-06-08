package parser

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type parserTestSuite struct {
	suite.Suite
}

func (s *parserTestSuite) TestParseMessage() {

}

func TestParser(t *testing.T) {
	suite.Run(t, new(parserTestSuite))
}
