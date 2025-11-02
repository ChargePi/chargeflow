package validation

import (
	"encoding/csv"
	"os"
	"strings"

	"github.com/ChargePi/chargeflow/pkg/report"
)

// csvStrategy implements OutputStrategy for CSV output.
type csvStrategy struct{}

func (csvStrategy) Write(path string, r *report.Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	if err = w.Write([]string{"message_id", "type", "errors"}); err != nil {
		return err
	}

	// Invalid messages
	for msgID, rr := range r.InvalidMessages {
		for typ, errs := range rr {
			if err = w.Write([]string{msgID, typ, strings.Join(errs, " | ")}); err != nil {
				return err
			}
		}
	}

	// Non parsable messages
	for msgID, errs := range r.NonParsableMessages {
		if err = w.Write([]string{msgID, "non_parsable", strings.Join(errs, " | ")}); err != nil {
			return err
		}
	}

	return nil
}
