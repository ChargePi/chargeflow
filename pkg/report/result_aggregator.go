package report

import (
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/parser"
	"github.com/ChargePi/chargeflow/pkg/validator"
)

const (
	requestKey  = "request"
	responseKey = "response"
)

func getKey(isRequest bool) string {
	key := requestKey
	if !isRequest {
		key = responseKey
	}
	return key
}

// Aggregator is a stateful object that aggregates validation and parser results for messages.
// It can be reset to clear its state and start fresh.
type Aggregator struct {
	logger *zap.Logger

	// Map by message ID and then by request/response
	results             map[string]map[string]Results
	nonParsableMessages map[string][]string

	reportGenerated bool
	stats           Statistics
	report          Report
}

func NewAggregator(logger *zap.Logger) *Aggregator {
	return &Aggregator{
		logger:              logger.Named("result_aggregator"),
		stats:               Statistics{},
		results:             make(map[string]map[string]Results),
		nonParsableMessages: make(map[string][]string),
		reportGenerated:     false,
		report:              Report{},
	}
}

// AddValidationResults adds the validation results for a given message ID and request/response type.
func (a *Aggregator) AddValidationResults(messageId string, isRequest bool, validationResult validator.ValidationResult) {
	if messageId == "" {
		return // Skip if message ID is empty
	}

	a.logger.Debug("Adding validation result", zap.String("messageId", messageId), zap.Any("validationResult", validationResult))

	if _, exists := a.results[messageId]; !exists {
		a.results[messageId] = make(map[string]Results)
	}
	key := getKey(isRequest)

	results := a.results[messageId][key]
	results.ValidationResult = validationResult
	a.results[messageId][key] = results
}

// AddParserResult adds the parser result for a given message ID and request/response type.
func (a *Aggregator) AddParserResult(messageId string, isRequest bool, parserResult parser.Result) {
	if messageId == "" {
		return // Skip if message ID is empty
	}

	a.logger.Debug("Adding parser result", zap.String("messageId", messageId), zap.Any("parserResult", parserResult))

	if _, exists := a.results[messageId]; !exists {
		a.results[messageId] = make(map[string]Results)
	}
	key := getKey(isRequest)

	results := a.results[messageId][key]
	results.Result = parserResult
	a.results[messageId][key] = results
}

// AddNonParsableMessage adds a message ID that could not be parsed, along with the parser result containing errors.
func (a *Aggregator) AddNonParsableMessage(messageId string, parserResult parser.Result) {
	if messageId == "" {
		return // Skip if message ID is empty
	}

	a.logger.Debug("Adding non parsable message", zap.String("messageId", messageId))
	a.nonParsableMessages[messageId] = parserResult.Errors()
}

// CreateReport creates a report based on the collected results.
func (a *Aggregator) CreateReport() Report {
	if a.reportGenerated {
		return a.report
	}

	a.logger.Debug("Creating report from aggregated results")

	defer func() { a.reportGenerated = true }()

	report := Report{
		InvalidMessages:     make(map[string]map[string][]string),
		NonParsableMessages: a.nonParsableMessages,
	}

	for messageId, reqResponse := range a.results {
		for r, results := range reqResponse {

			isRequest := r == requestKey
			isValid := results.ValidationResult.IsValid() && results.Result.IsValid()

			// Keep track of statistics
			switch {
			case isRequest && isValid:
				a.stats.ValidRequests++
			case isRequest:
				a.stats.InvalidRequests++
			case isValid:
				a.stats.ValidResponses++
			default:
				a.stats.InvalidResponses++
			}

			// Request failed validation or parsing
			if !results.ValidationResult.IsValid() || !results.Result.IsValid() {
				if report.InvalidMessages[messageId] == nil {
					report.InvalidMessages[messageId] = make(map[string][]string)
				}

				report.InvalidMessages[messageId][r] = append(results.ValidationResult.Errors(), results.Result.Errors()...)
			}
		}
	}

	// Store the report in the aggregator
	a.report = report

	return report
}

// GetStatistics returns the request and response statistics.
func (a *Aggregator) GetStatistics() Statistics {
	if !a.reportGenerated {
		a.logger.Debug("Calculating statistics from aggregated results")
		// If the report has already been generated, stats are already calculated
		for _, reqResponse := range a.results {
			for r, results := range reqResponse {

				isRequest := r == requestKey
				isValid := results.ValidationResult.IsValid() && results.Result.IsValid()

				// Keep track of statistics
				switch {
				case isRequest && isValid:
					a.stats.ValidRequests++
				case isRequest:
					a.stats.InvalidRequests++
				case isValid:
					a.stats.ValidResponses++
				default:
					a.stats.InvalidResponses++
				}
			}
		}
	}

	return a.stats
}

// Reset clears the aggregator's internal state
func (a *Aggregator) Reset() {
	a.logger.Debug("Resetting aggregator state")
	a.results = make(map[string]map[string]Results)
	a.nonParsableMessages = make(map[string][]string)
	a.reportGenerated = false
	a.stats = Statistics{}
}
