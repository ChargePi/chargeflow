package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ChargePi/chargeflow/pkg/report"
)

func TestTXTStrategy_Write(t *testing.T) {
	dir, err := os.MkdirTemp("", "txt-strat-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "out.txt")

	r := &report.Report{
		InvalidMessages: map[string]map[string][]string{
			"mX": {
				"response": []string{"rerr"},
			},
		},
		NonParsableMessages: map[string][]string{"ln": {"parseerr"}},
		Statistics:          report.Statistics{ValidRequests: 0, InvalidRequests: 0, ValidResponses: 0, InvalidResponses: 1, UnparsableMessages: 1},
	}

	s := txtStrategy{}
	require.NoError(t, s.Write(path, r))

	b, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(b)
	require.Contains(t, content, "Invalid responses")
	require.Contains(t, content, "mX")
}
