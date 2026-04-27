package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
)

type registerConfig struct {
	SchemaFile string
	SchemaDir  string
	Action     string
}

var registerCfg registerConfig

var register = &cobra.Command{
	Use:   "register",
	Short: "Register schemas on a remote schema registry",
	Long: `Register OCPP schemas on a remote schema registry.
You can register a single schema file or all schemas from a directory.
The schema file names should match the OCPP action names (e.g., "BootNotificationRequest.json" or "BootNotificationResponse.json").`,
	Example: `  # Register a single schema file
  chargeflow schema --url http://localhost:8081 --version 1.6 register --file BootNotificationRequest.json --action BootNotificationRequest

  # Register all schemas from a directory
  chargeflow schema --url http://localhost:8081 --version 1.6 register --dir ./schemas

  # Register with bearer token
  chargeflow schema --url http://localhost:8081 --auth-type bearer --bearer-token token123 register --file schema.json --action BootNotificationRequest`,
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := validateSchemaConfig(); err != nil {
			return err
		}

		cfg := loadRegisterConfig()

		if cfg.SchemaFile == "" && cfg.SchemaDir == "" {
			return errors.New("either --file or --dir must be specified")
		}

		if cfg.SchemaFile != "" && cfg.SchemaDir != "" {
			return errors.New("cannot specify both --file and --dir")
		}

		if cfg.SchemaFile != "" && cfg.Action == "" {
			return errors.New("--action is required when using --file")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := zap.L()

		cfg := loadRegisterConfig()
		version := ocpp.Version(viper.GetString("ocpp.version"))

		remoteRegistry, err := buildRemoteRegistry(logger)
		if err != nil {
			return errors.Wrap(err, "failed to create remote schema registry")
		}

		ctx := cmd.Context()

		switch {
		case cfg.SchemaFile != "":
			return registerSingleSchema(ctx, logger, remoteRegistry, version, cfg.SchemaFile, cfg.Action)
		default:
			return registerSchemasFromDir(ctx, logger, remoteRegistry, version, cfg.SchemaDir)
		}
	},
}

func registerSingleSchema(
	ctx context.Context,
	logger *zap.Logger,
	registry schema_registry.SchemaRegistry,
	version ocpp.Version,
	filePath, action string,
) error {
	logger.Info("Registering schema",
		zap.String("file", filePath),
		zap.String("action", action),
		zap.String("version", version.String()))

	schemaData, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to read schema file: %s", filePath)
	}

	if err := registry.RegisterSchema(ctx, version, action, schemaData); err != nil {
		return errors.Wrapf(err, "failed to register schema for action %s", action)
	}

	logger.Info("Successfully registered schema",
		zap.String("action", action),
		zap.String("version", version.String()))
	return nil
}

func registerSchemasFromDir(
	ctx context.Context,
	logger *zap.Logger,
	registry schema_registry.SchemaRegistry,
	version ocpp.Version,
	dir string,
) error {
	logger.Info("Registering schemas from directory",
		zap.String("directory", dir),
		zap.String("version", version.String()))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to read directory: %s", dir)
	}

	successCount := 0
	errorCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if !strings.HasSuffix(strings.ToLower(fileName), ".json") {
			logger.Debug("Skipping non-JSON file", zap.String("file", fileName))
			continue
		}

		action, _ := strings.CutSuffix(fileName, ".json")

		schemaPath := filepath.Join(dir, fileName)
		schemaData, err := os.ReadFile(schemaPath)
		if err != nil {
			logger.Error("Failed to read schema file",
				zap.String("file", schemaPath),
				zap.Error(err))
			errorCount++
			continue
		}

		if err := registry.RegisterSchema(ctx, version, action, schemaData); err != nil {
			logger.Error("Failed to register schema",
				zap.String("file", schemaPath),
				zap.String("action", action),
				zap.Error(err))
			errorCount++
			continue
		}

		logger.Debug("Successfully registered schema",
			zap.String("action", action),
			zap.String("file", fileName))
		successCount++
	}

	if successCount > 0 && errorCount == 0 {
		logger.Info("Successfully registered schemas", zap.Int("successful", successCount))
	}

	if errorCount > 0 {
		logger.Info("Registered schemas with failures", zap.Int("successful", successCount), zap.Int("failed", errorCount))
	}

	return nil
}

func loadRegisterConfig() registerConfig {
	return registerConfig{
		SchemaFile: viper.GetString("schema.register.file"),
		SchemaDir:  viper.GetString("schema.register.dir"),
		Action:     viper.GetString("schema.register.action"),
	}
}

func init() {
	register.Flags().StringVarP(&registerCfg.SchemaFile, "file", "f", "", "Path to a single schema file to register")
	register.Flags().StringVar(&registerCfg.SchemaDir, "dir", "", "Path to a directory containing schema files to register")
	register.Flags().StringVarP(&registerCfg.Action, "action", "a", "", "OCPP action name (required when using --file, e.g., 'BootNotificationRequest')")

	_ = viper.BindPFlag("schema.register.file", register.Flags().Lookup("file"))
	_ = viper.BindPFlag("schema.register.dir", register.Flags().Lookup("dir"))
	_ = viper.BindPFlag("schema.register.action", register.Flags().Lookup("action"))
}
