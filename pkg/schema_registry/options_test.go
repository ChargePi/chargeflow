package schema_registry

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type optionsTestSuite struct {
	suite.Suite
}

func (s *optionsTestSuite) TestNewoptions() {

}

func TestRegistrySchemaOptions(t *testing.T) {
	suite.Run(t, new(optionsTestSuite))
}
