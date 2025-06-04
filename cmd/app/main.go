package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	devxCfg "github.com/xBlaz3kx/DevX/configuration"
	devxHttp "github.com/xBlaz3kx/DevX/http"
	"github.com/xBlaz3kx/DevX/observability"
	"go.uber.org/zap"
)

const (
	serviceName    = "local-dev"
	serviceVersion = "1.0.0-beta"
)

type Configuration struct {
	Database      devxCfg.Database       `json:"database"         yaml:"database"         mapstructure:"database"         validate:"required"`
	Observability observability.Config   `json:"observability"    yaml:"observability"    mapstructure:"observability"    validate:"required"`
	Http          devxHttp.Configuration `json:"http"             yaml:"http"             mapstructure:"http"             validate:"required"`
	// Unleash       unleash.Configuration  `json:"unleash"          yaml:"unleash"          mapstructure:"unleash"          validate:"required"`
}

var (
	configurationFile string

	rootCmd = &cobra.Command{
		Use:     "local",
		Short:   "All in one service for simplified local development.",
		Long:    `Note: This will not run the OCPP service and does require a RabbitMQ instance to be running.`,
		Version: serviceVersion,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			cfg := getConfiguration()
			// Run the app
		},
	}
)

// InitConfig sets up the environment and loads the configuration from a file if the path is provided.
func InitConfig(configurationFilePath string) {
	devxCfg.SetupEnv(serviceName)
	setDefaults()
	devxCfg.InitConfig(configurationFilePath, "$HOME/chargex/", "/usr/chargex/config/")
}

// setDefaults sets the default values for the configuration.
func setDefaults() {
	devxCfg.SetDefaults(serviceName)
	viper.SetDefault(devxCfg.ServerAddress, "localhost:80")
	viper.SetDefault(devxCfg.ServerPathPrefix, "/")
}

// getConfiguration get the configuration from cache or file.
func getConfiguration() *Configuration {
	zap.L().Info("Getting configuration")
	defer zap.L().Info("Loaded and validated configuration!")

	var config Configuration

	devxCfg.GetConfiguration(viper.GetViper(), &config)

	return &config
}

func setupGlobalLogger() {
	logger, _ := zap.NewProduction()
	zap.ReplaceGlobals(logger)
}

func initConfig() {
	InitConfig(configurationFile)
}

func main() {
	cobra.OnInitialize(setupGlobalLogger, initConfig)
	configuration.SetupFlags(rootCmd, &configurationFile)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGKILL,
	)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		zap.L().Fatal("Unable to run", zap.Error(err))
	}
}
