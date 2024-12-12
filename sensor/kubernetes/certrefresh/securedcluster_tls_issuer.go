package certrefresh

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certificates"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/securedcluster"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var _ common.SensorComponent = (*securedClusterTLSIssuerImpl)(nil)

// NewSecuredClusterTLSIssuer creates a sensor component that will keep the Secured Cluster certificates
// up to date, using the retry parameters in tls_issuer_common.go
func NewSecuredClusterTLSIssuer(
	k8sClient kubernetes.Interface,
	sensorNamespace string,
	sensorPodName string,
) common.SensorComponent {
	msgToCentralC := make(chan *message.ExpiringMessage)
	msgFromCentralC := make(chan *central.IssueSecuredClusterCertsResponse)
	return &securedClusterTLSIssuerImpl{
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    k8sClient,
		msgToCentralC:                msgToCentralC,
		msgFromCentralC:              msgFromCentralC,
		certRefreshBackoff:           certRefreshBackoff,
		getCertificateRefresherFn:    newCertificatesRefresher,
		getServiceCertificatesRepoFn: securedcluster.NewServiceCertificatesRepo,
		certRequester:                certificates.NewSecuredClusterCertificateRequester(msgToCentralC, msgFromCentralC),
	}
}

type securedClusterTLSIssuerImpl struct {
	sensorNamespace              string
	sensorPodName                string
	k8sClient                    kubernetes.Interface
	msgToCentralC                chan *message.ExpiringMessage
	msgFromCentralC              chan *central.IssueSecuredClusterCertsResponse
	certRefreshBackoff           wait.Backoff
	getCertificateRefresherFn    certificateRefresherGetter
	getServiceCertificatesRepoFn serviceCertificatesRepoGetter
	certRequester                certificates.Requester
	certRefresher                concurrency.RetryTicker
}

// Start starts the Sensor component and launches a certificate refresher that immediately checks the certificates,
// and keeps them updated.
// In case a secret doesn't have the expected owner, this logs a warning and returns nil.
// In case this component was already started, it fails immediately.
func (i *securedClusterTLSIssuerImpl) Start() error {
	log.Debug("Starting Secured Cluster TLS issuer.")
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	if i.certRefresher != nil {
		return i.abortStart(errors.New("already started"))
	}

	sensorOwnerReference, fetchSensorDeploymentErr := FetchSensorDeploymentOwnerRef(ctx, i.sensorPodName,
		i.sensorNamespace, i.k8sClient, fetchSensorDeploymentOwnerRefBackoff)
	if fetchSensorDeploymentErr != nil {
		return i.abortStart(errors.Wrap(fetchSensorDeploymentErr, "fetching sensor deployment"))
	}

	certsRepo := i.getServiceCertificatesRepoFn(*sensorOwnerReference, i.sensorNamespace,
		i.k8sClient.CoreV1().Secrets(i.sensorNamespace))
	i.certRefresher = i.getCertificateRefresherFn("secured cluster certificates", i.certRequester.RequestCertificates, certsRepo,
		certRefreshTimeout, i.certRefreshBackoff)

	i.certRequester.Start()
	if refreshStartErr := i.certRefresher.Start(); refreshStartErr != nil {
		return i.abortStart(errors.Wrap(refreshStartErr, "starting certificate certRefresher"))
	}

	log.Debug("Secured Cluster TLS issuer started.")
	return nil
}

func (i *securedClusterTLSIssuerImpl) abortStart(err error) error {
	log.Errorf("Secured Cluster TLS issuer start aborted due to error: %s", err)
	i.Stop(err)
	return err
}

func (i *securedClusterTLSIssuerImpl) Stop(_ error) {
	if i.certRefresher != nil {
		i.certRefresher.Stop()
		i.certRefresher = nil
	}

	i.certRequester.Stop()
	log.Debug("Secured Cluster TLS issuer stopped.")
}

func (i *securedClusterTLSIssuerImpl) Notify(common.SensorComponentEvent) {}

func (i *securedClusterTLSIssuerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.SecuredClusterCertificatesRefresh}
}

// ResponsesC is called "responses" because for other SensorComponents it is Central that
// initiates the interaction. However, here it is Sensor which sends a request to Central.
func (i *securedClusterTLSIssuerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return i.msgToCentralC
}

// ProcessMessage dispatches Central's messages to Sensor received via the Central receiver.
// This method must not block as it would prevent centralReceiverImpl from sending messages
// to other SensorComponents.
func (i *securedClusterTLSIssuerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	switch m := msg.GetMsg().(type) {
	case *central.MsgToSensor_IssueSecuredClusterCertsResponse:
		response := m.IssueSecuredClusterCertsResponse
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), processMessageTimeout)
			defer cancel()
			select {
			case <-ctx.Done():
				// certRefresher will retry.
				log.Errorf("timeout forwarding response %s from central: %s", response, ctx.Err())
			case i.msgFromCentralC <- response:
			}
		}()
		return nil
	default:
		// messages not supported by this component are ignored because unknown messages types are handled by the Central receiver.
		return nil
	}
}
