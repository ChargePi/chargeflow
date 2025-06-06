package schema_registry

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type registryTestSuite struct {
	suite.Suite
}

func (s *registryTestSuite) TestRegisterSchema() {

}

func (s *registryTestSuite) TestGetSchema() {

}

func TestRegistry(t *testing.T) {
	suite.Run(t, new(registryTestSuite))
}
