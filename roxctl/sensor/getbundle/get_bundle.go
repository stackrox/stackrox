package getbundle

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/sensor/util"
)

func downloadBundle(outputDir, clusterIDOrName string, timeout time.Duration,
	createUpgraderSA bool, istioVersion string, env environment.Environment,
) error {
	clusterID, err := util.ResolveClusterID(clusterIDOrName, timeout, 20*time.Second, env)
	if err != nil {
		return errors.Wrapf(err, "error resolving cluster ID for %q", clusterIDOrName)
	}

	params := apiparams.ClusterZip{
		ID:               clusterID,
		CreateUpgraderSA: &createUpgraderSA,
		IstioVersion:     istioVersion,
	}

	if err := util.GetBundle(params, outputDir, timeout, env); err != nil {
		return errors.Wrap(err, "error getting cluster zip file")
	}

	return nil
}

// Command defines the deploy command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	var createUpgraderSA bool
	var outputDir string
	var istioVersion string

	c := &cobra.Command{
		Use:   "get-bundle <cluster-name-or-id>",
		Args:  cobra.ExactArgs(1),
		Short: "Download a bundle with the files to deploy StackRox services into a cluster",
		Long:  "Download a bundle with the required YAML configuration files to deploy StackRox Sensor, Collector, and Admission controller (optional).",
		RunE: func(c *cobra.Command, args []string) error {
			if err := downloadBundle(outputDir, args[0], flags.Timeout(c), createUpgraderSA, istioVersion, cliEnvironment); err != nil {
				return errors.Wrap(err, "error downloading sensor bundle")
			}
			return nil
		},
	}

	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "Output directory for bundle contents (default: auto-generated directory name inside the current directory).")
	c.PersistentFlags().BoolVar(&createUpgraderSA, "create-upgrader-sa", true, "Whether to create the upgrader service account, with cluster-admin privileges, to facilitate automated sensor upgrades.")
	c.PersistentFlags().StringVar(&istioVersion, "istio-support", "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s.",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))

	flags.AddTimeoutWithDefault(c, 5*time.Minute)

	return c
}
