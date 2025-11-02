package validation

import (
	"fmt"
	"os"
	"strings"

	"github.com/ChargePi/chargeflow/pkg/report"
)

// txtStrategy implements OutputStrategy for plain text output.
type txtStrategy struct{}

func (txtStrategy) Write(path string, r *report.Report) error {
	var b strings.Builder

	stats := r.Statistics
	b.WriteString(fmt.Sprintf("Valid requests: %d\n", stats.ValidRequests))
	b.WriteString(fmt.Sprintf("Invalid requests: %d\n", stats.InvalidRequests))
	b.WriteString(fmt.Sprintf("Valid responses: %d\n", stats.ValidResponses))
	b.WriteString(fmt.Sprintf("Invalid responses: %d\n", stats.InvalidResponses))
	b.WriteString(fmt.Sprintf("Unparsable messages: %d\n", stats.UnparsableMessages))
	b.WriteString(fmt.Sprintf("Success rate: %.2f%%\n\n", stats.TotalValidMessagesPercentage()))

	if len(r.InvalidMessages) == 0 && len(r.NonParsableMessages) == 0 {
		b.WriteString("All messages are valid!\n")
	} else {
		for msgID, rr := range r.InvalidMessages {
			b.WriteString(fmt.Sprintf("Message %s:\n", msgID))
			for typ, errs := range rr {
				b.WriteString(fmt.Sprintf("  %s:\n", typ))
				for _, e := range errs {
					b.WriteString(fmt.Sprintf("    - %s\n", e))
				}
			}
			b.WriteString("\n")
		}

		if len(r.NonParsableMessages) > 0 {
			b.WriteString("Non parsable messages:\n")
			for msgID, errs := range r.NonParsableMessages {
				b.WriteString(fmt.Sprintf("  %s:\n", msgID))
				for _, e := range errs {
					b.WriteString(fmt.Sprintf("    - %s\n", e))
				}
				b.WriteString("\n")
			}
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return err
	}

	return nil
}
