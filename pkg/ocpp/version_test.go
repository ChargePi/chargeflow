package ocpp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOcppVersionValid(t *testing.T) {
	tests := []struct {
		name    string
		version Version
		want    bool
	}{
		{
			name:    "valid version 1.5",
			version: V15,
			want:    true,
		},
		{
			name:    "valid version 1.6",
			version: V16,
			want:    true,
		},
		{
			name:    "valid version 2.0",
			version: V20,
			want:    true,
		},
		{
			name:    "valid version 2.1",
			version: V20,
			want:    true,
		},
		{
			name:    "invalid version",
			version: "OCPP2.2",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidProtocolVersion(tt.version))
		})
	}
}
