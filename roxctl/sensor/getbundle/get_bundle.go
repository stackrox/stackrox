package getbundle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/pflag/autobool"
	"github.com/stackrox/rox/roxctl/sensor/util"
)

const (
	infoDefaultingToSlimCollector = `Specified cluster is configured to use a slim collector image, hence producing deployment bundle using a slim collector image.
Use --slim-collector=false if that is not desired.`
	infoDefaultingToComprehensiveCollector = `Specified cluster is configured to use a comprehensive collector image, hence producing deployment bundle using a comprehensive collector image.
Use --slim-collector if that is not desired.`
)

func downloadBundle(outputDir, clusterIDOrName string, timeout time.Duration, retryTimeout time.Duration,
	createUpgraderSA bool, slimCollectorP *bool, istioVersion string, env environment.Environment,
) error {
	conn, err := env.GRPCConnection(retryTimeout)
	if err != nil {
		return err
	}
	service := v1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	clusterID, err := util.ResolveClusterID(clusterIDOrName, timeout, retryTimeout, env)
	if err != nil {
		return errors.Wrapf(err, "error resolving cluster ID for %q", clusterIDOrName)
	}
	slimCollector := false
	if slimCollectorP != nil {
		slimCollector = *slimCollectorP
	} else {
		// Query Central for slimCollector property of the specified cluster.
		resp, err := service.GetCluster(ctx, &v1.ResourceByID{
			Id: clusterID,
		})
		if err != nil {
			return errors.Wrapf(err, "error resolving cluster for ID %q", clusterID)
		}
		cluster := resp.GetCluster()
		slimCollector = cluster.GetSlimCollector()
		if slimCollector {
			env.Logger().InfofLn(infoDefaultingToSlimCollector)
		} else {
			env.Logger().InfofLn(infoDefaultingToComprehensiveCollector)
		}
	}

	params := apiparams.ClusterZip{
		ID:               clusterID,
		CreateUpgraderSA: &createUpgraderSA,
		SlimCollector:    pointers.Bool(slimCollector),
		IstioVersion:     istioVersion,
	}

	if err := util.GetBundle(params, outputDir, timeout, env); err != nil {
		return errors.Wrap(err, "error getting cluster zip file")
	}

	if slimCollector {
		centralEnv, err := util.RetrieveCentralEnvOrDefault(ctx, service)
		if err != nil {
			env.Logger().WarnfLn("Sensor bundle has been created successfully, but it was not possible to retrieve Central's runtime environment information: %v.", err)
		} else if !centralEnv.KernelSupportAvailable {
			env.Logger().WarnfLn(util.WarningSlimCollectorModeWithoutKernelSupport)
		}
	}

	return nil
}

// Command defines the deploy command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	var createUpgraderSA bool
	var outputDir string
	var slimCollector *bool
	var istioVersion string

	c := &cobra.Command{
		Use:   "get-bundle <cluster-name-or-id>",
		Args:  cobra.ExactArgs(1),
		Short: "Download a bundle with the files to deploy StackRox services into a cluster.",
		Long:  "Download a bundle with the required YAML configuration files to deploy StackRox Sensor, Collector, and Admission controller (optional).",
		RunE: func(c *cobra.Command, args []string) error {
			if err := downloadBundle(outputDir, args[0], flags.Timeout(c), flags.RetryTimeout(c), createUpgraderSA, slimCollector, istioVersion, cliEnvironment); err != nil {
				return errors.Wrap(err, "error downloading sensor bundle")
			}
			return nil
		},
	}

	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")
	c.PersistentFlags().BoolVar(&createUpgraderSA, "create-upgrader-sa", true, "whether to create the upgrader service account, with cluster-admin privileges, to facilitate automated sensor upgrades")
	c.PersistentFlags().StringVar(&istioVersion, "istio-support", "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))

	flags.AddTimeoutWithDefault(c, 5*time.Minute)
	flags.AddRetryTimeoutWithDefault(c, time.Duration(0))

	autobool.NewFlag(c.PersistentFlags(), &slimCollector, "slim-collector", "Use slim collector in deployment bundle")

	return c
}
