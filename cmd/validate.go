package cmd

import (
	"embed"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/internal/validation"
	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/schema_registry/registries"
)

var (
	registry schema_registry.SchemaRegistry

	// OCPP 1.6 schemas
	//
	//go:embed schemas/ocpp_16/*
	ocpp16Schemas embed.FS

	// OCPP 1.6 Security Extension schemas
	//
	//go:embed schemas/ocpp_16_security/*
	ocpp16Security embed.FS

	//go:embed schemas/ocpp_201/*
	ocpp201Schemas embed.FS

	//go:embed schemas/ocpp_21/*
	ocpp21Schemas embed.FS
)

var (
	additionalOcppSchemasFolder = ""

	// supportedOutputFormats lists allowed output file formats for the CLI report writer.
	supportedOutputFormats = map[string]bool{".json": true, ".csv": true, ".txt": true}
)

// registerSchemas registers all schemas from the embedded filesystem for a specific OCPP version.
func registerSchemas(logger *zap.Logger, embeddedDir embed.FS, version ocpp.Version, registry schema_registry.SchemaRegistry) error {
	logger.Debug("Registering OCPP schemas", zap.String("version", version.String()))

	dirPath := "schemas/ocpp_" + strings.ReplaceAll(version.String(), ".", "")

	// Exception for OCPP 1.6 Security Extension schemas
	if embeddedDir == ocpp16Security {
		dirPath = "schemas/ocpp_" + strings.ReplaceAll(version.String(), ".", "") + "_security"
	}

	dir, err := embeddedDir.ReadDir(dirPath)
	if err != nil {
		return errors.Wrapf(err, "unable to read OCPP schemas directory for version: %s", version.String())
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

var validate = &cobra.Command{
	Use:          "validate",
	Short:        "Validate the OCPP message(s) against the registered OCPP schemas",
	Long:         `Validate the OCPP message(s) against the registered OCPP schema(s).`,
	Example:      "chargeflow --version 1.6 validate '[2, \"123456\", \"BootNotification\", {\"chargePointVendor\": \"TestVendor\", \"chargePointModel\": \"TestModel\"}]'",
	Args:         cobra.RangeArgs(0, 1),
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		ocppVersion := viper.GetString("ocpp.version")
		logger := zap.L()
		registryType := viper.GetString("schema.registry.type")

		overwrite := additionalOcppSchemasFolder != ""

		// todo provision registries based on config
		switch registryType {
		default:
			registry = registries.NewFileSchemaRegistry(
				logger,
				registries.WithOverwrite(overwrite),
			)
		}

		// Populate the schema registry with OCPP schemas
		var err error
		switch ocppVersion {
		case ocpp.V16.String():
			err = registerSchemas(logger, ocpp16Schemas, ocpp.V16, registry)
			if err != nil {
				return err
			}

			err = registerSchemas(logger, ocpp16Security, ocpp.V16, registry)
			if err != nil {
				return err
			}
		case ocpp.V20.String():
			err = registerSchemas(logger, ocpp201Schemas, ocpp.V20, registry)
			if err != nil {
				return err
			}
		case ocpp.V21.String():
			err = registerSchemas(logger, ocpp21Schemas, ocpp.V21, registry)
			if err != nil {
				return err
			}
		}

		if overwrite {
			ocppVersion := viper.GetString("ocpp.version")
			version := ocpp.Version(ocppVersion)
			err := registerSchemasFromDir(logger, registry, version, additionalOcppSchemasFolder)
			if err != nil {
				return err
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ocppVersion := viper.GetString("ocpp.version")
		file := viper.GetString("file")
		version := ocpp.Version(ocppVersion)

		logger := zap.L()
		logger = logger.WithOptions(zap.WithCaller(false), zap.AddStacktrace(zap.FatalLevel))

		service := validation.NewService(logger, registry)

		var message string
		if len(args) > 0 {
			message = args[0]
		}

		output := viper.GetString("output")
		validationOpts := []validation.Option{}

		// Validate provided output extension if present
		if output != "" {
			ext := strings.ToLower(filepath.Ext(output))
			if !supportedOutputFormats[ext] {
				return errors.Errorf("unsupported output format '%s', supported: .json, .csv, .txt", ext)
			}

			validationOpts = append(validationOpts, validation.WithOutput(output))
		}

		switch {
		case file == "" && message == "":
			return errors.New("no message provided to validate, please provide a message as a command line argument or use the --file flag to read from a file")
		case message != "":
			// The message is expected to be a JSON string in the format:
			// '[2, "uniqueId", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]'
			if output == "" {
				return service.ValidateMessage(message, version)
			}

			// Validate and write report
			r, err := service.ValidateMessageWithReport(message, version)
			if err != nil {
				return err
			}

			return validation.WriteReport(output, r)

		case file != "":
			// Use the options pattern to write output using registered strategies
			// Read the messages from the file
			return service.ValidateFile(file, version, validationOpts...)
		}

		return nil
	},
}

func init() {
	// Add flags for additional OCPP schemas folder
	validate.Flags().StringVarP(&additionalOcppSchemasFolder, "schemas", "a", "", "Path to additional OCPP schemas folder")
	validate.Flags().StringP("response-type", "r", "", "Response type to validate against (e.g. 'BootNotificationResponse'). Currently needed if you want to validate a single response message. ")
	validate.Flags().StringP("file", "f", "", "Path to a file containing the OCPP message to validate. If this flag is set, the message will be read from the file instead of the command line argument.")
	validate.Flags().StringP("output", "o", "", "Path to write validation report. Supports .json, .csv and .txt extensions.")

	_ = viper.BindPFlag("response-type", validate.Flags().Lookup("response-type"))
	_ = viper.BindPFlag("file", validate.Flags().Lookup("file"))
	_ = viper.BindPFlag("output", validate.Flags().Lookup("output"))
}
