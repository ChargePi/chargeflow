package report

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/parser"
	"github.com/ChargePi/chargeflow/pkg/validator"

	"github.com/stretchr/testify/suite"
)

func TestUnitGetKey(t *testing.T) {
	t.Run("Request", func(t *testing.T) {
		assert.Equal(t, requestKey, getKey(true))
	})

	t.Run("Response", func(t *testing.T) {
		assert.Equal(t, responseKey, getKey(false))
	})
}

type aggregatorTestSuite struct {
	suite.Suite
	logger *zap.Logger
}

func (s *aggregatorTestSuite) SetupSuite() {
	s.logger = zap.NewExample()
}

func (s *aggregatorTestSuite) TestAddParserResult() {
	tests := []struct {
		name          string
		messageId     string
		isRequest     bool
		resultMutator func(result *parser.Result)
	}{
		{
			name:      "Valid Request",
			messageId: uuid.NewString(),
			isRequest: true,
		},
		{
			name:      "Valid Response",
			messageId: uuid.NewString(),
			isRequest: false,
		},
		{
			name:      "Invalid Request",
			messageId: uuid.NewString(),
			isRequest: true,
			resultMutator: func(result *parser.Result) {
				result.AddError("error in request parsing")
			},
		},
		{
			name:      "Invalid Response",
			messageId: uuid.NewString(),
			isRequest: false,
			resultMutator: func(result *parser.Result) {
				result.AddError("error in request parsing")
			},
		},
		{
			name:      "Message ID is empty",
			messageId: "",
			isRequest: true,
		},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(t *testing.T) {
			aggregator := NewAggregator(s.logger)
			s.Require().NotNil(aggregator)

			result := parser.NewResult()

			if test.resultMutator != nil {
				test.resultMutator(result)
			}

			aggregator.AddParserResult(test.messageId, test.isRequest, *result)

			if test.messageId == "" {
				s.Empty(aggregator.results)
				return
			}

			if test.isRequest {
				s.NotEmpty(aggregator.results[test.messageId])
				s.Equal(*result, aggregator.results[test.messageId]["request"].Result)
			} else {
				s.NotEmpty(aggregator.results[test.messageId])
				s.Equal(*result, aggregator.results[test.messageId]["response"].Result)
			}
		})
	}
}

func (s *aggregatorTestSuite) TestAddNonParsableMessage() {
	tests := []struct {
		name      string
		messageId string
	}{
		{
			name:      "Non-parsable message",
			messageId: uuid.NewString(),
		},
		{
			name:      "Message ID is empty",
			messageId: "",
		},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(t *testing.T) {
			aggregator := NewAggregator(s.logger)
			s.Require().NotNil(aggregator)

			result := parser.NewResult()
			result.AddError("error in request non parsable")

			aggregator.AddNonParsableMessage(test.messageId, *result)

			if test.messageId == "" {
				s.Empty(aggregator.results)
				return
			}

			s.Equal(result.Errors(), aggregator.nonParsableMessages[test.messageId])
		})
	}
}

func (s *aggregatorTestSuite) TestAddValidationResults() {
	tests := []struct {
		name          string
		messageId     string
		isRequest     bool
		resultMutator func(result *validator.ValidationResult)
	}{
		{
			name:          "Valid Request",
			messageId:     uuid.NewString(),
			isRequest:     true,
			resultMutator: nil,
		},
		{
			name:          "Valid Response",
			messageId:     uuid.NewString(),
			isRequest:     true,
			resultMutator: nil,
		},
		{
			name:      "Invalid Request",
			messageId: uuid.NewString(),
			isRequest: true,
			resultMutator: func(result *validator.ValidationResult) {
				result.AddError("invalid request")
			},
		},
		{
			name:      "Invalid Response",
			messageId: uuid.NewString(),
			isRequest: true,
			resultMutator: func(result *validator.ValidationResult) {
				result.AddError("invalid response")
			},
		},
		{
			name:      "Message ID is empty",
			messageId: "",
			isRequest: true,
		},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(t *testing.T) {
			aggregator := NewAggregator(s.logger)
			s.Require().NotNil(aggregator)

			result := validator.NewValidationResult()

			if test.resultMutator != nil {
				test.resultMutator(result)
			}

			aggregator.AddValidationResults(test.messageId, test.isRequest, *result)

			if test.messageId == "" {
				s.Empty(aggregator.results)
				return
			}

			if test.isRequest {
				s.NotEmpty(aggregator.results[test.messageId])
				s.Equal(*result, aggregator.results[test.messageId]["request"].ValidationResult)
			} else {
				s.NotEmpty(aggregator.results[test.messageId])
				s.Equal(*result, aggregator.results[test.messageId]["response"].ValidationResult)
			}
		})
	}
}

