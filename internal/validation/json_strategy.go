package validation

import (
	"encoding/json"
	"os"

	"github.com/ChargePi/chargeflow/pkg/report"
)

// jsonStrategy implements OutputStrategy for JSON output.
type jsonStrategy struct{}

func (jsonStrategy) Write(path string, r *report.Report) error {
	out := struct {
		Report     *report.Report    `json:"report"`
		Statistics report.Statistics `json:"statistics"`
	}{
		Report:     r,
		Statistics: r.Statistics,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}

	if err = os.WriteFile(path, b, 0644); err != nil {
		return err
	}

	return nil
}
