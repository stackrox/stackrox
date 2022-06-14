package info

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

type centralLicenseInfoCommand struct {
	// Properties that are bound to cobra flags.
	licenseData []byte
	json        bool

	timeout time.Duration
}

// Command defines the command.. See usage strings for details.
func Command() *cobra.Command {
	centralLicenseInfoCmd := &centralLicenseInfoCommand{}
	c := &cobra.Command{
		Use: "info",
		RunE: util.RunENoArgs(func(cmd *cobra.Command) error {
			fmt.Fprintln(os.Stdout, "Licenses are no longer required")
			return nil
		}),
		Deprecated: "Licenses are no longer required",
	}

	c.Flags().Var(&flags.LicenseVar{Data: &centralLicenseInfoCmd.licenseData}, "license", flags.LicenseUsage)
	c.Flags().BoolVar(&centralLicenseInfoCmd.json, "json", false, "output as json")
	return c
}

func (cmd *centralLicenseInfoCommand) construct(cbr *cobra.Command) error {
	cmd.timeout = flags.Timeout(cbr)
	return nil
}

func (cmd *centralLicenseInfoCommand) validate(cbr *cobra.Command) error {
	if len(cmd.licenseData) == 0 {
		return errors.New("no license data supplied")
	}
	return nil
}

func formatList(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("Only %s", values[0])
	case 2:
		return fmt.Sprintf("Either %s or %s", values[0], values[1])
	default:
		return fmt.Sprintf("Any of %s, or %s", strings.Join(values[:len(values)-1], ", "), values[len(values)-1])
	}
}
