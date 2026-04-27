package cmd

import (
	"context"

	"go.uber.org/zap"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

const (
	serviceVersion = "0.1.0-beta"
)

var (
	vendor = ""
	model  = ""
)

var rootCmd = &cobra.Command{
	Use:     "chargeflow",
	Short:   "",
	Long:    ``,
	Version: serviceVersion,
	Run:     func(cmd *cobra.Command, args []string) {},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("debug") {
			l, _ := zap.NewDevelopment()
			zap.ReplaceGlobals(l)
		}
	},
}

func init() {
	rootCmd.AddCommand(validate)
	rootCmd.AddCommand(schemaCmd)
}

// setDefaults sets the default values for the configuration.
func setDefaults() {
	viper.SetDefault("ocpp.version", ocpp.V16.String())
	viper.SetDefault("debug", false)
}

func rootFlags() {
	// Add flag for OCPP version
	rootCmd.PersistentFlags().StringP("version", "v", ocpp.V16.String(), "OCPP version to use (1.6, 2.0.1 or 2.1)")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVarP(&vendor, "vendor", "V", "", "Charging-station vendor for vendor/model-specific schema selection")
	rootCmd.PersistentFlags().StringVarP(&model, "model", "m", "", "Charging-station model for vendor/model-specific schema selection")

	_ = viper.BindPFlag("ocpp.version", rootCmd.PersistentFlags().Lookup("version"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("vendor", rootCmd.PersistentFlags().Lookup("vendor"))
	_ = viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))
}

func Execute(ctx context.Context) error {
	cobra.OnInitialize(setDefaults)

	rootFlags()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return errors.Wrap(err, "executing root command")
	}
	return nil
}
