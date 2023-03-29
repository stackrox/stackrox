package generate

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/roxctl"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	defaultCentralDBBundle = "central-db-bundle"
)

func orchestratorCommand(shortName, longName string) *cobra.Command {
	c := &cobra.Command{
		Use:   shortName,
		Short: shortName,
		Long:  longName,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errox.InvalidArgs.New("storage type must be specified")
			}
			return errox.InvalidArgs.Newf("unexpected storage type %q", args[0])
		},
	}
	if !roxctl.InMainImage() {
		c.PersistentFlags().Var(common.NewOutputDir(&cfg.OutputDir, defaultCentralDBBundle), "output-dir", "the directory to output the deployment bundle to")
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

	// Adds k8s specific flags
	flags.AddImageDefaults(c.PersistentFlags(), &k8sConfig.ImageFlavorName)

	defaultImageHelp := fmt.Sprintf(" (if unset, a default will be used according to --%s)", flags.ImageDefaultsFlagName)
	c.PersistentFlags().StringVarP(&k8sConfig.CentralDBImage, flags.FlagNameCentralDBImage, "", "", "central-db image to use"+defaultImageHelp)
	k8sConfig.EnableCentralDB = true

	return c
}

func newK8sConfig() *renderer.K8sConfig {
	return &renderer.K8sConfig{}
}

func k8s(cliEnvironment environment.Environment) *cobra.Command {
	k8sConfig := newK8sConfig()
	return k8sBasedOrchestrator(cliEnvironment, k8sConfig, "k8s", "Kubernetes", func() (storage.ClusterType, error) { return storage.ClusterType_KUBERNETES_CLUSTER, nil })
}

func openshift(cliEnvironment environment.Environment) *cobra.Command {
	k8sConfig := newK8sConfig()

	var openshiftVersion int
	c := k8sBasedOrchestrator(cliEnvironment, k8sConfig, "openshift", "Openshift", func() (storage.ClusterType, error) {
		switch openshiftVersion {
		case 3:
			return storage.ClusterType_OPENSHIFT_CLUSTER, nil
		case 4:
			return storage.ClusterType_OPENSHIFT4_CLUSTER, nil
		default:
			return 0, errors.Errorf("invalid OpenShift version %d, supported values are '3' and '4'", openshiftVersion)
		}
	})

	c.PersistentFlags().IntVar(&openshiftVersion, "openshift-version", 3, "the OpenShift major version (3 or 4) to deploy on")
	return c
}
