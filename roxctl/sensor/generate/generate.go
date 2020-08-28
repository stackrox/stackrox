package generate

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/pflag/autobool"
	"github.com/stackrox/rox/roxctl/sensor/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/pointer"
)

const (
	warningDeprecatedImageFlagSet = `WARNING: The --image flag has been renamed to --main-image-repository and will be removed in future versions of roxctl.
Please use --main-image-repository to suppress this warning text and avoid breakages in the future.`

	errorDeprecatedAndNewImageFlagSet = `It is illegal to specify both the --image and --main-image-repository flags.
Please use --main-image-repository exclusively in all invocations.`

	warningDeprecatedCollectorImageFlagSet = `WARNING: The --collector-image flag has been renamed to --collector-image-repository and will be removed in future versions of roxctl.
Please use --collector-image-repository to suppress this warning text and avoid breakages in the future.`
	errorDeprecatedAndNewCollectorImageFlagSet = `It is illegal to specify both the --collector-image and --collector-image-repository flags.
Please use --collector-image-repository exclusively in all invocations.`

	warningDeprecatedFlagMonitoringEndpointUsed = `WARNING: The flag --monitoring-endpoint has been deprecated and has no impact. It will be removed in a future version of roxctl.
Remove it from your invocations of roxctl to avoid future breakages.`

	infoDefaultingToSlimCollector          = `Defaulting to slim collector image since kernel probes seem to be available for central.`
	infoDefaultingToComprehensiveCollector = `Defaulting to comprehensive collector image since kernel probes seem to be unavailable for central.`
)

var (
	cluster = storage.Cluster{
		TolerationsConfig: &storage.TolerationsConfig{
			Disabled: false,
		},
		DynamicConfig: &storage.DynamicClusterConfig{
			AdmissionControllerConfig: &storage.AdmissionControllerConfig{},
		},
	}
	continueIfExists bool

	createUpgraderSA bool

	istioVersion string

	outputDir string

	slimCollectorP *bool
)

func fullClusterCreation(timeout time.Duration) error {
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	service := v1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	env := util.RetrieveCentralEnvOrDefault(ctx, service)
	// Here we only set the cluster property, which will be persisted by central.
	// This is not directly related to fetching the bundle.
	// It should only be used when the request to download a bundle does not contain a `slimCollector` setting.
	if slimCollectorP != nil {
		cluster.SlimCollector = *slimCollectorP
		if cluster.SlimCollector && !env.KernelSupportAvailable {
			fmt.Fprintf(os.Stderr, "%s\n\n", util.WarningSlimCollectorModeWithoutKernelSupport)
		}
	} else {
		cluster.SlimCollector = env.KernelSupportAvailable
		if cluster.GetSlimCollector() {
			fmt.Fprintln(os.Stderr, infoDefaultingToSlimCollector)
		} else {
			fmt.Fprintln(os.Stderr, infoDefaultingToComprehensiveCollector)
		}
	}

	id, err := createCluster(ctx, service)
	// If the error is not explicitly AlreadyExists or it is AlreadyExists AND continueIfExists isn't set
	// then return an error

	if err != nil {
		if status.Code(err) == codes.AlreadyExists && continueIfExists {
			// Need to get the clusters and get the one with the name
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			clusterResponse, err := service.GetClusters(ctx, &v1.GetClustersRequest{Query: search.NewQueryBuilder().AddExactMatches(search.Cluster, cluster.GetName()).Query()})
			if err != nil {
				return errors.Wrap(err, "error getting clusters")
			}
			for _, c := range clusterResponse.GetClusters() {
				if strings.EqualFold(c.GetName(), cluster.GetName()) {
					id = c.GetId()
				}
			}
			if id == "" {
				return fmt.Errorf("error finding preexisting cluster with name %q", cluster.GetName())
			}
		} else {
			return errors.Wrap(err, "error creating cluster")
		}
	}

	params := apiparams.ClusterZip{
		ID:               id,
		CreateUpgraderSA: &createUpgraderSA,
		SlimCollector:    pointer.BoolPtr(cluster.GetSlimCollector()),
		IstioVersion:     istioVersion,
	}
	if err := util.GetBundle(params, outputDir, timeout); err != nil {
		return errors.Wrap(err, "error getting cluster zip file")
	}
	return nil
}

