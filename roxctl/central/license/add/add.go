package add

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

type centralLicenseAddCommand struct {
	// Properties that are bound to cobra flags.
	licenseData []byte
	activate    bool

	// Properties that are injected or constructed.
	env     environment.Environment
	timeout time.Duration
}

// Command defines the command. See usage strings for details.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	centralLicenseAddCmd := &centralLicenseAddCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use: "add",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if err := centralLicenseAddCmd.construct(c); err != nil {
				return err
			}
			if err := centralLicenseAddCmd.validate(c); err != nil {
				return err
			}
			return centralLicenseAddCmd.addLicense()
		}),
	}

	c.Flags().Var(&flags.LicenseVar{Data: &centralLicenseAddCmd.licenseData}, "license", flags.LicenseUsage)
	c.Flags().BoolVarP(&centralLicenseAddCmd.activate, "activate", "a", false, "whether to immediately activate the passed-in license")
	return c
}

func (cmd *centralLicenseAddCommand) construct(cbr *cobra.Command) error {
	cmd.timeout = flags.Timeout(cbr)
	return nil
}

func (cmd *centralLicenseAddCommand) validate(cbr *cobra.Command) error {
	if len(cmd.licenseData) == 0 {
		return errors.New("no license data supplied")
	}
	return nil
}

func (cmd *centralLicenseAddCommand) addLicense() error {
	// Create the connection to the central detection service.
	conn, err := cmd.env.GRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	service := v1.NewLicenseServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	response, err := service.AddLicense(ctx, &v1.AddLicenseRequest{LicenseKey: string(cmd.licenseData), Activate: cmd.activate})
	if err != nil {
		return err
	}
	if !response.GetAccepted() {
		return fmt.Errorf("license was not accepted (%s): %s ", response.GetLicense().GetStatus(), response.GetLicense().GetStatusReason())
	}

	cmd.env.Logger().PrintfLn("License was accepted. License status: %s", response.GetLicense().GetStatus())
	if response.GetLicense().GetActive() {
		cmd.env.Logger().PrintfLn("The license has been activated.")
	}
	return nil
}
