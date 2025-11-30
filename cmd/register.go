package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/schema_registry/registries"
)

type registerConfig struct {
	URL          string
	Username     string
	Password     string
	BearerToken  string
	APIKey       string
	APIKeyHeader string
	CustomHeader string
	CustomValue  string
	Timeout      time.Duration
	SchemaFile   string
	SchemaDir    string
	Action       string
}

var registerCfg = registerConfig{
	Timeout: 5 * time.Second,
}

var register = &cobra.Command{
	Use:   "register",
	Short: "Register schemas on a remote schema registry",
	Long: `Register OCPP schemas on a remote schema registry. 
You can register a single schema file or all schemas from a directory.
The schema file names should match the OCPP action names (e.g., "BootNotificationRequest.json" or "BootNotificationResponse.json").`,
	Example: `  # Register a single schema file
  chargeflow --version 1.6 register --url http://localhost:8081 --file BootNotificationRequest.json --action BootNotificationRequest

  # Register all schemas from a directory
  chargeflow --version 1.6 register --url http://localhost:8081 --dir ./schemas

  # Register with basic authentication
  chargeflow register --url http://localhost:8081 --username admin --password secret --file schema.json --action BootNotificationRequest

  # Register with bearer token
  chargeflow register --url http://localhost:8081 --bearer-token token123 --file schema.json --action BootNotificationRequest

  # Register with API key
  chargeflow register --url http://localhost:8081 --api-key key123 --api-key-header X-API-Key --file schema.json --action BootNotificationRequest`,
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadRegisterConfig()

		if cfg.URL == "" {
			return errors.New("remote registry URL is required (use --url flag)")
		}

		if cfg.SchemaFile == "" && cfg.SchemaDir == "" {
			return errors.New("either --file or --dir must be specified")
		}

		if cfg.SchemaFile != "" && cfg.SchemaDir != "" {
			return errors.New("cannot specify both --file and --dir")
		}

		if cfg.SchemaFile != "" && cfg.Action == "" {
			return errors.New("--action is required when using --file")
		}

		// Validate authentication options
		if cfg.Username != "" || cfg.Password != "" {
			if cfg.Username == "" || cfg.Password == "" {
				return errors.New("both --username and --password are required for basic authentication")
			}
		}
		if cfg.CustomHeader != "" || cfg.CustomValue != "" {
			if cfg.CustomHeader == "" || cfg.CustomValue == "" {
				return errors.New("both --custom-header and --custom-value are required for custom header authentication")
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := zap.L()

		cfg := loadRegisterConfig()
		ocppVersion := viper.GetString("ocpp.version")
		version := ocpp.Version(ocppVersion)

		// Build remote registry options
		opts := []registries.RemoteOptions{
			registries.WithTimeout(cfg.Timeout),
		}

		// Add authentication option
		switch {
		case cfg.Username != "" && cfg.Password != "":
			opts = append(opts, registries.WithBasicAuth(cfg.Username, cfg.Password))
		case cfg.BearerToken != "":
			opts = append(opts, registries.WithBearerToken(cfg.BearerToken))
		case cfg.APIKey != "":
			opts = append(opts, registries.WithAPIKey(cfg.APIKey, cfg.APIKeyHeader))
		case cfg.CustomHeader != "" && cfg.CustomValue != "":
			opts = append(opts, registries.WithCustomHeader(cfg.CustomHeader, cfg.CustomValue))
		}

		// Create remote registry
		remoteRegistry, err := registries.NewRemoteSchemaRegistry(cfg.URL, logger, opts...)
		if err != nil {
			return errors.Wrap(err, "failed to create remote schema registry")
		}

		// Register schema(s)
		switch {
		case cfg.SchemaFile != "":
			// Register single schema file
			return registerSingleSchema(logger, remoteRegistry, version, cfg.SchemaFile, cfg.Action)
		default:
			// Register all schemas from directory
			return registerSchemasFromDir(logger, remoteRegistry, version, cfg.SchemaDir)
		}
	},
}

func registerSingleSchema(logger *zap.Logger, registry *registries.RemoteSchemaRegistry, version ocpp.Version, filePath, action string) error {
	logger.Info("Registering schema",
		zap.String("file", filePath),
		zap.String("action", action),
		zap.String("version", version.String()))

	schemaData, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to read schema file: %s", filePath)
	}

	if err := registry.RegisterSchema(version, action, schemaData); err != nil {
		return errors.Wrapf(err, "failed to register schema for action %s", action)
	}

	logger.Info("Successfully registered schema",
		zap.String("action", action),
		zap.String("version", version.String()))
	return nil
}

