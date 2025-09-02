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
	"github.com/stackrox/rox/roxctl/sensor/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	mainImageRepository = "main-image-repository"

	warningCentralEnvironmentError = "It was not possible to retrieve Central's runtime environment information: %v. Will use fallback defaults for " + mainImageRepository + " setting."

	warningAdmissionControllerListenOnCreatesSet  = `The --admission-controller-listen-on-creates will be removed in future versions of roxctl. It will be ignored from version 4.9 onwards.`
	warningAdmissionControllerListenOnUpdatesSet  = `The --admission-controller-listen-on-updates will be removed in future versions of roxctl. It will be ignored from version 4.9 onwards.`
	warningAdmissionControllerScanInlineSet       = `The --admission-controller-scan-inline will be removed in future versions of roxctl. It will be ignored from version 4.9 onwards.`
	warningAdmissionControllerEnforceOnCreatesSet = `The --admission-controller-enforce-on-creates flag will be removed in future versions of roxctl. It will be ignored from version 4.9 onwards.`
	warningAdmissionControllerEnforceOnUpdatesSet = `The --admission-controller-enforce-on-updates flag will be removed in future versions of roxctl. It will be ignored from version 4.9 onwards.`
	warningAdmissionControllerTimeoutSet          = `The --admission-controller-timeout flag will be removed in future versions of roxctl. It will be ignored from version 4.9 onwards.`
)

type sensorGenerateCommand struct {
	// properties bound to cobra flags
	continueIfExists bool
	createUpgraderSA bool
	istioVersion     string
	outputDir        string
	timeout          time.Duration

	enablePodSecurityPolicies bool

	// injected or constructed values
	cluster     *storage.Cluster
	env         environment.Environment
	getBundleFn util.GetBundleFn
}

func defaultCluster() *storage.Cluster {
	return &storage.Cluster{
		AdmissionController:            true,
		AdmissionControllerEvents:      true,
		AdmissionControllerUpdates:     true,
		AdmissionControllerFailOnError: false,
		TolerationsConfig: &storage.TolerationsConfig{
			Disabled: false,
		},
		DynamicConfig: &storage.DynamicClusterConfig{
			AdmissionControllerConfig: &storage.AdmissionControllerConfig{
				Enabled:          true,
				ScanInline:       true,
				DisableBypass:    false,
				EnforceOnUpdates: true,
				TimeoutSeconds:   0,
			},
		},
	}
}

func (s *sensorGenerateCommand) Construct(cmd *cobra.Command) error {
	s.timeout = flags.Timeout(cmd)
	s.getBundleFn = util.GetBundle
	return nil
}

func (s *sensorGenerateCommand) setClusterDefaults(envDefaults *util.CentralEnv) {
	if s.cluster.MainImage == "" {
		// If no override was provided, use a possible default value from `envDefaults`. If this is a legacy central,
		// envDefaults.MainImage will hold a local default (from Release Flag). If env.Defaults.MainImage is empty it
		// means that roxctl is talking to newer version of Central which will accept empty MainImage values.
		s.cluster.MainImage = envDefaults.MainImage
	}
	s.cluster.AdmissionController = true
	s.cluster.AdmissionControllerUpdates = true
	s.cluster.AdmissionControllerEvents = true

	acc := s.cluster.GetDynamicConfig().GetAdmissionControllerConfig()
	if acc != nil {
		// This ensures the new --admission-controller-enforcement flag value "wins". The Enabled value is
		// used in admission controller business logic as "enforce on creates". The line below ensures we have
		// enforcement "on" for both operations, or "off" for both, in line with the new design based on
		// customer expectations.
		acc.Enabled = acc.EnforceOnUpdates
		acc.ScanInline = true

		// We set the timeout to 0 so that the Helm rendering takes care of setting the default value for timeout
		acc.TimeoutSeconds = 0
	}
}

func (s *sensorGenerateCommand) fullClusterCreation() error {
	conn, err := s.env.GRPCConnection()
	if err != nil {
		return errors.Wrap(err, "failed to create GRPC connection")
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
		IstioVersion:     s.istioVersion,

		DisablePodSecurityPolicies: !s.enablePodSecurityPolicies,
	}
	if err := s.getBundleFn(params, s.outputDir, s.timeout, s.env); err != nil {
		return errors.Wrap(err, "error getting cluster zip file")
	}

	return nil
}

func (s *sensorGenerateCommand) createCluster(ctx context.Context, svc v1.ClustersServiceClient) (string, error) {
	response, err := svc.PostCluster(ctx, s.cluster)
	if err != nil {
		return "", errors.Wrap(err, "error creating cluster")
	}
	return response.GetCluster().GetId(), nil
}

