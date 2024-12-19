package certrefresh

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certificates"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/securedcluster"
	"k8s.io/client-go/kubernetes"
)

var (
	securedClusterComponentName    = "secured cluster"
	securedClusterSensorCapability = centralsensor.SecuredClusterCertificatesRefresh
	securedClusterResponseFn       = func(msg *central.MsgToSensor) *certificates.Response {
		return certificates.NewResponseFromSecuredClusterCerts(msg.GetIssueSecuredClusterCertsResponse())
	}
)

// NewSecuredClusterTLSIssuer creates a sensor component that will keep the Secured Cluster certificates
// up to date, using the retry parameters in tls_issuer_common.go
func NewSecuredClusterTLSIssuer(
	k8sClient kubernetes.Interface,
	sensorNamespace string,
	sensorPodName string,
) common.SensorComponent {
	tlsIssuer := &tlsIssuerImpl{
		componentName:                securedClusterComponentName,
		sensorCapability:             securedClusterSensorCapability,
		getResponseFn:                securedClusterResponseFn,
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    k8sClient,
		certRefreshBackoff:           certRefreshBackoff,
		getCertificateRefresherFn:    newCertificatesRefresher,
		getServiceCertificatesRepoFn: securedcluster.NewServiceCertificatesRepo,
		msgToCentralC:                make(chan *message.ExpiringMessage),
		stopSig:                      concurrency.NewErrorSignal(),
	}

	tlsIssuer.certRequester = certificates.NewSecuredClusterCertificateRequester(tlsIssuer.msgToCentralHandler)
	return tlsIssuer
}
