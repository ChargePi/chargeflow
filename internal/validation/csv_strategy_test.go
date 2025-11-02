package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ChargePi/chargeflow/pkg/report"
)

func TestCSVStrategy_Write(t *testing.T) {
	dir, err := os.MkdirTemp("", "csv-strat-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "out.csv")

	r := &report.Report{
		InvalidMessages: map[string]map[string][]string{
			"m1": {
				"request": []string{"e1", "e2"},
			},
		},
		NonParsableMessages: map[string][]string{"p1": {"pe1"}},
		Statistics:          report.Statistics{},
	}

	s := csvStrategy{}
	require.NoError(t, s.Write(path, r))

	b, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(b)
	require.Truef(t, strings.Contains(content, "message_id,type,errors") || strings.Contains(content, "message_id, type, errors"), "csv header missing, got: %s", content)
	require.Contains(t, content, "m1", "expected m1 in csv")
	require.Contains(t, content, "non_parsable", "expected non_parsable in csv")
}
