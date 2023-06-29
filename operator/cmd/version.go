package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:  "version",
	Long: `Print the version number of RHACS Operator`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(fullOperatorVersion())
	},
}

func fullOperatorVersion() string {
	return fmt.Sprintf("%s Operator %q, go version: %q, GOOS: %q, GOARCH: %q", branding.GetProductName(), version.GetMainVersion(), runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
