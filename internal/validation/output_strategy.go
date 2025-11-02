package validation

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/ChargePi/chargeflow/pkg/report"
)

// OutputStrategy defines how to write a validation report.
type OutputStrategy interface {
	Write(path string, r *report.Report) error
}

// outputStrategyFactory returns an OutputStrategy based on the file extension.
func outputStrategyFactory(path string) (OutputStrategy, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return jsonStrategy{}, nil
	case ".csv":
		return csvStrategy{}, nil
	case ".txt":
		return txtStrategy{}, nil
	default:
		return nil, errors.Errorf("unsupported output extension: %s", ext)
	}
}

// WriteReport is a convenience exported helper that writes the report using the
// appropriate OutputStrategy based on the provided path extension.
func WriteReport(path string, r *report.Report) error {
	strat, err := outputStrategyFactory(path)
	if err != nil {
		return err
	}
	return strat.Write(path, r)
}
