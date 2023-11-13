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
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
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

	warningDeprecatedAdmControllerEnableSet = `The --admission-controller-enabled flag has been deprecated and will be removed in future versions of roxctl.
Please use --admission-controller-enforce-on-creates instead to suppress this warning text and avoid breakages in the future.`

	mainImageRepository = "main-image-repository"
	slimCollector       = "slim-collector"

	warningCentralEnvironmentError = "It was not possible to retrieve Central's runtime environment information: %v. Will use fallback defaults for " + mainImageRepository + " and " + slimCollector + " settings."
)

type sensorGenerateCommand struct {
	// properties bound to cobra flags
	continueIfExists bool
	createUpgraderSA bool
	istioVersion     string
	outputDir        string
	slimCollectorP   *bool
	timeout          time.Duration

	enablePodSecurityPolicies bool

	// injected or constructed values
	cluster     *storage.Cluster
	env         environment.Environment
	getBundleFn util.GetBundleFn
}

func defaultCluster() *storage.Cluster {
	return &storage.Cluster{
		TolerationsConfig: &storage.TolerationsConfig{
			Disabled: false,
		},
		DynamicConfig: &storage.DynamicClusterConfig{
			AdmissionControllerConfig: &storage.AdmissionControllerConfig{},
		},
	}
}

func (s *sensorGenerateCommand) Construct(cmd *cobra.Command) error {
	s.timeout = flags.Timeout(cmd)
	// Migration process for renaming "--create-admission-controller" parameter to "--admission-controller-listen-on-creates".
	// Can be removed in a future release.
	if cmd.PersistentFlags().Lookup("create-admission-controller").Changed && cmd.PersistentFlags().Lookup("admission-controller-listen-on-creates").Changed {
		return common.ErrDeprecatedFlag("--create-admission-controller", "--admission-controller-listen-on-creates")
	}
	if cmd.PersistentFlags().Lookup("create-admission-controller").Changed {
		s.env.Logger().WarnfLn(warningDeprecatedAdmControllerCreateSet)
	}

	// Migration process for renaming "--admission-controller-enabled" parameter to "--admission-controller-enforce-on-creates".
	// Can be removed in a future release.
	if cmd.PersistentFlags().Lookup("admission-controller-enabled").Changed && cmd.PersistentFlags().Lookup("admission-controller-enforce-on-creates").Changed {
		return common.ErrDeprecatedFlag("--admission-controller-enabled", "--admission-controller-enforce-on-creates")
	}
	if cmd.PersistentFlags().Lookup("admission-controller-enabled").Changed {
		s.env.Logger().WarnfLn(warningDeprecatedAdmControllerEnableSet)
	}

	s.getBundleFn = util.GetBundle
	return nil
}

func (s *sensorGenerateCommand) setClusterDefaults(envDefaults *util.CentralEnv) {
	// Here we only set the cluster property, which will be persisted by central.
	// This is not directly related to fetching the bundle.
	// It should only be used when the request to download a bundle does not contain a `slimCollector` setting.
	if s.slimCollectorP != nil {
		s.cluster.SlimCollector = *s.slimCollectorP
	} else {
		s.cluster.SlimCollector = envDefaults.KernelSupportAvailable
	}

	if s.cluster.MainImage == "" {
		// If no override was provided, use a possible default value from `envDefaults`. If this is a legacy central,
		// envDefaults.MainImage will hold a local default (from Release Flag). If env.Defaults.MainImage is empty it
		// means that roxctl is talking to newer version of Central which will accept empty MainImage values.
		s.cluster.MainImage = envDefaults.MainImage
	}
}

func (s *sensorGenerateCommand) fullClusterCreation() error {
	conn, err := s.env.GRPCConnection()
	if err != nil {
		return err
	}
	service := v1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	env, err := util.RetrieveCentralEnvOrDefault(ctx, service)
	if err != nil {
		s.env.Logger().WarnfLn(warningCentralEnvironmentError, err)
	}
	for _, warning := range env.Warnings {
		s.env.Logger().WarnfLn(warning)
	}
	s.setClusterDefaults(env)

	common.LogInfoPsp(s.env.Logger(), s.enablePodSecurityPolicies)

	id, err := s.createCluster(ctx, service)
	// If the error is not explicitly AlreadyExists or it is AlreadyExists AND continueIfExists isn't set
	// then return an error
	if err != nil {
		if status.Code(err) == codes.AlreadyExists && s.continueIfExists {
			// Need to get the clusters and get the one with the name
			ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
			defer cancel()
			clusterResponse, err := service.GetClusters(ctx, &v1.GetClustersRequest{Query: search.NewQueryBuilder().AddExactMatches(search.Cluster, s.cluster.GetName()).Query()})
			if err != nil {
				return errors.Wrap(err, "error getting clusters")
			}
			for _, cluster := range clusterResponse.GetClusters() {
				if strings.EqualFold(cluster.GetName(), s.cluster.GetName()) {
					id = cluster.GetId()
				}
			}
			if id == "" {
				return errox.NotFound.Newf("error finding preexisting cluster with name %q", s.cluster.GetName())
			}
		} else {
			return errors.Wrap(err, "error creating cluster")
		}
	}

	params := apiparams.ClusterZip{
		ID:               id,
		CreateUpgraderSA: &s.createUpgraderSA,
		SlimCollector:    pointer.Bool(s.cluster.GetSlimCollector()),
		IstioVersion:     s.istioVersion,

		DisablePodSecurityPolicies: !s.enablePodSecurityPolicies,
	}
	if err := s.getBundleFn(params, s.outputDir, s.timeout, s.env); err != nil {
		return errors.Wrap(err, "error getting cluster zip file")
	}

	if s.slimCollectorP != nil {
		if s.cluster.SlimCollector && !env.KernelSupportAvailable {
			s.env.Logger().WarnfLn(util.WarningSlimCollectorModeWithoutKernelSupport)
		}
	} else if s.cluster.GetSlimCollector() {
		s.env.Logger().InfofLn(infoDefaultingToSlimCollector)
	} else {
		s.env.Logger().InfofLn(infoDefaultingToComprehensiveCollector)
	}

	return nil
}

