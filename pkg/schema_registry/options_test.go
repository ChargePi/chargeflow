package schema_registry

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type optionsTestSuite struct {
	suite.Suite
}

func (s *optionsTestSuite) TestOptions() {
	tests := []struct {
		name     string
		opts     []Option
		expected Options
	}{
		{
			name: "default options",
			opts: []Option{},
			expected: Options{
				overwrite: false,
			},
		},
		{
			name: "WithOverwrite",
			opts: []Option{
				WithOverwrite(true),
			},
			expected: Options{
				overwrite: true,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			options := &Options{}
			for _, opt := range tt.opts {
				opt(options)
			}
			s.Equal(tt.expected, *options)
		})
	}
}

func TestRegistrySchemaOptions(t *testing.T) {
	suite.Run(t, new(optionsTestSuite))
}
