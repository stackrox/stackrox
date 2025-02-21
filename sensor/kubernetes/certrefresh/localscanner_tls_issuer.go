package certrefresh

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/localscanner"
	"k8s.io/client-go/kubernetes"
)

var (
	localScannerComponentName    = "local scanner"
	localScannerSensorCapability = centralsensor.LocalScannerCredentialsRefresh
	localScannerResponseFn       = func(msg *central.MsgToSensor) *Response {
		return NewResponseFromLocalScannerCerts(msg.GetIssueLocalScannerCertsResponse())
	}
)

// NewLocalScannerTLSIssuer creates a sensor component that will keep the local scanner certificates
// up to date, using the retry parameters in tls_issuer_common.go
func NewLocalScannerTLSIssuer(
	k8sClient kubernetes.Interface,
	sensorNamespace string,
	sensorPodName string,
) common.SensorComponent {
	return &tlsIssuerImpl{
		componentName:                localScannerComponentName,
		sensorCapability:             localScannerSensorCapability,
		getResponseFn:                localScannerResponseFn,
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    k8sClient,
		certRefreshBackoff:           certRefreshBackoff,
		getCertificateRefresherFn:    newCertificatesRefresher,
		getServiceCertificatesRepoFn: localscanner.NewServiceCertificatesRepo,
		msgToCentralC:                make(chan *message.ExpiringMessage),
		newMsgFromSensorFn:           newLocalScannerMsgFromSensor,
		responseReceived:             concurrency.NewSignal(),
		requiredCentralCapability:    nil,
	}
}

func newLocalScannerMsgFromSensor(requestID string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
			IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{
				RequestId: requestID,
			},
		},
	}
}
