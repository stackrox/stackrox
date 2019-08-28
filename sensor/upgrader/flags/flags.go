package flags

import (
	"flag"
)

var (
	// KubeConfigSource is the source of the kubernetes config.
	KubeConfigSource = flag.String("kube-config", "in-cluster", "source for the Kubernetes config")
)