// Command defines the sensor generate command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	generateCmd := &sensorGenerateCommand{env: cliEnvironment, cluster: defaultCluster()}
	c := &cobra.Command{
		Use:   "generate",
		Short: "Commands that generate files to deploy StackRox services into secured clusters",
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			return generateCmd.Construct(c)
		},
	}

	c.PersistentFlags().StringVar(&generateCmd.outputDir, "output-dir", "", "Output directory for bundle contents (default: auto-generated directory name inside the current directory).")
	c.PersistentFlags().BoolVar(&generateCmd.continueIfExists, "continue-if-exists", false, "Continue with downloading the sensor bundle even if the cluster already exists.")
	c.PersistentFlags().StringVar(&generateCmd.cluster.Name, "name", "", "Cluster name to identify the cluster.")
	c.PersistentFlags().StringVar(&generateCmd.cluster.CentralApiEndpoint, "central", "central.stackrox:443", "Endpoint that sensor should connect to.")
	c.PersistentFlags().StringVar(&generateCmd.cluster.MainImage, mainImageRepository, "", "Image repository sensor should be deployed with (if unset, a default will be used).")
	c.PersistentFlags().StringVar(&generateCmd.cluster.CollectorImage, "collector-image-repository", "", "Image repository collector should be deployed with (if unset, a default will be derived according to the effective --"+mainImageRepository+" value).")

	c.PersistentFlags().Var(&collectionTypeWrapper{CollectionMethod: &generateCmd.cluster.CollectionMethod}, "collection-method", "Which collection method to use for runtime support (none, default, core_bpf).")

	c.PersistentFlags().BoolVar(&generateCmd.createUpgraderSA, "create-upgrader-sa", true, "Whether to create the upgrader service account, with cluster-admin privileges, to facilitate automated sensor upgrades.")

	c.PersistentFlags().StringVar(&generateCmd.istioVersion, "istio-support", "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s.",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))

	c.PersistentFlags().BoolVar(&generateCmd.cluster.GetTolerationsConfig().Disabled, "disable-tolerations", false, "Disable tolerations for tainted nodes.")
	c.PersistentFlags().BoolVar(&generateCmd.enablePodSecurityPolicies, "enable-pod-security-policies", false, "Create PodSecurityPolicy resources (for pre-v1.25 Kubernetes).")

	// Note: If you need to change the default values for any of the flags below this comment, please change the defaults in the defaultCluster function above
	c.PersistentFlags().BoolVar(&generateCmd.cluster.AdmissionController, "admission-controller-listen-on-creates", true, "Whether or not to configure the admission controller webhook to listen on deployment creates.")
	utils.Must(c.PersistentFlags().MarkDeprecated("admission-controller-listen-on-creates", warningAdmissionControllerListenOnCreatesSet))
	c.PersistentFlags().BoolVar(&generateCmd.cluster.AdmissionControllerUpdates, "admission-controller-listen-on-updates", true, "Whether or not to configure the admission controller webhook to listen on deployment updates.")
	utils.Must(c.PersistentFlags().MarkDeprecated("admission-controller-listen-on-updates", warningAdmissionControllerListenOnUpdatesSet))

	// Admission controller config
	ac := generateCmd.cluster.DynamicConfig.AdmissionControllerConfig

	c.PersistentFlags().Int32Var(&ac.TimeoutSeconds, "admission-controller-timeout", 0, "Timeout in seconds for the admission controller.")
	utils.Must(c.PersistentFlags().MarkDeprecated("admission-controller-timeout", warningAdmissionControllerTimeoutSet))

	c.PersistentFlags().BoolVar(&ac.ScanInline, "admission-controller-scan-inline", true, "Get scans inline when using the admission controller.")
	utils.Must(c.PersistentFlags().MarkDeprecated("admission-controller-scan-inline", warningAdmissionControllerScanInlineSet))

	c.PersistentFlags().BoolVar(&ac.DisableBypass, "admission-controller-disable-bypass", false, "Disable the bypass annotations for the admission controller.")

	c.PersistentFlags().BoolVar(&ac.Enabled, "admission-controller-enforce-on-creates", true, "Dynamic enable for enforcing on object creates in the admission controller.")
	utils.Must(c.PersistentFlags().MarkDeprecated("admission-controller-enforce-on-creates", warningAdmissionControllerEnforceOnCreatesSet))

	c.PersistentFlags().BoolVar(&ac.EnforceOnUpdates, "admission-controller-enforce-on-updates", true, "Dynamic enable for enforcing on object updates in the admission controller.")
	utils.Must(c.PersistentFlags().MarkDeprecated("admission-controller-enforce-on-updates", warningAdmissionControllerEnforceOnUpdatesSet))

	c.PersistentFlags().BoolVar(&generateCmd.cluster.AdmissionControllerFailOnError, "admission-controller-fail-on-error", false, "Fail the admission review request in case of errors or timeouts in request evaluation.")
	c.PersistentFlags().BoolVar(&ac.EnforceOnUpdates, "admission-controller-enforcement", true, "Enforce security policies on the admission review request.")

	c.MarkFlagsMutuallyExclusive("admission-controller-enforce-on-creates", "admission-controller-enforcement")
	c.MarkFlagsMutuallyExclusive("admission-controller-enforce-on-updates", "admission-controller-enforcement")

	flags.AddTimeoutWithDefault(c, 5*time.Minute)

	c.AddCommand(k8s(generateCmd))
	c.AddCommand(openshift(generateCmd))

	return c
}
