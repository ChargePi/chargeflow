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

type removeConfig struct {
	SchemaFile string
	SchemaDir  string
	Action     string
}

var removeCfg removeConfig

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove schemas from a remote schema registry",
	Long: `Remove OCPP schemas from a remote schema registry.
You can remove a single schema by action name, derive the action from a file name, or remove all schemas matching a directory.`,
	Example: `  # Remove a single schema by action name
  chargeflow schema --url http://localhost:8081 --version 1.6 remove --action BootNotificationRequest

  # Remove a schema using a file name as the action
  chargeflow schema --url http://localhost:8081 --version 1.6 remove --file BootNotificationRequest.json

  # Remove all schemas matching files in a directory
  chargeflow schema --url http://localhost:8081 --version 1.6 remove --dir ./schemas

  # Remove with basic authentication
  chargeflow schema --url http://localhost:8081 --auth-type basic --username admin --password secret remove --action BootNotificationRequest`,
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := validateSchemaConfig(); err != nil {
			return err
		}

		cfg := loadRemoveConfig()

		specified := 0
		if cfg.Action != "" {
			specified++
		}
		if cfg.SchemaFile != "" {
			specified++
		}
		if cfg.SchemaDir != "" {
			specified++
		}

		if specified == 0 {
			return errors.New("one of --action, --file or --dir must be specified")
		}
		if specified > 1 {
			return errors.New("only one of --action, --file or --dir may be specified")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := zap.L()

		cfg := loadRemoveConfig()
		version := ocpp.Version(viper.GetString("ocpp.version"))

		remoteRegistry, err := buildRemoteRegistry(logger)
		if err != nil {
			return errors.Wrap(err, "failed to create remote schema registry")
		}

		ctx := cmd.Context()

		switch {
		case cfg.Action != "":
			return removeSingleSchema(ctx, logger, remoteRegistry, version, cfg.Action)
		case cfg.SchemaFile != "":
			action, _ := strings.CutSuffix(filepath.Base(cfg.SchemaFile), ".json")
			return removeSingleSchema(ctx, logger, remoteRegistry, version, action)
		default:
			return removeSchemasFromDir(ctx, logger, remoteRegistry, version, cfg.SchemaDir)
		}
	},
}

func removeSingleSchema(
	ctx context.Context,
	logger *zap.Logger,
	registry schema_registry.SchemaRegistry,
	version ocpp.Version,
	action string,
) error {
	logger.Info("Removing schema",
		zap.String("action", action),
		zap.String("version", version.String()))

	if err := registry.DeleteSchema(ctx, version, action); err != nil {
		return errors.Wrapf(err, "failed to remove schema for action %s", action)
	}

	logger.Info("Successfully removed schema",
		zap.String("action", action),
		zap.String("version", version.String()))
	return nil
}

func removeSchemasFromDir(
	ctx context.Context,
	logger *zap.Logger,
	registry schema_registry.SchemaRegistry,
	version ocpp.Version,
	dir string,
) error {
	logger.Info("Removing schemas matching directory",
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

		if err := registry.DeleteSchema(ctx, version, action); err != nil {
			logger.Error("Failed to remove schema",
				zap.String("action", action),
				zap.String("file", fileName),
				zap.Error(err))
			errorCount++
			continue
		}

		logger.Debug("Successfully removed schema",
			zap.String("action", action),
			zap.String("file", fileName))
		successCount++
	}

	if successCount > 0 && errorCount == 0 {
		logger.Info("Successfully removed schemas", zap.Int("successful", successCount))
	}

	if errorCount > 0 {
		logger.Info("Removed schemas with failures", zap.Int("successful", successCount), zap.Int("failed", errorCount))
	}

	return nil
}

func loadRemoveConfig() removeConfig {
	return removeConfig{
		SchemaFile: viper.GetString("schema_remove.file"),
		SchemaDir:  viper.GetString("schema_remove.dir"),
		Action:     viper.GetString("schema_remove.action"),
	}
}

func init() {
	removeCmd.Flags().StringVarP(&removeCfg.Action, "action", "a", "", "OCPP action name to remove (e.g., 'BootNotificationRequest')")
	removeCmd.Flags().StringVarP(&removeCfg.SchemaFile, "file", "f", "", "Path to a schema file; the file name (without .json) is used as the action name")
	removeCmd.Flags().StringVar(&removeCfg.SchemaDir, "dir", "", "Path to a directory; removes schemas matching each JSON file name")

	_ = viper.BindPFlag("schema_remove.action", removeCmd.Flags().Lookup("action"))
	_ = viper.BindPFlag("schema_remove.file", removeCmd.Flags().Lookup("file"))
	_ = viper.BindPFlag("schema_remove.dir", removeCmd.Flags().Lookup("dir"))
}
