package config

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// Handler executes the input scrape commands, and reconciles scrapes with input ComplianceReturns,
// outputing the ScrapeUpdates we expect to be sent back to central.
type Handler interface {
	Stop()

	GetConfig() *storage.DynamicClusterConfig
	SendCommand(cluster *central.ClusterConfig) bool
}

// NewCommandHandler returns a new instance of a Handler using the input image and Orchestrator.
func NewCommandHandler() Handler {
	return &configHandlerImpl{
		stopC: concurrency.NewErrorSignal(),
	}
}

type configHandlerImpl struct {
	config *storage.DynamicClusterConfig
	lock   sync.RWMutex

	stopC concurrency.ErrorSignal
}

func (c *configHandlerImpl) Stop() {
	c.stopC.Signal()
}

func (c *configHandlerImpl) SendCommand(config *central.ClusterConfig) bool {
	select {
	case <-c.stopC.Done():
		return false
	default:
		log.Infof("Received configuration from Central: %s", protoutils.NewWrapper(config))

		c.lock.Lock()
		c.config = config.Config
		c.lock.Unlock()
		return true
	}
}

func (c *configHandlerImpl) GetConfig() *storage.DynamicClusterConfig {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.config
}
