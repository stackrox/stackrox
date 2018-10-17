package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/cmd/deploy/central"
	"github.com/stackrox/rox/generated/api/v1"
)

func orchestratorCommand(shortName, longName string) *cobra.Command {
	c := &cobra.Command{
		Use:   shortName,
		Short: fmt.Sprintf("%s specifies that you are going to launch StackRox Prevent Central in %s.", shortName, longName),
		Long: fmt.Sprintf(`%s specifies that you are going to launch StackRox Prevent Central in %s.
Output is a zip file printed to stdout.`, shortName, longName),
		SilenceErrors: true,
		Annotations: map[string]string{
			"category": "Enter orchestrator",
		},
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("storage type must be specified")
		},
	}
	return c
}

func k8sBasedOrchestrator(k8sConfig *central.K8sConfig, shortName, longName string, cluster v1.ClusterType) *cobra.Command {
	c := orchestratorCommand(shortName, longName)
	c.PersistentPreRun = func(*cobra.Command, []string) {
		cfg.K8sConfig = k8sConfig
		cfg.ClusterType = cluster
	}

	c.AddCommand(externalVolume())
	c.AddCommand(hostPathVolume(cluster))
	c.AddCommand(noVolume())

	// Adds k8s specific flags
	c.PersistentFlags().StringVarP(&k8sConfig.Namespace, "namespace", "n", "stackrox", "namespace")
	c.PersistentFlags().StringVarP(&k8sConfig.MonitoringEndpoint, "monitoring-endpoint", "", "monitoring.stackrox", "monitoring endpoint")
	c.PersistentFlags().Var(&monitoringWrapper{Monitoring: &k8sConfig.MonitoringType}, "monitoring-type", "where to host the monitoring (on-prem, none)")
	c.PersistentFlags().StringVarP(&k8sConfig.PreventImage, "prevent-image", "i", "stackrox.io/"+preventImage, "Prevent image to use")
	c.PersistentFlags().StringVarP(&k8sConfig.ClairifyImage, "clairify-image", "", "stackrox.io/"+clairifyImage, "Clairify image to use")
	return c
}

func newK8sConfig(monitoringDefault central.MonitoringType) *central.K8sConfig {
	return &central.K8sConfig{
		MonitoringType: monitoringDefault,
	}
}

func k8s() *cobra.Command {
	k8sConfig := newK8sConfig(central.OnPrem)
	c := k8sBasedOrchestrator(k8sConfig, "k8s", "Kubernetes", v1.ClusterType_KUBERNETES_CLUSTER)
	return c
}

func openshift() *cobra.Command {
	k8sConfig := newK8sConfig(central.OnPrem)
	c := k8sBasedOrchestrator(k8sConfig, "openshift", "Openshift", v1.ClusterType_OPENSHIFT_CLUSTER)
	return c
}
