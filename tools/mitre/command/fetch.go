package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/httputil"
	"github.com/stackrox/stackrox/pkg/jsonutil"
	"github.com/stackrox/stackrox/pkg/mitre"
	"github.com/stackrox/stackrox/pkg/utils"
	"github.com/stackrox/stackrox/tools/mitre/common"
)

var (
	out       string
	domain    string
	platforms []string
)

// FetchC defines the fetch command.
func FetchC() *cobra.Command {
	c := &cobra.Command{
		Use:   fmt.Sprintf("fetch --%s <domain> [--%s <platform>] [--%s <file>]", common.DomainFlagName, common.PlatformFlagName, common.OutputFlagName),
		Short: "Fetch most recent MITRE Tactics and (Sub-)Techniques",
		RunE:  fetchC,
	}

	c.PersistentFlags().StringVarP(
		&out,
		common.OutputFlagName,
		common.OutputFlagShorthand,
		common.DefaultOutFile,
		"output file")

	c.PersistentFlags().StringVarP(
		&domain,
		common.DomainFlagName,
		"",
		"",
		fmt.Sprintf("MITRE ATT&CK domain %+s [required]", common.MitreDomainsCmdArgs))

	c.PersistentFlags().StringArrayVarP(
		&platforms,
		common.PlatformFlagName,
		"",
		[]string{"all"},
		fmt.Sprintf("MITRE ATT&CK platform %+s", common.MitrePlatformsCmdArgs))

	return c
}

func fetchC(_ *cobra.Command, _ []string) error {
	var multiErr *multierror.Error
	if out == "" {
		_, _ = os.Stdout.WriteString(
			fmt.Sprintf("'--%s' is not set; setting to default %s\n", common.OutputFlagName, common.DefaultOutFile),
		)
		out = common.DefaultOutFile
	}

	if len(domain) == 0 {
		multiErr = multierror.Append(multiErr,
			errors.Errorf("'--%s' must be specified %+s", common.DomainFlagName, common.MitreDomainsCmdArgs),
		)
	}

	if len(platforms) == 0 {
		multiErr = multierror.Append(multiErr,
			errors.Errorf("'--%s' must be %+s or all", common.PlatformFlagName, common.MitrePlatformsCmdArgs),
		)
	}

	if multiErr != nil {
		return multiErr
	}

	if len(platforms) == 1 && platforms[0] == "all" {
		platforms = common.MitrePlatformsCmdArgs
	}
	bundle, err := fetch(domain, platforms)
	if err != nil {
		return err
	}

	str, err := jsonutil.ProtoToJSON(bundle, jsonutil.OptUnEscape)
	if err != nil {
		return errors.Wrap(err, "marshalling parsed MITRE ATT&CK bundle")
	}

	if err := os.WriteFile(out, []byte(str), 0644); err != nil {
		return errors.Wrapf(err, "writing MITRE ATT&CK bundle to file %q", out)
	}
	return nil
}

func fetch(domain string, platforms []string) (*storage.MitreAttackBundle, error) {
	data, mitreDomain, err := fetchForDomain(domain)
	if err != nil {
		utils.Should(err)
		return nil, nil
	}

	mitrePlatforms, err := getMitrePlatforms(platforms...)
	if err != nil {
		utils.Should(err)
		return nil, nil
	}

	bundle, err := mitre.UnmarshalAndExtractMitreAttackBundle(mitreDomain, mitrePlatforms, data)
	if err != nil {
		return nil, errors.Wrap(err, "parsing fetched MITRE ATT&CK data")
	}

	return bundle, nil
}

func fetchForDomain(domain string) ([]byte, mitre.Domain, error) {
	val := common.CmdArgMitreDomainMap[domain]
	if val == nil {
		return nil, mitre.Domain(""), errors.Errorf("MITRE ATT&CK domain for command arg %q not found", domain)
	}

	data, err := httputil.HTTPGet(val.URL)
	if err != nil {
		return nil, mitre.Domain(""), errors.Wrapf(err, "getting %q", common.MitreEnterpriseAttackSrcURL)
	}

	return data, val.Domain, nil
}

func getMitrePlatforms(platforms ...string) ([]mitre.Platform, error) {
	convertedPlatform := make([]mitre.Platform, 0, len(platforms))
	for _, p := range platforms {
		val := common.CmdArgMitrePlatformMap[p]
		if len(val) == 0 {
			return nil, errors.Errorf("MITRE ATT&CK platform for command arg %q not found", p)
		}
		convertedPlatform = append(convertedPlatform, val)
	}
	return convertedPlatform, nil
}