// Command defines the sensor generate command tree
func Command() *cobra.Command {
	var monitoringEndpointNoop string

	c := &cobra.Command{
		Use: "generate",
		Run: func(c *cobra.Command, _ []string) {
			_ = c.Help()
		},
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			// Print deprecation warnings. Note that we print out an additional newline in order to seperate
			// the warning from the rest of the program output.

			// Can be removed in a future release.
			if c.Flags().Lookup("monitoring-endpoint").Changed {
				fmt.Fprintf(os.Stderr, "%s\n\n", warningDeprecatedFlagMonitoringEndpointUsed)
			}

			// Migration process for renaming "--image" parameter to "--main-image-repository".
			// Can be removed in a future release.
			if c.PersistentFlags().Lookup("image").Changed && c.PersistentFlags().Lookup("main-image-repository").Changed {
				fmt.Fprintln(os.Stderr, errorDeprecatedAndNewImageFlagSet)
				return errors.New("Specified deprecated flag --image and new flag --main-image-repository at the same time")
			}
			if c.PersistentFlags().Lookup("image").Changed {
				fmt.Fprintf(os.Stderr, "%s\n\n", warningDeprecatedImageFlagSet)
			}

			// Migration process for renaming "--collector-image" parameter to "--collector-image-repository".
			// Can be removed in a future release.
			if c.PersistentFlags().Lookup("collector-image").Changed && c.PersistentFlags().Lookup("collector-image-repository").Changed {
				fmt.Fprintln(os.Stderr, errorDeprecatedAndNewCollectorImageFlagSet)
				return errors.New("Specified deprecated flag --collector-image and new flag --collector-image-repository at the same time")
			}
			if c.PersistentFlags().Lookup("collector-image").Changed {
				fmt.Fprintf(os.Stderr, "%s\n\n", warningDeprecatedCollectorImageFlagSet)
			}

			return nil
		},
	}

	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")
	c.PersistentFlags().BoolVar(&continueIfExists, "continue-if-exists", false, "continue with downloading the sensor bundle even if the cluster already exists")
	c.PersistentFlags().StringVar(&cluster.Name, "name", "", "cluster name to identify the cluster")
	c.PersistentFlags().StringVar(&cluster.CentralApiEndpoint, "central", "central.stackrox:443", "endpoint that sensor should connect to")
	c.PersistentFlags().StringVar(&cluster.MainImage, "image", defaults.MainImageRepo(), "image repo sensor should be deployed with")
	utils.Must(c.PersistentFlags().MarkHidden("image"))
	c.PersistentFlags().StringVar(&cluster.MainImage, "main-image-repository", defaults.MainImageRepo(), "image repository sensor should be deployed with")
	c.PersistentFlags().StringVar(&cluster.CollectorImage, "collector-image", "", "image repo collector should be deployed with (leave blank to use default)")
	utils.Must(c.PersistentFlags().MarkHidden("collector-image"))
	c.PersistentFlags().StringVar(&cluster.CollectorImage, "collector-image-repository", "", "image repository collector should be deployed with (leave blank to use default)")

	c.PersistentFlags().StringVar(&monitoringEndpointNoop, "monitoring-endpoint", "", "endpoint for monitoring (DEPRECATED)")
	utils.Must(c.PersistentFlags().MarkHidden("monitoring-endpoint"))

	c.PersistentFlags().BoolVar(&cluster.RuntimeSupport, "runtime", true, "whether or not to have runtime support (DEPRECATED, use Collection Method instead)")

	c.PersistentFlags().Var(&collectionTypeWrapper{CollectionMethod: &cluster.CollectionMethod}, "collection-method", "which collection method to use for runtime support (none, default, kernel-module, ebpf)")

	c.PersistentFlags().BoolVar(&createUpgraderSA, "create-upgrader-sa", true, "whether to create the upgrader service account, with cluster-admin privileges, to facilitate automated sensor upgrades")

	c.PersistentFlags().StringVar(&istioVersion, "istio-support", "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))

	c.PersistentFlags().BoolVar(&cluster.GetTolerationsConfig().Disabled, "disable-tolerations", false, "Disable tolerations for tainted nodes")

	if features.SupportSlimCollectorMode.Enabled() {
		autobool.NewFlag(c.PersistentFlags(), &slimCollectorP, "slim-collector", "Use slim collector in deployment bundle")
	} else {
		slimCollectorP = pointers.Bool(false)
	}

	c.AddCommand(k8s())
	c.AddCommand(openshift())

	return c
}

func createCluster(ctx context.Context, svc v1.ClustersServiceClient) (string, error) {
	if !cluster.GetAdmissionController() && cluster.GetDynamicConfig().GetAdmissionControllerConfig() != nil {
		cluster.DynamicConfig.AdmissionControllerConfig = nil
	}

	// Call detection and return the returned alerts.
	response, err := svc.PostCluster(ctx, &cluster)
	if err != nil {
		return "", err
	}
	return response.GetCluster().GetId(), nil
}
