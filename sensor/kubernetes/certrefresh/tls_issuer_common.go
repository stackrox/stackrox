package certrefresh

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	startTimeout                         = 6 * time.Minute
	fetchSensorDeploymentOwnerRefBackoff = wait.Backoff{
		Duration: 10 * time.Millisecond,
		Factor:   3,
		Jitter:   0.1,
		Steps:    10,
		Cap:      startTimeout,
	}
	processMessageTimeout = 5 * time.Second
	certRefreshTimeout    = 5 * time.Minute
	certRefreshBackoff    = wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   3.0,
		Jitter:   0.1,
		Steps:    5,
		Cap:      10 * time.Minute,
	}
)