func (s *aggregatorTestSuite) TestGetStatistics() {
	s.T().Run("Report wasnt already generated", func(t *testing.T) {
		aggregator := NewAggregator(s.logger)
		s.Require().NotNil(aggregator)
		aggregator.results = make(map[string]map[string]Results)
		message1 := uuid.NewString()
		message2 := uuid.NewString()
		message3 := uuid.NewString()

		setupAggregator(aggregator, message1,
			Results{
				ValidationResult: validator.ValidationResult{},
				Result:           parser.Result{},
			},
			Results{
				ValidationResult: validator.ValidationResult{},
				Result:           parser.Result{},
			},
		)

		setupAggregator(aggregator, message2,
			Results{
				ValidationResult: validator.ValidationResult{},
				Result:           parser.Result{},
			},
			Results{
				ValidationResult: validator.ValidationResult{},
				Result:           parser.Result{},
			})

		setupAggregator(aggregator, message3,
			Results{
				ValidationResult: validator.ValidationResult{},
				Result:           parser.Result{},
			},
			Results{
				ValidationResult: validator.ValidationResult{},
				Result:           parser.Result{},
			})

		stats := aggregator.GetStatistics()
		s.Require().NotNil(stats)
	})

	s.T().Run("Report was already generated", func(t *testing.T) {
		aggregator := NewAggregator(s.logger)
		s.Require().NotNil(aggregator)

		aggregator.reportGenerated = true
		aggregator.stats = Statistics{
			ValidRequests:      10,
			ValidResponses:     9,
			InvalidRequests:    1,
			InvalidResponses:   2,
			UnparsableMessages: 2,
		}
		stats := aggregator.GetStatistics()
		s.Require().NotNil(stats)
		s.Equal(10, stats.ValidRequests)
		s.Equal(9, stats.ValidResponses)
		s.Equal(1, stats.InvalidRequests)
		s.Equal(2, stats.InvalidResponses)
		s.Equal(2, stats.UnparsableMessages)
	})
}

func setupAggregator(aggregator *Aggregator, messageId string, request Results, response Results) {
	if aggregator.results == nil {
		aggregator.results = make(map[string]map[string]Results)
	}
	if _, exists := aggregator.results[messageId]; !exists {
		aggregator.results[messageId] = make(map[string]Results)
	}
	aggregator.results[messageId]["request"] = request
	aggregator.results[messageId]["response"] = response
}

// Example flow:
// 1. Create an aggregator instance.
// 2. Add a parsable result.
// 3. Add a non-parsable message.
// 4. Add validation results.
// 5. Create a report.
func (s *aggregatorTestSuite) TestExampleFlow() {
	// Step 1: Create an aggregator instance.
	aggregator := NewAggregator(s.logger)
	s.Require().NotNil(aggregator)

	// Step 2: Add a parsable result.
	messageId := uuid.NewString()
	aggregator.AddParserResult(messageId, true, *parser.NewResult())

	// Example error for the parsable result.
	parseErrorMessageId := uuid.NewString()
	parseResult := parser.NewResult()
	parseResult.AddError("example error")
	aggregator.AddParserResult(parseErrorMessageId, true, *parseResult)

	// Step 3: Add a non-parsable message.
	nonParsableMessageId := uuid.NewString()
	parseResult = parser.NewResult()
	parseResult.AddError("example error")
	aggregator.AddNonParsableMessage(nonParsableMessageId, *parseResult)

	// Step 4: Add validation results.
	validationResult := validator.NewValidationResult()
	aggregator.AddValidationResults(messageId, true, *validationResult)

	// Step 5: Create a report.
	report := aggregator.CreateReport()
	s.Require().NotNil(report)
	s.NotEmpty(report.InvalidMessages)
	s.NotContains(report.InvalidMessages, messageId)
	s.Contains(report.InvalidMessages, parseErrorMessageId)

	s.NotEmpty(report.NonParsableMessages)
	s.Contains(report.NonParsableMessages, nonParsableMessageId)

	// Fetch the report again
	report2 := aggregator.CreateReport()
	s.Equal(report, report2)

	// Reset the aggregator
	aggregator.Reset()
	s.Empty(aggregator.results)
}

func TestAggregator(t *testing.T) {
	suite.Run(t, new(aggregatorTestSuite))
}
