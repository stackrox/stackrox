package reconciliation

import (
	"context"
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/store"
)

var (
	log = logging.LoggerForModule()
)

var _ common.SensorComponent = (*DeduperStateProcessor)(nil)

type DeduperStateProcessor struct {
	responseC      chan *message.ExpiringMessage
	deduperState   map[deduper.Key]uint64
	stateLock      sync.RWMutex
	contextLock    sync.RWMutex
	reconciler     store.HashReconciler
	stateReceived  atomic.Bool
	currentContext context.Context
	cancelContext  func()
}

func NewDeduperStateProcessor(reconciler store.HashReconciler) *DeduperStateProcessor {
	return &DeduperStateProcessor{
		responseC:    make(chan *message.ExpiringMessage),
		stateLock:    sync.RWMutex{},
		deduperState: make(map[deduper.Key]uint64),
		reconciler:   reconciler,
	}
}

func (c *DeduperStateProcessor) SetDeduperState(state map[deduper.Key]uint64) {
	if len(c.deduperState) != 0 {
		log.Warnf("SetDeduperState called but current deduperState is not empty (%d entries being overwritten)", len(state))
	}
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.deduperState = state
	c.stateReceived.Store(true)
}

func (c *DeduperStateProcessor) Notify(e common.SensorComponentEvent) {
	if e == common.SensorComponentEventSyncFinished {
		if !c.stateReceived.Load() {
			log.Warnf("Processing sync event in reconciler without having received a deduper state. No deletes will be generated.")
		}

		c.stateLock.RLock()
		defer c.stateLock.RUnlock()
		messages := c.reconciler.ProcessHashes(c.deduperState)
		log.Infof("Hashes reconciled: %d messages generated", len(messages))
		for _, msg := range messages {
			c.responseC <- c.generateMessageWithCurrentContext(&msg)
		}
		log.Infof("Client reconciliation done")
	} else if e == common.SensorComponentEventOfflineMode {
		c.swapContext()
		c.cleanState()
	}
}

func (c *DeduperStateProcessor) cleanState() {
	c.stateReceived.Store(false)

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.deduperState = make(map[deduper.Key]uint64)
}

func (c *DeduperStateProcessor) swapContext() {
	c.contextLock.Lock()
	defer c.contextLock.Unlock()

	c.cancelContext()
	c.currentContext, c.cancelContext = context.WithCancel(context.Background())
}

func (c *DeduperStateProcessor) generateMessageWithCurrentContext(msg *central.MsgFromSensor) *message.ExpiringMessage {
	c.contextLock.RLock()
	defer c.contextLock.RUnlock()

	return message.NewExpiring(c.currentContext, msg)
}

func (c *DeduperStateProcessor) Start() error {
	c.currentContext, c.cancelContext = context.WithCancel(context.Background())
	return nil
}

func (c *DeduperStateProcessor) Stop(_ error) {}

func (c *DeduperStateProcessor) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (c *DeduperStateProcessor) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (c *DeduperStateProcessor) ResponsesC() <-chan *message.ExpiringMessage {
	return c.responseC
}
