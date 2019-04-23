package add

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	description = "Add a license"
)

// Command defines the command. See usage strings for details.
func Command() *cobra.Command {
	var licenseData []byte
	var activate bool
	c := &cobra.Command{
		Use:   "add",
		Short: description,
		Long:  description,
		RunE: func(c *cobra.Command, _ []string) error {
			if len(licenseData) == 0 {
				return errors.New("no license data supplied")
			}
			timeout := flags.Timeout(c)
			return addLicense(licenseData, activate, timeout)
		},
	}

	c.Flags().Var(&flags.LicenseVar{Data: &licenseData}, "license", flags.LicenseUsage)
	c.Flags().BoolVarP(&activate, "activate", "a", false, "whether to immediately activate the passed-in license")
	return c
}

func addLicense(licenseData []byte, activate bool, timeout time.Duration) error {
	// Create the connection to the central detection service.
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	service := v1.NewLicenseServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := service.AddLicense(ctx, &v1.AddLicenseRequest{LicenseKey: string(licenseData), Activate: activate})
	if err != nil {
		return err
	}
	if !response.GetAccepted() {
		return fmt.Errorf("license was not accepted (%s): %s ", response.GetLicense().GetStatus(), response.GetLicense().GetStatusReason())
	}

	fmt.Printf("License was accepted. License status: %s\n", response.GetLicense().GetStatus())
	if response.GetLicense().GetActive() {
		fmt.Println("The license has been activated.")
	}
	return nil
}
