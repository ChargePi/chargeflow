package cmd

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/schema_registry/registries/remote_registry"
)

type schemaConfig struct {
	URL          string
	AuthType     string
	Username     string
	Password     string
	BearerToken  string
	APIKey       string
	APIKeyHeader string
	CustomHeader string
	CustomValue  string
	Timeout      time.Duration
}

var schemaCfg schemaConfig

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage schemas on a remote schema registry",
	Long:  `Commands for registering and removing OCPP schemas on a remote schema registry.`,
}

// loadSchemaConfig loads common schema registry configuration from viper.
func loadSchemaConfig() schemaConfig {
	cfg := schemaConfig{
		URL:          viper.GetString("schema.url"),
		AuthType:     viper.GetString("schema.auth-type"),
		Username:     viper.GetString("schema.username"),
		Password:     viper.GetString("schema.password"),
		BearerToken:  viper.GetString("schema.bearer-token"),
		APIKey:       viper.GetString("schema.api-key"),
		APIKeyHeader: viper.GetString("schema.api-key-header"),
		CustomHeader: viper.GetString("schema.custom-header"),
		CustomValue:  viper.GetString("schema.custom-value"),
		Timeout:      viper.GetDuration("schema.timeout"),
	}

	if cfg.APIKey != "" && cfg.APIKeyHeader == "" {
		cfg.APIKeyHeader = "X-API-Key"
	}

	return cfg
}

// validateSchemaConfig validates the common schema registry flags.
// Called from each subcommand's PreRunE to avoid shadowing rootCmd's PersistentPreRun.
func validateSchemaConfig() error {
	cfg := loadSchemaConfig()

	if cfg.URL == "" {
		return errors.New("remote registry URL is required (use --url flag)")
	}

	switch cfg.AuthType {
	case "basic":
		if cfg.Username == "" || cfg.Password == "" {
			return errors.New("both --username and --password are required for basic authentication")
		}
	case "bearer":
		if cfg.BearerToken == "" {
			return errors.New("Bearer token is required for API-key authentication")
		}
	case "api-key":
		if cfg.APIKey == "" {
			return errors.New("API-key is required for API-key authentication")
		}
	}

	return nil
}

// buildRemoteRegistry creates a remote schema registry from the common schema config.
func buildRemoteRegistry(logger *zap.Logger) (*remote_registry.SchemaRegistry, error) {
	cfg := loadSchemaConfig()

	opts := []remote_registry.Options{
		remote_registry.WithTimeout(cfg.Timeout),
	}

	switch cfg.AuthType {
	case "basic":
		opts = append(opts, remote_registry.WithBasicAuth(cfg.Username, cfg.Password))
	case "bearer":
		opts = append(opts, remote_registry.WithBearerToken(cfg.BearerToken))
	case "api-key":
		opts = append(opts, remote_registry.WithAPIKey(cfg.APIKey, cfg.APIKeyHeader))
	}

	if cfg.CustomHeader != "" && cfg.CustomValue != "" {
		opts = append(opts, remote_registry.WithCustomHeader(cfg.CustomHeader, cfg.CustomValue))
	}

	return remote_registry.NewRemoteSchemaRegistry(cfg.URL, logger, opts...)
}

func init() {
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.URL, "url", "", "Remote schema registry URL (required)")
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.AuthType, "auth-type", "", "Authentication type (basic, bearer, api-key or none)")
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.Username, "username", "", "Username for basic authentication")
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.Password, "password", "", "Password for basic authentication")
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.BearerToken, "bearer-token", "", "Bearer token for authentication")
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.APIKey, "api-key", "", "API key for authentication")
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.APIKeyHeader, "api-key-header", "X-API-Key", "Header name for API key authentication")
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.CustomHeader, "custom-header", "", "Custom header name for authentication")
	schemaCmd.PersistentFlags().StringVar(&schemaCfg.CustomValue, "custom-value", "", "Custom header value for authentication")
	schemaCmd.PersistentFlags().DurationVar(&schemaCfg.Timeout, "timeout", 5*time.Second, "Request timeout duration")

	_ = viper.BindPFlag("schema.url", schemaCmd.PersistentFlags().Lookup("url"))
	_ = viper.BindPFlag("schema.auth-type", schemaCmd.PersistentFlags().Lookup("auth-type"))
	_ = viper.BindPFlag("schema.username", schemaCmd.PersistentFlags().Lookup("username"))
	_ = viper.BindPFlag("schema.password", schemaCmd.PersistentFlags().Lookup("password"))
	_ = viper.BindPFlag("schema.bearer-token", schemaCmd.PersistentFlags().Lookup("bearer-token"))
	_ = viper.BindPFlag("schema.api-key", schemaCmd.PersistentFlags().Lookup("api-key"))
	_ = viper.BindPFlag("schema.api-key-header", schemaCmd.PersistentFlags().Lookup("api-key-header"))
	_ = viper.BindPFlag("schema.custom-header", schemaCmd.PersistentFlags().Lookup("custom-header"))
	_ = viper.BindPFlag("schema.custom-value", schemaCmd.PersistentFlags().Lookup("custom-value"))
	_ = viper.BindPFlag("schema.timeout", schemaCmd.PersistentFlags().Lookup("timeout"))

	schemaCmd.AddCommand(register)
	schemaCmd.AddCommand(removeCmd)
}
