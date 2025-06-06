package cmd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	serviceVersion = "0.1.0-beta"
)

var (
	configurationFile string

	rootCmd = &cobra.Command{
		Use:     "chargeflow",
		Short:   "",
		Long:    ``,
		Version: serviceVersion,
		Run:     func(cmd *cobra.Command, args []string) {},
	}
)

func init() {
	rootCmd.AddCommand(validate)
}

// setDefaults sets the default values for the configuration.
func setDefaults() {
	viper.SetDefault("", "")
	viper.SetDefault("", "/")
}

func rootFlags() {
	rootCmd.PersistentFlags().StringVarP(
		&configurationFile,
		"config",
		"c",
		"",
		"Path to the configuration file",
	)

	rootCmd.Flags().BoolP(
		"version",
		"v",
		false,
		"Print the version of the service",
	)
}

func Execute(ctx context.Context) error {
	cobra.OnInitialize(setDefaults)

	rootFlags()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return errors.Wrap(err, "executing root command")
	}
	return nil
}
