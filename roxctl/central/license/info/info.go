package info

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/hako/durafmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/pkg/license"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	description = "Get information about a license"
)

// Command defines the command.. See usage strings for details.
func Command() *cobra.Command {
	var licenseData []byte
	c := &cobra.Command{
		Use:   "info",
		Short: description,
		Long:  description,
		RunE: func(c *cobra.Command, _ []string) error {
			if len(licenseData) == 0 {
				return errors.New("no license data supplied")
			}
			return infoLicense(licenseData)
		},
	}

	c.Flags().Var(&flags.LicenseVar{Data: &licenseData}, "license", flags.LicenseUsage)
	return c
}

func infoLicense(licenseData []byte) error {
	protoBytes, _, err := license.ParseLicenseKey(string(licenseData))
	if err != nil {
		return errors.Wrap(err, "failed to parse license key")
	}

	license, err := license.UnmarshalLicense(protoBytes)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal license key")
	}

	printLicense(os.Stdout, license)
	return nil
}

func printLicense(w io.Writer, license *licenseproto.License) {
	metadata := license.GetMetadata()
	fmt.Fprintln(w, "Metadata")
	fmt.Fprintln(w, "========")
	fmt.Fprintf(w, "  Customer Name ..... %s\n", metadata.GetLicensedForName())
	fmt.Fprintf(w, "  Customer ID ....... %s\n", metadata.GetLicensedForId())
	fmt.Fprintf(w, "  License ID ........ %s\n", metadata.GetId())
	fmt.Fprintf(w, "  Issued On ......... %s\n", formatTimestamp(metadata.GetIssueDate()))
	fmt.Fprintf(w, "  Signing Key ID .... %s\n", metadata.GetSigningKeyId())
	fmt.Fprintln(w)

	restrictions := license.GetRestrictions()
	fmt.Fprintln(w, "Restrictions")
	fmt.Fprintln(w, "============")
	fmt.Fprintf(w, "  Not Valid Before .. %s\n", formatTimestamp(restrictions.GetNotValidBefore()))
	fmt.Fprintf(w, "  Not Valid After ... %s\n", formatTimestamp(restrictions.GetNotValidAfter()))
	fmt.Fprintf(w, "  Duration .......... %s\n", formatDelta(restrictions.GetNotValidBefore(), restrictions.GetNotValidAfter()))

	if restrictions.GetNoBuildFlavorRestriction() {
		fmt.Fprintln(w, "  Build Flavors ..... Unrestricted")
	} else {
		fmt.Fprintf(w, "  Build Flavors ..... %s\n", formatList(restrictions.GetBuildFlavors()))
	}

	if restrictions.GetAllowOffline() {
		fmt.Fprintln(w, "  Enforcement ....... Offline")
	} else {
		fmt.Fprintf(w, "  Enforcement ....... %s\n", restrictions.GetEnforcementUrl())
	}

	if restrictions.GetNoDeploymentEnvironmentRestriction() {
		fmt.Fprintln(w, "  Environments ...... Unrestricted")
	} else {
		fmt.Fprintf(w, "  Environments ...... %s\n", formatList(restrictions.GetDeploymentEnvironments()))
	}

	if restrictions.GetNoNodeRestriction() {
		fmt.Fprintln(w, "  Node Count ........ Unlimited")
	} else {
		fmt.Fprintf(w, "  Node Count ........ %d\n", restrictions.GetMaxNodes())
	}
}

func formatTimestamp(timestamp *types.Timestamp) string {
	now := types.TimestampNow()
	if timestamp.Compare(now) <= 0 {
		return fmt.Sprintf("%v (%s ago)", timestamp, formatDelta(timestamp, now))
	}
	return fmt.Sprintf("%v (%s from now)", timestamp, formatDelta(now, timestamp))
}

func formatDelta(start *types.Timestamp, end *types.Timestamp) string {
	startTs, _ := types.TimestampFromProto(start)
	endTs, _ := types.TimestampFromProto(end)
	delta := endTs.Sub(startTs)
	return durafmt.ParseShort(delta).String()
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
