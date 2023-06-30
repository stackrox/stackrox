package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stackrox/rox/pkg/branding"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.SetArgs(useDefaultCommand(os.Args[1:]))
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func useDefaultCommand(args []string) []string {
	cmd, _, err := rootCmd.Find(args)
	// default to start cmd if no cmd is given
	if err == nil && cmd.Use == rootCmd.Use && cmd.Flags().Parse(args) != pflag.ErrHelp {
		fmt.Println("Warning: No command specified, defaulting to 'start'. This behavior will be deprecated in the future.")
		return append([]string{startCmd.Use}, args...)
	}
	return args
}

var rootCmd = cobra.Command{
	Use:   "operator",
	Short: branding.GetProductName() + " operator",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := initConfig(cmd); err != nil {
			return err
		}
		return nil
	},
}

// initConfig reads in ENV variables if set.
func initConfig(cmd *cobra.Command) error {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv() // read in environment variables that match
	// Bind the current command's flags to viper
	return bindFlags(cmd, v)
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	var err error
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		name := strings.ReplaceAll(f.Name, "-", "_")
		if !f.Changed && v.IsSet(name) {
			val := v.Get(name)
			fmt.Println(name, val)
			setErr := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if setErr != nil && err == nil {
				err = setErr
			}
		}
	})
	return err
}