func (s *sensorGenerateCommand) createCluster(ctx context.Context, svc v1.ClustersServiceClient) (string, error) {
	if !s.cluster.GetAdmissionController() && s.cluster.GetDynamicConfig().GetAdmissionControllerConfig() != nil {
		s.cluster.DynamicConfig.AdmissionControllerConfig = nil
	}

	response, err := svc.PostCluster(ctx, s.cluster)
	if err != nil {
		return "", err
	}
	return response.GetCluster().GetId(), nil
}

// Command defines the sensor generate command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	generateCmd := &sensorGenerateCommand{env: cliEnvironment, cluster: defaultCluster()}
	c := &cobra.Command{
		Use:   "generate",
		Short: "Commands that generate files to deploy StackRox services into secured clusters.",
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			return generateCmd.Construct(c)
		},
	}

	c.PersistentFlags().StringVar(&generateCmd.outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")
	c.PersistentFlags().BoolVar(&generateCmd.continueIfExists, "continue-if-exists", false, "continue with downloading the sensor bundle even if the cluster already exists")
	c.PersistentFlags().StringVar(&generateCmd.cluster.Name, "name", "", "cluster name to identify the cluster")
	c.PersistentFlags().StringVar(&generateCmd.cluster.CentralApiEndpoint, "central", "central.stackrox:443", "endpoint that sensor should connect to")
	c.PersistentFlags().StringVar(&generateCmd.cluster.MainImage, mainImageRepository, "", "image repository sensor should be deployed with (if unset, a default will be used)")
	c.PersistentFlags().StringVar(&generateCmd.cluster.CollectorImage, "collector-image-repository", "", "image repository collector should be deployed with (if unset, a default will be derived according to the effective --"+mainImageRepository+" value)")

	c.PersistentFlags().Var(&collectionTypeWrapper{CollectionMethod: &generateCmd.cluster.CollectionMethod}, "collection-method", "which collection method to use for runtime support (none, default, ebpf, core_bpf)")

	c.PersistentFlags().BoolVar(&generateCmd.createUpgraderSA, "create-upgrader-sa", true, "whether to create the upgrader service account, with cluster-admin privileges, to facilitate automated sensor upgrades")

	c.PersistentFlags().StringVar(&generateCmd.istioVersion, "istio-support", "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))

	c.PersistentFlags().BoolVar(&generateCmd.cluster.GetTolerationsConfig().Disabled, "disable-tolerations", false, "Disable tolerations for tainted nodes")

	autobool.NewFlag(c.PersistentFlags(), &generateCmd.slimCollectorP, slimCollector, "Use slim collector in deployment bundle")

	c.PersistentFlags().BoolVar(&generateCmd.cluster.AdmissionController, "create-admission-controller", false, "whether or not to use an admission controller for enforcement (WARNING: deprecated; admission controller will be deployed by default")
	utils.Must(c.PersistentFlags().MarkHidden("create-admission-controller"))

	c.PersistentFlags().BoolVar(&generateCmd.cluster.AdmissionController, "admission-controller-listen-on-creates", false, "whether or not to configure the admission controller webhook to listen on deployment creates")
	c.PersistentFlags().BoolVar(&generateCmd.cluster.AdmissionControllerUpdates, "admission-controller-listen-on-updates", false, "whether or not to configure the admission controller webhook to listen on deployment updates")
	c.PersistentFlags().BoolVar(&generateCmd.enablePodSecurityPolicies, "enable-pod-security-policies", true, "Create PodSecurityPolicy resources (for pre-v1.25 Kubernetes)")

	// Admission controller config
	ac := generateCmd.cluster.DynamicConfig.AdmissionControllerConfig
	c.PersistentFlags().BoolVar(&ac.Enabled, "admission-controller-enabled", false, "dynamic enable for the admission controller (WARNING: deprecated; use --admission-controller-enforce-on-creates instead")
	utils.Must(c.PersistentFlags().MarkHidden("admission-controller-enabled"))

	c.PersistentFlags().Int32Var(&ac.TimeoutSeconds, "admission-controller-timeout", 3, "timeout in seconds for the admission controller")
	c.PersistentFlags().BoolVar(&ac.ScanInline, "admission-controller-scan-inline", false, "get scans inline when using the admission controller")
	c.PersistentFlags().BoolVar(&ac.DisableBypass, "admission-controller-disable-bypass", false, "disable the bypass annotations for the admission controller")
	c.PersistentFlags().BoolVar(&ac.Enabled, "admission-controller-enforce-on-creates", false, "dynamic enable for enforcing on object creates in the admission controller")
	c.PersistentFlags().BoolVar(&ac.EnforceOnUpdates, "admission-controller-enforce-on-updates", false, "dynamic enable for enforcing on object updates in the admission controller")

	flags.AddTimeoutWithDefault(c, 5*time.Minute)

	c.AddCommand(k8s(generateCmd))
	c.AddCommand(openshift(generateCmd))

	return c
}
