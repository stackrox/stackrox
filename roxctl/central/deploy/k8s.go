package deploy

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/roxctl"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
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
			return errors.New("storage type must be specified")
		}),
	}
	if !roxctl.InMainImage() {
		c.PersistentFlags().Var(newOutputDir(&cfg.OutputDir), "output-dir", "the directory to output the deployment bundle to")
	}
	return c
}

func k8sBasedOrchestrator(k8sConfig *renderer.K8sConfig, shortName, longName string, cluster storage.ClusterType) *cobra.Command {
	c := orchestratorCommand(shortName, longName)
	c.PersistentPreRun = func(*cobra.Command, []string) {
		cfg.K8sConfig = k8sConfig
		cfg.ClusterType = cluster
	}

	c.AddCommand(externalVolume())
	c.AddCommand(hostPathVolume())
	c.AddCommand(noVolume())

	flagWrap := &persistentFlagsWrapper{FlagSet: c.PersistentFlags()}

	// Adds k8s specific flags
	flagWrap.StringVarP(&k8sConfig.MainImage, "main-image", "i", defaults.MainImage(), "main image to use", "central")
	flagWrap.BoolVar(&k8sConfig.OfflineMode, "offline", false, "whether to run StackRox in offline mode, which avoids reaching out to the Internet", "central")

	flagWrap.StringVar(&k8sConfig.ScannerImage, "scanner-image", defaults.ScannerImage(), "Scanner image to use", "scanner")
	flagWrap.StringVar(&k8sConfig.ScannerDBImage, "scanner-db-image", defaults.ScannerDBImage(), "Scanner DB image to use", "scanner")

	flagWrap.BoolVar(&k8sConfig.EnableTelemetry, "enable-telemetry", true, "whether to enable telemetry", "central")

	return c
}

func newK8sConfig() *renderer.K8sConfig {
	return &renderer.K8sConfig{}
}

func k8s() *cobra.Command {
	k8sConfig := newK8sConfig()
	c := k8sBasedOrchestrator(k8sConfig, "k8s", "Kubernetes", storage.ClusterType_KUBERNETES_CLUSTER)
	flagWrap := &persistentFlagsWrapper{FlagSet: c.PersistentFlags()}

	flagWrap.Var(&loadBalancerWrapper{LoadBalancerType: &k8sConfig.LoadBalancerType}, "lb-type", "the method of exposing Central (lb, np, none)", "central")

	validFormats := []string{"kubectl", "helm"}
	if features.CentralInstallationExperience.Enabled() {
		validFormats = append(validFormats, "helm-values")
	}
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

func openshift() *cobra.Command {
	k8sConfig := newK8sConfig()
	c := k8sBasedOrchestrator(k8sConfig, "openshift", "Openshift", storage.ClusterType_OPENSHIFT_CLUSTER)

	flagWrap := &persistentFlagsWrapper{FlagSet: c.PersistentFlags()}

	flagWrap.Var(&loadBalancerWrapper{LoadBalancerType: &k8sConfig.LoadBalancerType}, "lb-type", "the method of exposing Central (route, lb, np, none)", "central")

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
