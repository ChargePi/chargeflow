package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ChargePi/chargeflow/pkg/report"
)

func TestJSONStrategy_Write(t *testing.T) {
	dir, err := os.MkdirTemp("", "json-strat-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "out.json")

	r := &report.Report{
		InvalidMessages: map[string]map[string][]string{
			"msg1": {"request": {"req-err"}},
		},
		NonParsableMessages: map[string][]string{"line1": {"parse-err"}},
		Statistics:          report.Statistics{ValidRequests: 1, InvalidRequests: 1, ValidResponses: 0, InvalidResponses: 0, UnparsableMessages: 1},
	}

	s := jsonStrategy{}
	require.NoError(t, s.Write(path, r))

	b, err := os.ReadFile(path)
	require.NoError(t, err)

	// Unmarshal into a generic map to ensure fields exist
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &out))

	require.Contains(t, out, "report")
	require.Contains(t, out, "statistics")
}
