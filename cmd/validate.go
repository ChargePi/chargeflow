package cmd

import (
	"embed"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/parser"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/validator"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var registry = schema_registry.NewSchemaRegistry(zap.L())

var messageParser = parser.NewParser(zap.L())

// OCPP 1.6 schemas
//
//go:embed ../schemas/ocpp_16/*.json
var ocpp16Schemas embed.FS

//go:embed ../schemas/ocpp_201/*.json
var ocpp201Schemas embed.FS

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

var validate = &cobra.Command{
	Use:     "validate",
	Short:   "Validate the OCPP message(s) against the registered OCPP schemas",
	Long:    `Validate the OCPP message(s) against the registered OCPP schema(s).`,
	Example: "chargeflow validate 1.6 [1234567, \"1\", \"BootNotification\", {\"chargePointVendor\": \"TestVendor\", \"chargePointModel\": \"TestModel\"}]",
	Args:    cobra.MinimumNArgs(2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Populate the schema registry with OCPP schemas
		err := registerOcpp16Schemas()
		if err != nil {
			return err
		}

		err = registerOcpp201Schemas()
		if err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := zap.L()
		validator := validator.NewValidator(logger, registry)

		ocppVersion := args[0]
		message := args[1] // The message is expected to be a JSON string

		parseMessage, err := messageParser.ParseMessage(message)
		if err != nil {
			return err
		}

		result, err := validator.ValidateMessage(ocpp.Version(ocppVersion), parseMessage)
		if err != nil {
			return err
		}

		if result.IsValid() {
			println("✅ Success: The message is valid according to the OCPP schema.")
		} else {
			println("❌ Failure: The message is NOT valid according to the OCPP schema.")
			println("Validation errors:")
			for _, err := range result.Errors() {
				println("- " + err)
			}
		}

		return nil
	},
}
