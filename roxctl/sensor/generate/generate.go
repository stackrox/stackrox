package generate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/pflag/autobool"
	"github.com/stackrox/rox/roxctl/sensor/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/pointer"
)

const (
	infoDefaultingToSlimCollector          = `Defaulting to slim collector image since kernel probes seem to be available for central.`
	infoDefaultingToComprehensiveCollector = `Defaulting to comprehensive collector image since kernel probes seem to be unavailable for central.`

	warningDeprecatedAdmControllerCreateSet = `The --create-admission-controller flag has been deprecated and will be removed in future versions of roxctl.
Please use --admission-controller-listen-on-creates instead to suppress this warning text and avoid breakages in the future.`

	errorDeprecatedAdmControllerCreateSet = `It is illegal to specify both the --create-admission-controller and --admission-controller-listen-on-creates flags.
Please use --admission-controller-listen-on-creates exclusively in all invocations.`

	warningDeprecatedAdmControllerEnableSet = `The --admission-controller-enabled flag has been deprecated and will be removed in future versions of roxctl.
Please use --admission-controller-enforce-on-creates instead to suppress this warning text and avoid breakages in the future.`

	errorDeprecatedAdmControllerEnableSet = `It is illegal to specify both the --admission-controller-enabled and --admission-controller-enforce-on-creates flags.
Please use --admission-controller-enforce-on-creates exclusively in all invocations.`
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

	logger environment.Logger
)

func isLegacyValidationError(err error) bool {
	return err != nil &&
		status.Code(err) == codes.Internal &&
		cluster.MainImage == "" &&
		status.Convert(err).Message() == "Cluster Validation error: invalid main image '': invalid reference format"
}

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
	} else {
		cluster.SlimCollector = env.KernelSupportAvailable
	}

	id, err := createCluster(ctx, service)

	// Backward compatibility: if the central hasn't accepted the provided cluster
	// then fill default values as RHACS.
	if isLegacyValidationError(err) {
		var flavor defaults.ImageFlavor
		if buildinfo.ReleaseBuild {
			flavor = defaults.RHACSReleaseImageFlavor()
		} else {
			flavor = defaults.DevelopmentBuildImageFlavor()
		}

		logger.WarnfLn("Running older version of central. Can't rely on central configuration to determine default values. Using %s as main registry.",
			flavor.MainRegistry)

		cluster.MainImage = flavor.MainImageNoTag()
		id, err = createCluster(ctx, service)
	}

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

	if slimCollectorP != nil {
		if cluster.SlimCollector && !env.KernelSupportAvailable {
			logger.WarnfLn(util.WarningSlimCollectorModeWithoutKernelSupport)
		}
	} else if cluster.GetSlimCollector() {
		logger.InfofLn(infoDefaultingToSlimCollector)
	} else {
		logger.InfofLn(infoDefaultingToComprehensiveCollector)
	}

	if env.Error != nil {
		logger.WarnfLn("Sensor bundle has been created successfully, but it was not possible to retrieve Central's runtime environment information: %v",
			env.Error)
	}
	return nil
}

