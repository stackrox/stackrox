package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/central"
)

func orchestratorCommand(shortName, longName string, clusterType v1.ClusterType) *cobra.Command {
	c := &cobra.Command{
		Use:   shortName,
		Short: fmt.Sprintf("%s specifies that you are going to launch StackRox Prevent Central in %s.", shortName, longName),
		Long: fmt.Sprintf(`%s specifies that you are going to launch StackRox Prevent Central in %s.
Output is a zip file printed to stdout.`, shortName, longName),
		SilenceErrors: true,
		Annotations: map[string]string{
			"category": "Enter orchestrator",
		},
	}
	return c
}

func k8sBasedOrchestrator(k8sConfig *central.K8sConfig, shortName, longName string, cluster v1.ClusterType) *cobra.Command {
	c := orchestratorCommand(shortName, longName, cluster)
	c.PersistentPreRun = func(*cobra.Command, []string) {
		cfg.K8sConfig = k8sConfig
		cfg.ClusterType = cluster
	}
	c.RunE = func(*cobra.Command, []string) error {
		if err := validateConfig(cfg); err != nil {
			return err
		}
		return outputZip(cfg)
	}

	c.AddCommand(externalVolume(cluster))
	c.AddCommand(hostPathVolume(cluster))

	// Adds k8s specific flags
	c.PersistentFlags().StringVarP(&k8sConfig.Namespace, "namespace", "n", "stackrox", "namespace")
	return c
}

func k8s() *cobra.Command {
	k8sConfig := new(central.K8sConfig)
	c := k8sBasedOrchestrator(k8sConfig, "k8s", "Kubernetes", v1.ClusterType_KUBERNETES_CLUSTER)
	c.PersistentFlags().StringVarP(&k8sConfig.ClairifyImage, "clairify-image", "", "stackrox.io/"+clairifyImage, "Clairify image to use")
	c.PersistentFlags().StringVarP(&k8sConfig.PreventImage, "prevent-image", "i", "stackrox.io/"+preventImage, "Prevent image to use")
	c.PersistentFlags().StringVarP(&k8sConfig.ImagePullSecret, "image-pull-secret", "", "stackrox", "image pull secret")
	return c
}

func openshift() *cobra.Command {
	k8sConfig := new(central.K8sConfig)
	c := k8sBasedOrchestrator(k8sConfig, "openshift", "Openshift", v1.ClusterType_OPENSHIFT_CLUSTER)
	c.PersistentFlags().StringVarP(&k8sConfig.ClairifyImage, "clairify-image", "", "docker-registry.default.svc:5000/stackrox/"+clairifyImage, "Clairify image to use")
	c.PersistentFlags().StringVarP(&k8sConfig.PreventImage, "prevent-image", "i", "docker-registry.default.svc:5000/stackrox/"+preventImage, "Prevent image to use")
	return c
}
