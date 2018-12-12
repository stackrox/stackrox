package deploy

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/roxctl/central/deploy/renderer"
)

func dockerBasedOrchestrator(shortName, longName string, cluster storage.ClusterType) *cobra.Command {
	swarmConfig := new(renderer.SwarmConfig)

	c := orchestratorCommand(shortName, longName)
	c.PersistentPreRun = func(*cobra.Command, []string) {
		cfg.SwarmConfig = swarmConfig
		cfg.ClusterType = cluster
	}
	c.AddCommand(externalVolume())
	c.AddCommand(hostPathVolume(cluster))
	c.AddCommand(noVolume())

	// Adds swarm specific flags
	c.PersistentFlags().StringVarP(&swarmConfig.ClairifyImage, "clairify-image", "", "stackrox.io/"+clairifyImage, "Clairify image to use")
	c.PersistentFlags().StringVarP(&swarmConfig.MainImage, "main-image", "i", "stackrox.io/"+mainImage, "Tmage to use")
	c.PersistentFlags().StringVarP(&swarmConfig.NetworkMode, "mode", "m", "ingress", "network mode to use (ingress or host)")
	c.PersistentFlags().IntVarP(&swarmConfig.PublicPort, "port", "p", 443, "public port to expose")
	return c
}
