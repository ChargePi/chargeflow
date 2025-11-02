package validation

import (
	"encoding/json"
	"os"

	"github.com/ChargePi/chargeflow/pkg/report"
)

// jsonStrategy implements OutputStrategy for JSON output.
type jsonStrategy struct{}

func (jsonStrategy) Write(path string, r *report.Report) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	if err = os.WriteFile(path, b, 0644); err != nil {
		return err
	}

	return nil
}
