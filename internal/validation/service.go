package validation

import (
	"fmt"
	"io"
	"maps"
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

// Validate validates messages and returns the report. When vendor and/or model are set,
// the registry attempts vendor/model-specific schemas before falling back to the base OCPP spec schemas.
func (s *Service) Validate(req Request) (*report.Report, error) {
	logger := s.logger.With(
		zap.String("ocppVersion", req.OcppContext.Version.String()),
		zap.String("vendor", req.OcppContext.Vendor),
		zap.String("model", req.OcppContext.Model),
	)
	logger.Info("Validating messages")

	msgs := req.Messages
	if len(msgs) == 0 && req.File != "" {
		var err error
		msgs, err = s.getMessagesFromFile(req.File)
		if err != nil {
			return nil, errors.Wrap(err, "unable to read messages from file")
		}
	}

	validationReport, err := s.parseAndValidate(req.OcppContext, msgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse and validate messages")
	}

	s.outputValidationErrorToLogs(validationReport)

	if req.Output != "" {
		strat, err := outputStrategyFactory(req.Output)
		if err != nil {
			return nil, err
		}
		if err := strat.Write(req.Output, validationReport); err != nil {
			return nil, errors.Wrap(err, "failed to write validation report")
		}
	}

	return validationReport, nil
}

// outputValidationErrorToLogs outputs the validation errors to the logs.
func (s *Service) outputValidationErrorToLogs(validationReport *report.Report) {
	if len(validationReport.InvalidMessages) == 0 && len(validationReport.NonParsableMessages) == 0 {
		s.logger.Info("✅ All messages are valid!")
		return
	}

	for line, errs := range validationReport.NonParsableMessages {
		logger := s.logger.With(zap.String("line", line))
		logger.Error(fmt.Sprintf("Message could not be parsed at %s:", line))
		for _, parseErr := range errs {
			logger.Error(fmt.Sprintf("👉 %s", parseErr))
		}
	}

	for messageId, requestResponse := range validationReport.InvalidMessages {
		for k, validationErrors := range requestResponse {
			logger := s.logger.With(zap.String("messageId", messageId))
			switch k {
			case "request":
				logger.Error(fmt.Sprintf("Request for message %s has the following validation errors:", messageId))
			case "response":
				logger.Error(fmt.Sprintf("Response for message %s has the following validation errors:", messageId))
			}

			for _, parseErr := range validationErrors {
				logger.Error(fmt.Sprintf("👉 %s", parseErr))
			}
		}
	}
}

// parseAndValidate parses and validates a list of OCPP messages.
func (s *Service) parseAndValidate(octx ocpp.OcppContext, messages []string) (*report.Report, error) {
	logger := s.logger.With(zap.String("ocppVersion", octx.Version.String()), zap.Int("messages", len(messages)))
	logger.Info("Parsing and validating messages")

	parserResults, nonParsedMessages, err := s.parser.Parse(messages)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse messages")
	}

	for line, result := range nonParsedMessages {
		s.aggregator.AddNonParsableMessage(line, result)
	}

	for messageId, result := range parserResults {
		_, found := result.GetRequest()
		if found {
			s.aggregator.AddParserResult(messageId, true, result.Request)
		}

		_, found = result.GetResponse()
		if found {
			s.aggregator.AddParserResult(messageId, false, result.Response)
		}
	}

	validMessages := s.filterValidMessages(parserResults)
	invalidMessagesCount := len(parserResults) - len(validMessages)
	logger.Info("✅ OCPP messages parsed. Proceeding with validation.",
		zap.Int("invalid_messages", invalidMessagesCount),
		zap.Int("unparsable_messages", len(nonParsedMessages)),
	)

	for messageId, parserResult := range validMessages {
		request, found := parserResult.GetRequest()
		if found {
			result, err := s.validator.ValidateMessage(octx, request)
			if err != nil {
				return nil, errors.Wrap(err, "failed to validate request message")
			}
			s.aggregator.AddValidationResults(messageId, true, *result)
		}

		response, found := parserResult.GetResponse()
		if found {
			result, err := s.validator.ValidateMessage(octx, response)
			if err != nil {
				return nil, errors.Wrap(err, "failed to validate response message")
			}
			s.aggregator.AddValidationResults(messageId, false, *result)
		}

		responseError, found := parserResult.GetResponseError()
		if found {
			result, err := s.validator.ValidateMessage(octx, responseError)
			if err != nil {
				return nil, errors.Wrap(err, "failed to validate response error message")
			}
			s.aggregator.AddValidationResults(messageId, false, *result)
		}
	}

	validationReport := s.aggregator.CreateReport()
	return &validationReport, nil
}

// getMessagesFromFile reads newline-delimited OCPP messages from a file.
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

	return strings.Split(string(content), "\n"), nil
}

// filterValidMessages removes parser results that failed basic structural parsing.
func (s *Service) filterValidMessages(parserResults map[string]parser.RequestResponseResult) map[string]parser.RequestResponseResult {
	maps.DeleteFunc(parserResults, func(_ string, parserResult parser.RequestResponseResult) bool {
		return !parserResult.IsValid()
	})
	return parserResults
}
