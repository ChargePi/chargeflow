package validation

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/parser"
	"github.com/ChargePi/chargeflow/pkg/report"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/validator"
)

type Service struct {
	logger     *zap.Logger
	registry   schema_registry.SchemaRegistry
	parser     *parser.ParserV2
	validator  *validator.Validator
	aggregator *report.Aggregator
}

func NewService(
	logger *zap.Logger,
	registry schema_registry.SchemaRegistry,
) *Service {
	return &Service{
		logger:     logger,
		registry:   registry,
		parser:     parser.NewParserV2(logger),
		validator:  validator.NewValidator(logger, registry),
		aggregator: report.NewAggregator(logger),
	}
}

// ValidateMessage validates a single OCPP message against the schema.
func (s *Service) ValidateMessage(message string, ocppVersion ocpp.Version) error {
	logger := s.logger.With(zap.String("message", message), zap.String("ocppVersion", ocppVersion.String()))
	logger.Info("Validating message")

	validationReport, err := s.parseAndValidate(ocppVersion, []string{message})
	if err != nil {
		return errors.Wrap(err, "failed to parse message")
	}

	s.outputValidationErrorToLogs(validationReport)

	return nil
}

// ValidateFile validates a file containing multiple OCPP messages against the schema.
func (s *Service) ValidateFile(file string, ocppVersion ocpp.Version) error {
	logger := s.logger.With(zap.String("file", file), zap.String("ocppVersion", ocppVersion.String()))
	logger.Info("Validating file")

	messages, err := s.getMessagesFromFile(file)
	if err != nil {
		return errors.Wrap(err, "unable to read messages from file")
	}

	logger.Info("✅ Successfully parsed file", zap.Int("messages", len(messages)))

	validationReport, err := s.parseAndValidate(ocppVersion, messages)
	if err != nil {
		return errors.Wrap(err, "unable to parse messages")
	}

	s.outputValidationErrorToLogs(validationReport)

	return nil
}

// outputValidationErrorToLogs outputs the validation errors to the logs.
func (s *Service) outputValidationErrorToLogs(validationReport *report.Report) {
	if len(validationReport.InvalidMessages) == 0 && len(validationReport.NonParsableMessages) == 0 {
		s.logger.Info("✅ All messages are valid!")
		return
	}

	// Log the non-parsable messages first
	for line, errs := range validationReport.NonParsableMessages {
		logger := s.logger.With(zap.String("line", line))
		logger.Error(fmt.Sprintf("Message could not be parsed at %s:", line))
		if len(errs) == 0 {
			continue
		}

		for _, parseErr := range errs {
			logger.Error(fmt.Sprintf("👉 %s", parseErr))
		}
	}

	// Log any parsing or validation errors for messages
	for messageId, requestResponse := range validationReport.InvalidMessages {
		for k, validationErrors := range requestResponse {
			logger := s.logger.With(zap.String("messageId", messageId))
			switch k {
			case "request":
				logger.Error(fmt.Sprintf("Request for message %s has the following validation errors:", messageId))
			case "response":
				logger.Error(fmt.Sprintf("Response for message %s has the following validation errors:", messageId))
			}

			if len(validationErrors) == 0 {
				continue
			}

			for _, parseErr := range validationErrors {
				logger.Error(fmt.Sprintf("👉 %s", parseErr))
			}
		}
	}
}

// parseAndValidate parses and validates a list of OCPP messages.
func (s *Service) parseAndValidate(ocppVersion ocpp.Version, messages []string) (*report.Report, error) {
	logger := s.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.Int("messages", len(messages)))
	logger.Info("Parsing and validating messages")

	// Parse the messages
	parserResults, nonParsedMessages, err := s.parser.Parse(messages)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse messages")
	}

	// Add non-parsable messages to the aggregator
	for line, result := range nonParsedMessages {
		s.aggregator.AddNonParsableMessage(line, result)
	}

	// Add parsed messages to the aggregator
	for messageId, result := range parserResults {
		// Validate the request
		_, found := result.GetRequest()
		if found {
			s.aggregator.AddParserResult(messageId, true, result.Request)
		}

		_, found = result.GetResponse()
		if found {
			s.aggregator.AddParserResult(messageId, false, result.Response)
		}
	}

	// Only valid messages should be validated further
	validMessages := s.filterValidMessages(parserResults)
	invalidMessagesCount := len(parserResults) - len(validMessages)
	logger.Info(
		"✅ OCPP messages parsed. Proceeding with validation.",
		zap.Int("invalid_messages", invalidMessagesCount),
		zap.Int("unparsable_messages", len(nonParsedMessages)),
	)

	for messageId, parserResult := range validMessages {
		// Validate the request
		request, found := parserResult.GetRequest()
		if found {
			result, err := s.validator.ValidateMessage(ocppVersion, request)
			if err != nil {
				return nil, errors.Wrap(err, "failed to validate request message")
			}

			// Store the results in the aggregator
			s.aggregator.AddValidationResults(messageId, true, *result)
		}

		// Validate the response
		response, found := parserResult.GetResponse()
		if found {
			result, err := s.validator.ValidateMessage(ocppVersion, response)
			if err != nil {
				return nil, errors.Wrap(err, "failed to validate response message")
			}

			// Store the results in the aggregator
			s.aggregator.AddValidationResults(messageId, false, *result)
		}
	}

	validationReport := s.aggregator.CreateReport()
	return &validationReport, nil
}

// getMessagesFromFile reads messages from a file, where each message is separated by a newline character.
func (s *Service) getMessagesFromFile(file string) ([]string, error) {
	s.logger.Debug("Reading file", zap.String("file", file))

	openFile, err := os.OpenFile(file, os.O_RDONLY, 0666)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}

	content, err := io.ReadAll(openFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file content")
	}

	messages := strings.Split(string(content), "\n")
	return messages, nil
}

// filterValidMessages filters out invalid messages from the parser results.
func (s *Service) filterValidMessages(parserResults map[string]parser.RequestResponseResult) map[string]parser.RequestResponseResult {
	validMessages := make(map[string]parser.RequestResponseResult)

	for messageUniqueId, parserResult := range parserResults {
		if !parserResult.IsValid() {
			continue
		}
		validMessages[messageUniqueId] = parserResult
	}

	return validMessages
}
