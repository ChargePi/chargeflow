package validation

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputStrategyFactory(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"json", "a.json", false},
		{"csv", "b.csv", false},
		{"txt", "c.txt", false},
		{"bad", "d.unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strat, err := outputStrategyFactory(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// ensure strategy concrete type via extension
			ext := filepath.Ext(tt.path)
			switch ext {
			case ".json":
				require.IsType(t, jsonStrategy{}, strat)
			case ".csv":
				require.IsType(t, csvStrategy{}, strat)
			case ".txt":
				require.IsType(t, txtStrategy{}, strat)
			}
		})
	}
}
