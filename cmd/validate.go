package cmd

import (
	"embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/parser"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/validator"
)

var registry schema_registry.SchemaRegistry

var messageParser *parser.Parser

// OCPP 1.6 schemas
//
//go:embed schemas/ocpp_16/*
var ocpp16Schemas embed.FS

//go:embed schemas/ocpp_201/*
var ocpp201Schemas embed.FS

var additionalOcppSchemasFolder = ""

// registerSchemas registers all schemas from the embedded filesystem for a specific OCPP version.
func registerSchemas(logger *zap.Logger, embeddedDir embed.FS, version ocpp.Version, registry schema_registry.SchemaRegistry) error {
	logger.Debug("Registering OCPP schemas", zap.String("version", version.String()))

	dirPath := "schemas/ocpp_" + strings.ReplaceAll(version.String(), ".", "")
	dir, err := embeddedDir.ReadDir(dirPath)
	if err != nil {
		return errors.Wrap(err, "unable to read OCPP 1.6 schemas directory")
	}

	for _, file := range dir {
		if !file.IsDir() {
			name := file.Name()
			logger.Debug("Registering OCPP schema file", zap.String("file", name))

			// Open and read the schema file
			schemaData, err := embeddedDir.ReadFile(filepath.Join(dirPath, name))
			if err != nil {
				return errors.Wrapf(err, "unable to read OCPP 1.6 schema file: %s", name)
			}

			// Note: Assuming that the file name is equivalent to the action name
			// Improvement: Could extract the action name.
			// Also could determine the OCPP version from the schema ID.

			action, _ := strings.CutSuffix(name, ".json")
			err = registry.RegisterSchema(version, action, schemaData)
			if err != nil {
				return errors.Wrapf(err, "unable to register OCPP 1.6 schema: %s", name)
			}
		}
	}

	return nil
}

// registerAdditionalSchemas registers additional OCPP schemas from a specified directory.
// Files must be in JSON format and their names should match the OCPP message names (e.g. "BootNotificationRequest.json" or "BootNotificationResponse.json").

func registerAdditionalSchemas(logger *zap.Logger, dir string) error {
	ocppVersion := viper.GetString("ocpp.version")
	logger.Debug("Registering additional OCPP schemas from directory", zap.String("directory", dir))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrap(err, "unable to read provided additional OCPP schemas directory")
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			fileName := entry.Name()
			// Read the schema file
			schema, err := os.ReadFile(filepath.Join(dir, fileName))
			if err != nil {
				return errors.Wrap(err, "unable to read additional OCPP schemas directory")
			}

			// Read the directory and register additional OCPP schemas
			// Any existing schema with the same name will be overwritten
			action, _ := strings.CutSuffix(fileName, ".json")
			err = registry.RegisterSchema(ocpp.Version(ocppVersion), action, schema, schema_registry.WithOverwrite(true))
			if err != nil {
				return errors.Wrap(err, "failed to register additional OCPP schemas")
			}
		}
	}

	return nil
}

var validate = &cobra.Command{
	Use:          "validate",
	Short:        "Validate the OCPP message(s) against the registered OCPP schemas",
	Long:         `Validate the OCPP message(s) against the registered OCPP schema(s).`,
	Example:      "chargeflow --version 1.6 validate '[1, \"123456\", \"BootNotification\", {\"chargePointVendor\": \"TestVendor\", \"chargePointModel\": \"TestModel\"}]'",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		ocppVersion := viper.GetString("ocpp.version")
		logger := zap.L()

		registry = schema_registry.NewInMemorySchemaRegistry(logger)
		messageParser = parser.NewParser(logger)

		// Populate the schema registry with OCPP schemas
		var err error
		switch ocppVersion {
		case ocpp.V16.String():
			err = registerSchemas(logger, ocpp16Schemas, ocpp.V16, registry)
			if err != nil {
				return err
			}
		case ocpp.V20.String():
			err = registerSchemas(logger, ocpp201Schemas, ocpp.V20, registry)
			if err != nil {
				return err
			}
		}

		if additionalOcppSchemasFolder != "" {
			err := registerAdditionalSchemas(logger, additionalOcppSchemasFolder)
			if err != nil {
				return err
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ocppVersion := viper.GetString("ocpp.version")
		logger := zap.L()
		validator := validator.NewValidator(logger, registry)

		// The argument (message) is expected to be a JSON string in the format:
		// '[2, "uniqueId", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]'
		message := args[0]

		parseMessage, parseResult, err := messageParser.ParseMessage(message)
		if err != nil {
			return err
		}

		if !parseResult.IsValid() {
			logger.Info("❌ Failure: The message could not be parsed or had syntax errors:")
			for _, err := range parseResult.Errors() {
				logger.Info("- " + err)
			}

			return nil
		}

		logger.Info("✅ Message successfully parsed. Proceeding with validation.")

		result, err := validator.ValidateMessage(ocpp.Version(ocppVersion), parseMessage)
		if err != nil {
			return err
		}

		if result.IsValid() {
			logger.Info("✅ Success: The message is valid according to the OCPP schema.")
		} else {
			logger.Info("❌ Failure: The message is NOT valid according to the OCPP schema:")
			for _, err := range result.Errors() {
				logger.Info("- " + err)
			}
		}

		return nil
	},
}

func init() {
	// Add flags for additional OCPP schemas folder
	validate.Flags().StringVarP(&additionalOcppSchemasFolder, "schemas", "a", "", "Path to additional OCPP schemas folder")
	validate.Flags().StringP("response-type", "r", "", "Response type to validate against (e.g. 'BootNotificationResponse'). Currently needed if you want to validate a single response message. ")

	_ = viper.BindPFlag("response-type", validate.Flags().Lookup("response-type"))
}
