package generate

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/roxctl"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	noteOpenShift3xCompatibilityMode = `NOTE: Deployment files are generated in OpenShift 3.x compatibility mode. Set the --openshift-version flag to 3 to suppress this note, or to 4 take advantage of OpenShift 4.x features.`
)

type persistentFlagsWrapper struct {
	*pflag.FlagSet
}

func (w *persistentFlagsWrapper) UInt32Var(p *uint32, name string, value uint32, usage string, groups ...string) {
	w.FlagSet.Uint32Var(p, name, value, usage)
	utils.Must(w.SetAnnotation(name, groupAnnotationKey, groups))
}

func (w *persistentFlagsWrapper) StringVar(p *string, name, value, usage string, groups ...string) {
	w.StringVarP(p, name, "", value, usage, groups...)
}

func (w *persistentFlagsWrapper) StringVarP(p *string, name, shorthand, value, usage string, groups ...string) {
	w.FlagSet.StringVarP(p, name, shorthand, value, usage)
	utils.Must(w.SetAnnotation(name, groupAnnotationKey, groups))
}

func (w *persistentFlagsWrapper) BoolVar(p *bool, name string, value bool, usage string, groups ...string) {
	w.FlagSet.BoolVar(p, name, value, usage)
	utils.Must(w.SetAnnotation(name, groupAnnotationKey, groups))
}

func (w *persistentFlagsWrapper) Var(value pflag.Value, name, usage string, groups ...string) {
	w.FlagSet.Var(value, name, usage)
	utils.Must(w.SetAnnotation(name, groupAnnotationKey, groups))
}

func orchestratorCommand(shortName, longName string) *cobra.Command {
	c := &cobra.Command{
		Use: shortName,
		Annotations: map[string]string{
			categoryAnnotation: "Enter orchestrator",
		},
		RunE: util.RunENoArgs(func(*cobra.Command) error {
			return errox.InvalidArgs.New("storage type must be specified")
		}),
	}
	if !roxctl.InMainImage() {
		c.PersistentFlags().Var(newOutputDir(&cfg.OutputDir), "output-dir", "the directory to output the deployment bundle to")
	}
	return c
}

func k8sBasedOrchestrator(cliEnvironment environment.Environment, k8sConfig *renderer.K8sConfig, shortName, longName string, getClusterType func() (storage.ClusterType, error)) *cobra.Command {
	c := orchestratorCommand(shortName, longName)
	c.PersistentPreRunE = func(*cobra.Command, []string) error {
		clusterType, err := getClusterType()
		if err != nil {
			return errors.Wrap(err, "determining cluster type")
		}
		cfg.K8sConfig = k8sConfig
		cfg.ClusterType = clusterType
		return nil
	}

	c.AddCommand(externalVolume(cliEnvironment))
	c.AddCommand(hostPathVolume(cliEnvironment))
	c.AddCommand(noVolume(cliEnvironment))

	flagWrap := &persistentFlagsWrapper{FlagSet: c.PersistentFlags()}
	// Adds k8s specific flags
	flags.AddImageDefaults(flagWrap.FlagSet, &k8sConfig.ImageFlavorName)

	defaultImageHelp := fmt.Sprintf("(if unset, a default will be used according to --%s)", flags.ImageDefaultsFlagName)
	flagWrap.StringVarP(&k8sConfig.MainImage, flags.FlagNameMainImage, "i", "", "main image to use"+defaultImageHelp, "central")
	flagWrap.BoolVar(&k8sConfig.OfflineMode, "offline", false, "whether to run StackRox in offline mode, which avoids reaching out to the Internet", "central")
	//if features.PostgresDatastore.Enabled() {
	flagWrap.StringVar(&k8sConfig.CentralDBImage, flags.FlagNameCentralDBImage, "", "central-db image to use"+defaultImageHelp, "central")
	//}
	flagWrap.StringVar(&k8sConfig.ScannerImage, flags.FlagNameScannerImage, "", "scanner image to use"+defaultImageHelp, "scanner")
	flagWrap.StringVar(&k8sConfig.ScannerDBImage, flags.FlagNameScannerDBImage, "", "scanner-db image to use"+defaultImageHelp, "scanner")

	flagWrap.BoolVar(&k8sConfig.EnableTelemetry, "enable-telemetry", true, "whether to enable telemetry", "central")

	k8sConfig.EnableCentralDB = true //features.PostgresDatastore.Enabled()

	return c
}

