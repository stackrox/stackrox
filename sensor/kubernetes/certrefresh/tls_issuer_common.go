package certrefresh

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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

type certificateRefresherGetter func(certsDescription string, requestCertificates requestCertificatesFunc,
	repository certrepo.ServiceCertificatesRepo, timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker

type serviceCertificatesRepoGetter func(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) certrepo.ServiceCertificatesRepo
