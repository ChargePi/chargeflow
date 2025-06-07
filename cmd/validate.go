package cmd

import (
	"embed"
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/parser"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/validator"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

var registry = schema_registry.NewSchemaRegistry(zap.L())

var messageParser = parser.NewParser(zap.L())

// OCPP 1.6 schemas
//
//go:embed ../schemas/ocpp_16/*.json
var ocpp16Schemas embed.FS

//go:embed ../schemas/ocpp_201/*.json
var ocpp201Schemas embed.FS

var additionalOcppSchemasFolder = ""

// registerSchemas registers all schemas from the embedded filesystem for a specific OCPP version.
func registerSchemas(fs embed.FS, version ocpp.Version) error {
	dir, err := fs.ReadDir(".")
	if err != nil {
		return errors.Wrap(err, "unable to read OCPP 1.6 schemas directory")
	}

	for _, file := range dir {
		name := file.Name()

		// Open and read the schema file
		schemaData, err := fs.ReadFile(name)
		if err != nil {
			return errors.Wrapf(err, "unable to read OCPP 1.6 schema file: %s", name)
		}

		err = registry.RegisterSchema(version, name, schemaData)
		if err != nil {
			return errors.Wrapf(err, "unable to register OCPP 1.6 schema: %s", name)
		}
	}

	return nil
}

// registerOcpp16Schemas registers all OCPP 1.6 schemas from the embedded filesystem.
func registerOcpp16Schemas() error {
	return registerSchemas(ocpp16Schemas, ocpp.V16)
}

// registerOcpp201Schemas registers all OCPP 2.0.1 schemas from the embedded filesystem.
func registerOcpp201Schemas() error {
	return registerSchemas(ocpp201Schemas, ocpp.V20)
}

// registerAdditionalSchemas registers additional OCPP schemas from a specified directory.
// Files must be in JSON format and their names should match the OCPP message names (e.g. "BootNotificationRequest.json" or "BootNotificationResponse.json").

func registerAdditionalSchemas(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrap(err, "unable to read provided additional OCPP schemas directory")
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			// Read the schema file
			schema, err := os.ReadFile(dir + "/" + entry.Name())
			if err != nil {
				return errors.Wrap(err, "unable to read additional OCPP schemas directory")
			}

			// Read the directory and register additional OCPP schemas
			// Any existing schema with the same name will be overwritten
			err = registry.RegisterSchema(ocpp.Version(defaultOcppVersion), entry.Name(), schema, schema_registry.WithOverwrite(true))
			if err != nil {
				return errors.Wrap(err, "failed to register additional OCPP schemas")
			}

		}
	}

	return nil
}

var validate = &cobra.Command{
	Use:     "validate",
	Short:   "Validate the OCPP message(s) against the registered OCPP schemas",
	Long:    `Validate the OCPP message(s) against the registered OCPP schema(s).`,
	Example: "chargeflow --version 1.6 validate [1234567, \"1\", \"BootNotification\", {\"chargePointVendor\": \"TestVendor\", \"chargePointModel\": \"TestModel\"}]",
	Args:    cobra.MinimumNArgs(2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Populate the schema registry with OCPP schemas
		var err error
		switch defaultOcppVersion {
		case ocpp.V16.String():

			err = registerOcpp16Schemas()
			if err != nil {
				return err
			}
		case ocpp.V20.String():
			err = registerOcpp201Schemas()
			if err != nil {
				return err
			}
		}

		if additionalOcppSchemasFolder != "" {
			err := registerAdditionalSchemas(additionalOcppSchemasFolder)
			if err != nil {
				return err
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := zap.L()
		validator := validator.NewValidator(logger, registry)

		ocppVersion := args[0]
		message := args[1] // The message is expected to be a JSON string

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
}