func newK8sConfig() *renderer.K8sConfig {
	return &renderer.K8sConfig{}
}

func k8s(cliEnvironment environment.Environment) *cobra.Command {
	k8sConfig := newK8sConfig()
	c := k8sBasedOrchestrator(cliEnvironment, k8sConfig, "k8s", "Kubernetes", func() (storage.ClusterType, error) { return storage.ClusterType_KUBERNETES_CLUSTER, nil })
	flagWrap := &persistentFlagsWrapper{FlagSet: c.PersistentFlags()}

	flagWrap.Var(&loadBalancerWrapper{LoadBalancerType: &k8sConfig.LoadBalancerType}, "lb-type", "the method of exposing Central (lb, np, none)", "central")

	validFormats := []string{"kubectl", "helm", "helm-values"}
	flagWrap.Var(&fileFormatWrapper{DeploymentFormat: &k8sConfig.DeploymentFormat}, "output-format", fmt.Sprintf("the deployment tool to use (%s)", strings.Join(validFormats, ", ")), "central")

	flagWrap.Var(istioSupportWrapper{&k8sConfig.IstioVersion}, "istio-support",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version (kubectl output format only). Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")),
		"central", "output-format=kubectl",
	)
	utils.Must(
		flagWrap.SetAnnotation("istio-support", flags.OptionalKey, []string{"true"}),
		flagWrap.SetAnnotation("istio-support", flags.InteractiveUsageKey, []string{"Istio version when deploying into an Istio-enabled cluster (leave empty when not running Istio)"}),
	)

	return c
}

func openshift(cliEnvironment environment.Environment) *cobra.Command {
	k8sConfig := newK8sConfig()

	var openshiftVersion int
	c := k8sBasedOrchestrator(cliEnvironment, k8sConfig, "openshift", "Openshift", func() (storage.ClusterType, error) {
		clusterType := storage.ClusterType_OPENSHIFT_CLUSTER
		switch openshiftVersion {
		case 0:
			cliEnvironment.Logger().InfofLn("%s", noteOpenShift3xCompatibilityMode)
		case 3:
		case 4:
			clusterType = storage.ClusterType_OPENSHIFT4_CLUSTER
		default:
			return 0, errors.Errorf("invalid OpenShift version %d, supported values are '3' and '4'", openshiftVersion)
		}
		return clusterType, nil
	})

	flagWrap := &persistentFlagsWrapper{FlagSet: c.PersistentFlags()}

	flagWrap.Var(&loadBalancerWrapper{LoadBalancerType: &k8sConfig.LoadBalancerType}, "lb-type", "the method of exposing Central (route, lb, np, none)", "central")

	validFormats := []string{"kubectl", "helm", "helm-values"}
	flagWrap.Var(&fileFormatWrapper{DeploymentFormat: &k8sConfig.DeploymentFormat}, "output-format", fmt.Sprintf("the deployment tool to use (%s)", strings.Join(validFormats, ", ")), "central")

	flagWrap.IntVar(&openshiftVersion, "openshift-version", 0, "the OpenShift major version (3 or 4) to deploy on")
	flagWrap.Var(istioSupportWrapper{&k8sConfig.IstioVersion}, "istio-support",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")),
		"central",
	)
	utils.Must(
		flagWrap.SetAnnotation("istio-support", flags.OptionalKey, []string{"true"}),
		flagWrap.SetAnnotation("istio-support", flags.InteractiveUsageKey, []string{"Istio version when deploying into an Istio-enabled cluster (leave empty when not running Istio)"}),
	)

	return c
}
