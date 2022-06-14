package add

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

type centralLicenseAddCommand struct {
	// Properties that are bound to cobra flags.
	licenseData []byte
	activate    bool

	timeout time.Duration
}

// Command defines the command. See usage strings for details.
func Command() *cobra.Command {
	centralLicenseAddCmd := &centralLicenseAddCommand{}
	c := &cobra.Command{
		Use: "add",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return nil
		}),
		Deprecated: "Licenses are no longer required",
	}

	c.Flags().Var(&flags.LicenseVar{Data: &centralLicenseAddCmd.licenseData}, "license", flags.LicenseUsage)
	c.Flags().BoolVarP(&centralLicenseAddCmd.activate, "activate", "a", false, "whether to immediately activate the passed-in license")
	return c
}
