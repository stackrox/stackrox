package flags

import (
	"flag"
	"time"
)

var (
	// KubeConfigSource is the source of the kubernetes config.
	KubeConfigSource = flag.String("kube-config", "in-cluster", "source for the Kubernetes config")
	// KubeTimeout specifies the maximum duration for Kubernetes API operations:
	//   - If zero, no timeout is set (requests may run indefinitely).
	//   - If negative, an error is returned during config initialization.
	//   - If positive, the timeout is applied to all client operations.
	KubeTimeout = flag.Duration("kube-timeout", 60*time.Second, "timeout for Kubernetes API operations")
)
