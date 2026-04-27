package validation

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/ChargePi/chargeflow/pkg/report"
)

// jsonStrategy implements ReportWriter for JSON output.
type jsonStrategy struct{}

func (jsonStrategy) Write(path string, r *report.Report) error {
	if r == nil {
		return errors.New("report is nil")
	}

	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	if err = os.WriteFile(path, b, 0644); err != nil {
		return err
	}

	return nil
}
