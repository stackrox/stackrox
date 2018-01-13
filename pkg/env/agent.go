package env

import (
	"os"
)

var (
	// ClusterID is used to provide a cluster ID to a sensor.
	ClusterID = Setting(clusterID{})
	// ApolloEndpoint is used to provide Apollo's reachable endpoint to a sensor.
	ApolloEndpoint = Setting(apolloEndpoint{})
	// AdvertisedEndpoint is used to provide the Sensor with the endpoint it
	// should advertise to services that need to contact it, within its own cluster.
	AdvertisedEndpoint = Setting(advertisedEndpoint{})
	// Image is the image that should be launched for new benchmarks.
	Image = Setting(image{})
)

type clusterID struct{}

func (c clusterID) EnvVar() string {
	return "ROX_APOLLO_CLUSTER_ID"
}

func (c clusterID) Setting() string {
	return os.Getenv(c.EnvVar())
}

type apolloEndpoint struct{}

func (a apolloEndpoint) EnvVar() string {
	return "ROX_APOLLO_ENDPOINT"
}

func (a apolloEndpoint) Setting() string {
	ep := os.Getenv(a.EnvVar())
	if len(ep) == 0 {
		return "apollo.apollo_net:443"
	}
	return ep
}

type advertisedEndpoint struct{}

func (a advertisedEndpoint) EnvVar() string {
	return "ROX_APOLLO_ADVERTISED_ENDPOINT"
}

func (a advertisedEndpoint) Setting() string {
	ep := os.Getenv(a.EnvVar())
	if len(ep) == 0 {
		return "sensor.apollo_net:443"
	}
	return ep
}

type image struct{}

func (img image) EnvVar() string {
	return "ROX_APOLLO_IMAGE"
}

func (img image) Setting() string {
	name := os.Getenv(img.EnvVar())
	if len(name) == 0 {
		return "stackrox/apollo:latest"
	}
	return name
}