// Command defines the sensor generate command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	logger = cliEnvironment.Logger()
	c := &cobra.Command{
		Use: "generate",
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			// Migration process for renaming "--create-admission-controller" parameter to "--admission-controller-listen-on-creates".
			// Can be removed in a future release.
			if c.PersistentFlags().Lookup("create-admission-controller").Changed && c.PersistentFlags().Lookup("admission-controller-listen-on-creates").Changed {
				logger.ErrfLn(errorDeprecatedAdmControllerCreateSet)
				return errors.New("Specified deprecated flag --create-admission-controller and new flag --admission-controller-listen-on-creates at the same time")
			}
			if c.PersistentFlags().Lookup("create-admission-controller").Changed {
				logger.WarnfLn(warningDeprecatedAdmControllerCreateSet)
			}

			// Migration process for renaming "--admission-controller-enabled" parameter to "--admission-controller-enforce-on-creates".
			// Can be removed in a future release.
			if c.PersistentFlags().Lookup("admission-controller-enabled").Changed && c.PersistentFlags().Lookup("admission-controller-enforce-on-creates").Changed {
				logger.ErrfLn(errorDeprecatedAdmControllerEnableSet)
				return errors.New("Specified deprecated flag --admission-controller-enabled and new flag --admission-controller-enforce-on-creates at the same time")
			}
			if c.PersistentFlags().Lookup("admission-controller-enabled").Changed {
				logger.WarnfLn(warningDeprecatedAdmControllerEnableSet)
			}
			return nil
		},
	}

	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")
	c.PersistentFlags().BoolVar(&continueIfExists, "continue-if-exists", false, "continue with downloading the sensor bundle even if the cluster already exists")
	c.PersistentFlags().StringVar(&cluster.Name, "name", "", "cluster name to identify the cluster")
	c.PersistentFlags().StringVar(&cluster.CentralApiEndpoint, "central", "central.stackrox:443", "endpoint that sensor should connect to")
	c.PersistentFlags().StringVar(&cluster.MainImage, "main-image-repository", "", "image repository sensor should be deployed with (if unset, a default will be used)")
	c.PersistentFlags().StringVar(&cluster.CollectorImage, "collector-image-repository", "", "image repository collector should be deployed with (if unset, a default will be derived according to the effective --main-image-repository value)")

	c.PersistentFlags().Var(&collectionTypeWrapper{CollectionMethod: &cluster.CollectionMethod}, "collection-method", "which collection method to use for runtime support (none, default, kernel-module, ebpf)")

	c.PersistentFlags().BoolVar(&createUpgraderSA, "create-upgrader-sa", true, "whether to create the upgrader service account, with cluster-admin privileges, to facilitate automated sensor upgrades")

	c.PersistentFlags().StringVar(&istioVersion, "istio-support", "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))

	c.PersistentFlags().BoolVar(&cluster.GetTolerationsConfig().Disabled, "disable-tolerations", false, "Disable tolerations for tainted nodes")

	autobool.NewFlag(c.PersistentFlags(), &slimCollectorP, "slim-collector", "Use slim collector in deployment bundle")

	c.PersistentFlags().BoolVar(&cluster.AdmissionController, "create-admission-controller", false, "whether or not to use an admission controller for enforcement (WARNING: deprecated; admission controller will be deployed by default")
	utils.Must(c.PersistentFlags().MarkHidden("create-admission-controller"))

	c.PersistentFlags().BoolVar(&cluster.AdmissionController, "admission-controller-listen-on-creates", false, "whether or not to configure the admission controller webhook to listen on deployment creates")
	c.PersistentFlags().BoolVar(&cluster.AdmissionControllerUpdates, "admission-controller-listen-on-updates", false, "whether or not to configure the admission controller webhook to listen on deployment updates")

	// Admission controller config
	ac := cluster.DynamicConfig.AdmissionControllerConfig
	c.PersistentFlags().BoolVar(&ac.Enabled, "admission-controller-enabled", false, "dynamic enable for the admission controller (WARNING: deprecated; use --admission-controller-enforce-on-creates instead")
	utils.Must(c.PersistentFlags().MarkHidden("admission-controller-enabled"))

	c.PersistentFlags().Int32Var(&ac.TimeoutSeconds, "admission-controller-timeout", 3, "timeout in seconds for the admission controller")
	c.PersistentFlags().BoolVar(&ac.ScanInline, "admission-controller-scan-inline", false, "get scans inline when using the admission controller")
	c.PersistentFlags().BoolVar(&ac.DisableBypass, "admission-controller-disable-bypass", false, "disable the bypass annotations for the admission controller")
	c.PersistentFlags().BoolVar(&ac.Enabled, "admission-controller-enforce-on-creates", false, "dynamic enable for enforcing on object creates in the admission controller")
	c.PersistentFlags().BoolVar(&ac.EnforceOnUpdates, "admission-controller-enforce-on-updates", false, "dynamic enable for enforcing on object updates in the admission controller")

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
