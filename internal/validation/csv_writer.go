package validation

import (
	"encoding/csv"
	"errors"
	"os"
	"strings"

	"github.com/ChargePi/chargeflow/pkg/report"
)

var headers = []string{"message_id", "type", "errors"}

// csvWriter implements ReportWriter for CSV output.
type csvWriter struct{}

func (csvWriter) Write(path string, r *report.Report) error {
	if r == nil {
		return errors.New("report is nil")
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	if err = w.Write(headers); err != nil {
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
