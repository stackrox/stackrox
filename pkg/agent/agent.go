// Package agent facilitates interactions with deployed cluster agents.
// Agents are the "beachhead" in the cluster, and report back to central Apollo.
package agent

import "os"

// A Setting is a runtime configuration set using an environment variable.
type Setting interface {
	EnvVar() string
	Setting() string
}

var (
	// ClusterID is used to provide a cluster ID to an agent.
	ClusterID = Setting(clusterID{})
	// ApolloEndpoint is used to provide Apollo's reachable endpoint to an agent.
	ApolloEndpoint = Setting(apolloEndpoint{})
	// AdvertisedEndpoint is used to provide the Agent with the endpoint it
	// should advertise to services that need to contact it, within its own cluster.
	AdvertisedEndpoint = Setting(advertisedEndpoint{})
)

type clusterID struct{}

func (c clusterID) EnvVar() string {
	return "ROX_APOLLO_CLUSTER_ID"
}

func (c clusterID) Setting() string {
	return os.Getenv(c.EnvVar())
}

type apolloEndpoint struct{}

func (c apolloEndpoint) EnvVar() string {
	return "ROX_APOLLO_ENDPOINT"
}

func (c apolloEndpoint) Setting() string {
	ep := os.Getenv(c.EnvVar())
	if len(ep) == 0 {
		return "apollo.apollo_net:8080"
	}
	return ep
}

type advertisedEndpoint struct{}

func (c advertisedEndpoint) EnvVar() string {
	return "ROX_APOLLO_ADVERTISED_ENDPOINT"
}

func (c advertisedEndpoint) Setting() string {
	ep := os.Getenv(c.EnvVar())
	if len(ep) == 0 {
		return "agent.apollo_net:8080"
	}
	return ep
}
