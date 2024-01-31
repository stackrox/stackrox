package collector

import (
	"os"
	"path"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

type FakeCollectorConfig struct {
	sensorAddress string
	certsPath     string
}

func NewFakeCollector(cfg *FakeCollectorConfig) *FakeCollector {
	stopper := concurrency.NewStopper()
	return &FakeCollector{
		config:             cfg,
		stopper:            stopper,
		networkFlowManager: newFakeNetworkFlowManager(stopper),
		signalManager:      newSignalManager(stopper),
	}
}

func WithDefaultConfig() *FakeCollectorConfig {
	return &FakeCollectorConfig{
		sensorAddress: "localhost:8443",
		certsPath:     "tools/local-sensor/certs",
	}
}

func (cc *FakeCollectorConfig) WithCertsPath(path string) *FakeCollectorConfig {
	cc.certsPath = path
	return cc
}

func (cc *FakeCollectorConfig) WithSensorAddress(address string) *FakeCollectorConfig {
	cc.sensorAddress = address
	return cc
}

type FakeCollector struct {
	config             *FakeCollectorConfig
	stopper            concurrency.Stopper
	networkFlowManager *fakeNetworkFlowManager
	signalManager      *fakeSignalManager
}

func (c *FakeCollector) Start() error {
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", path.Join(c.config.certsPath, "/cert.pem")))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", path.Join(c.config.certsPath, "/key.pem")))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", path.Join(c.config.certsPath, "/caCert.pem")))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", path.Join(c.config.certsPath, "/caKey.pem")))

	if err := c.networkFlowManager.start(c.config.sensorAddress); err != nil {
		return err
	}
	if err := c.signalManager.start(c.config.sensorAddress); err != nil {
		return err
	}
	return nil
}

func (c *FakeCollector) SendFakeNetworkFlow(msg *sensor.NetworkConnectionInfoMessage) {
	c.networkFlowManager.send(msg)
}

func (c *FakeCollector) SendFakeSignal(msg *sensor.SignalStreamMessage) {
	c.signalManager.send(msg)

}
