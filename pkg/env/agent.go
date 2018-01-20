package env

import (
	"os"
)

var (
	// ClusterID is used to provide a cluster ID to a sensor.
	ClusterID = Setting(clusterID{})
	// CenralEndpoint is used to provide Central's reachable endpoint to a sensor.
	CenralEndpoint = Setting(centralEndpoint{})
	// AdvertisedEndpoint is used to provide the Sensor with the endpoint it
	// should advertise to services that need to contact it, within its own cluster.
	AdvertisedEndpoint = Setting(advertisedEndpoint{})
	// Image is the image that should be launched for new benchmarks.
	Image = Setting(image{})
)

type clusterID struct{}

func (c clusterID) EnvVar() string {
	return "ROX_MITIGATE_CLUSTER_ID"
}

func (c clusterID) Setting() string {
	return os.Getenv(c.EnvVar())
}

type centralEndpoint struct{}

func (a centralEndpoint) EnvVar() string {
	return "ROX_CENTRAL_ENDPOINT"
}

func (a centralEndpoint) Setting() string {
	ep := os.Getenv(a.EnvVar())
	if len(ep) == 0 {
		return "central.mitigate_net:443"
	}
	return ep
}

type advertisedEndpoint struct{}

func (a advertisedEndpoint) EnvVar() string {
	return "ROX_MITIGATE_ADVERTISED_ENDPOINT"
}

func (a advertisedEndpoint) Setting() string {
	ep := os.Getenv(a.EnvVar())
	if len(ep) == 0 {
		return "sensor.mitigate_net:443"
	}
	return ep
}

type image struct{}

func (img image) EnvVar() string {
	return "ROX_MITIGATE_IMAGE"
}

func (img image) Setting() string {
	name := os.Getenv(img.EnvVar())
	if len(name) == 0 {
		return "stackrox/mitigate:latest"
	}
	return name
}
