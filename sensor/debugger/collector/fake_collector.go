package collector

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

// FakeCollectorConfig FakeCollector's configuration.
type FakeCollectorConfig struct {
	sensorAddress string
	certsPath     string
}

// NewFakeCollector creates a new FakeCollector.
func NewFakeCollector(cfg *FakeCollectorConfig) *FakeCollector {
	stopper := concurrency.NewStopper()
	return &FakeCollector{
		config:             cfg,
		stopper:            stopper,
		networkFlowManager: newFakeNetworkFlowManager(stopper),
		signalManager:      newSignalManager(stopper),
	}
}

// WithDefaultConfig initializes the FakeCollector's default configuration.
func WithDefaultConfig() *FakeCollectorConfig {
	return &FakeCollectorConfig{
		sensorAddress: "localhost:8443",
		certsPath:     "tools/local-sensor/certs",
	}
}

// WithCertsPath sets the certificates' path.
func (cc *FakeCollectorConfig) WithCertsPath(path string) *FakeCollectorConfig {
	cc.certsPath = path
	return cc
}

// WithSensorAddress sets sensor's address.
func (cc *FakeCollectorConfig) WithSensorAddress(address string) *FakeCollectorConfig {
	cc.sensorAddress = address
	return cc
}

// FakeCollector a fake collector for testing.
type FakeCollector struct {
	config             *FakeCollectorConfig
	stopper            concurrency.Stopper
	networkFlowManager *fakeNetworkFlowManager
	signalManager      *fakeSignalManager
}

// Start FakeCollector.
func (c *FakeCollector) Start() error {
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", path.Join(c.config.certsPath, "/cert.pem")))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", path.Join(c.config.certsPath, "/key.pem")))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", path.Join(c.config.certsPath, "/caCert.pem")))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", path.Join(c.config.certsPath, "/caKey.pem")))

	if err := retry.WithRetry(func() error {
		log.Infof("Attempting to start network flow manager...")
		return c.networkFlowManager.start(c.config.sensorAddress)
	}, retry.Tries(10), retry.WithExponentialBackoff()); err != nil {
		return errors.Wrap(err, "starting network flow manager for fake collector")
	}
	log.Infof("Network flow manager started")

	if err := retry.WithRetry(func() error {
		log.Infof("Attempting to start signal manager...")
		return c.signalManager.start(c.config.sensorAddress)
	}, retry.Tries(5)); err != nil {
		return errors.Wrap(err, "starting signal manager for fake collector")
	}
	log.Infof("Signal manager started")

	return nil
}

// Stop FakeCollector.
func (c *FakeCollector) Stop() {
	c.stopper.Client().Stop()
}

// SendFakeNetworkFlow sends a NetworkConnectionInfoMessage to sensor.
func (c *FakeCollector) SendFakeNetworkFlow(msg *sensor.NetworkConnectionInfoMessage) {
	c.networkFlowManager.send(msg)
}

// SendFakeSignal sends a SignalStreamMessage to sensor.
func (c *FakeCollector) SendFakeSignal(msg *sensor.SignalStreamMessage) {
	c.signalManager.send(msg)

}
