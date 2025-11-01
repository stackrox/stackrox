package flags

import (
	"flag"
	"time"
)

var (
	// KubeConfigSource is the source of the kubernetes config.
	KubeConfigSource = flag.String("kube-config", "in-cluster", "source for the Kubernetes config")
	// KubeTimeout is the timeout for Kubernetes API operations.
	KubeTimeout = flag.Duration("kube-timeout", 60*time.Second, "timeout for Kubernetes API operations")
)