// registerSchemasFromDir registers all schemas from a directory to the given registry.
// This function is shared between validate and register commands.
func registerSchemasFromDir(logger *zap.Logger, registry schema_registry.SchemaRegistry, version ocpp.Version, dir string) error {
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

		// Extract action name from filename (remove .json extension)
		action, _ := strings.CutSuffix(fileName, ".json")

		// Read schema file
		schemaPath := filepath.Join(dir, fileName)
		schemaData, err := os.ReadFile(schemaPath)
		if err != nil {
			logger.Error("Failed to read schema file",
				zap.String("file", schemaPath),
				zap.Error(err))
			errorCount++
			continue
		}

		// Register schema
		if err := registry.RegisterSchema(version, action, schemaData); err != nil {
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

	if errorCount > 0 {
		return errors.Errorf("failed to register %d schema(s), %d succeeded", errorCount, successCount)
	}

	logger.Info("Successfully registered schemas", zap.Int("count", successCount))
	return nil
}

// loadRegisterConfig loads configuration from viper with fallback to flag values.
func loadRegisterConfig() registerConfig {
	cfg := registerConfig{
		URL:          getStringOrDefault("register.url", registerCfg.URL),
		Username:     getStringOrDefault("register.username", registerCfg.Username),
		Password:     getStringOrDefault("register.password", registerCfg.Password),
		BearerToken:  getStringOrDefault("register.bearer-token", registerCfg.BearerToken),
		APIKey:       getStringOrDefault("register.api-key", registerCfg.APIKey),
		APIKeyHeader: getStringOrDefault("register.api-key-header", registerCfg.APIKeyHeader),
		CustomHeader: getStringOrDefault("register.custom-header", registerCfg.CustomHeader),
		CustomValue:  getStringOrDefault("register.custom-value", registerCfg.CustomValue),
		SchemaFile:   getStringOrDefault("register.file", registerCfg.SchemaFile),
		SchemaDir:    getStringOrDefault("register.dir", registerCfg.SchemaDir),
		Action:       getStringOrDefault("register.action", registerCfg.Action),
		Timeout:      getDurationOrDefault("register.timeout", registerCfg.Timeout),
	}

	// Set default API key header if API key is provided but header is not
	if cfg.APIKey != "" && cfg.APIKeyHeader == "" {
		cfg.APIKeyHeader = "X-API-Key"
	}

	return cfg
}

// getStringOrDefault returns the viper string value or the default if empty.
func getStringOrDefault(key string, defaultValue string) string {
	if value := viper.GetString(key); value != "" {
		return value
	}
	return defaultValue
}

// getDurationOrDefault returns the viper duration value or the default if zero.
func getDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := viper.GetDuration(key); value != 0 {
		return value
	}
	return defaultValue
}

func init() {
	// Registry URL
	register.Flags().StringVar(&registerCfg.URL, "url", "", "Remote schema registry URL (required)")

	// Authentication options
	register.Flags().StringVar(&registerCfg.Username, "username", "", "Username for basic authentication")
	register.Flags().StringVar(&registerCfg.Password, "password", "", "Password for basic authentication")
	register.Flags().StringVar(&registerCfg.BearerToken, "bearer-token", "", "Bearer token for authentication")
	register.Flags().StringVar(&registerCfg.APIKey, "api-key", "", "API key for authentication")
	register.Flags().StringVar(&registerCfg.APIKeyHeader, "api-key-header", "X-API-Key", "Header name for API key authentication")
	register.Flags().StringVar(&registerCfg.CustomHeader, "custom-header", "", "Custom header name for authentication")
	register.Flags().StringVar(&registerCfg.CustomValue, "custom-value", "", "Custom header value for authentication")

	// Schema input options
	register.Flags().StringVarP(&registerCfg.SchemaFile, "file", "f", "", "Path to a single schema file to register")
	register.Flags().StringVar(&registerCfg.SchemaDir, "dir", "", "Path to a directory containing schema files to register")
	register.Flags().StringVarP(&registerCfg.Action, "action", "a", "", "OCPP action name (required when using --file, e.g., 'BootNotificationRequest')")

	// Timeout option
	register.Flags().DurationVar(&registerCfg.Timeout, "timeout", 5*time.Second, "Request timeout duration")

	// Bind flags to viper
	_ = viper.BindPFlag("register.url", register.Flags().Lookup("url"))
	_ = viper.BindPFlag("register.username", register.Flags().Lookup("username"))
	_ = viper.BindPFlag("register.password", register.Flags().Lookup("password"))
	_ = viper.BindPFlag("register.bearer-token", register.Flags().Lookup("bearer-token"))
	_ = viper.BindPFlag("register.api-key", register.Flags().Lookup("api-key"))
	_ = viper.BindPFlag("register.api-key-header", register.Flags().Lookup("api-key-header"))
	_ = viper.BindPFlag("register.custom-header", register.Flags().Lookup("custom-header"))
	_ = viper.BindPFlag("register.custom-value", register.Flags().Lookup("custom-value"))
	_ = viper.BindPFlag("register.file", register.Flags().Lookup("file"))
	_ = viper.BindPFlag("register.dir", register.Flags().Lookup("dir"))
	_ = viper.BindPFlag("register.action", register.Flags().Lookup("action"))
	_ = viper.BindPFlag("register.timeout", register.Flags().Lookup("timeout"))
}
