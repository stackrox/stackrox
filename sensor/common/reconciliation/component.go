package reconciliation

import (
	"context"
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/deduperkey"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/store"
)

var (
	log = logging.LoggerForModule()
)

var _ common.SensorComponent = (*DeduperStateProcessor)(nil)

// DeduperStateProcessor is the component that processes deduper state and generates delete messages for removed
// resources.
type DeduperStateProcessor struct {
	responseC      chan *message.ExpiringMessage
	deduperState   map[deduperkey.Key]uint64
	stateLock      sync.RWMutex
	contextLock    sync.RWMutex
	reconciler     store.HashReconciler
	stateReceived  atomic.Bool
	currentContext context.Context
	cancelContext  func()
}

// NewDeduperStateProcessor returns a new DeduperStateProcessor using store.HashReconciler dependency.
func NewDeduperStateProcessor(reconciler store.HashReconciler) *DeduperStateProcessor {
	return &DeduperStateProcessor{
		responseC:    make(chan *message.ExpiringMessage),
		stateLock:    sync.RWMutex{},
		deduperState: make(map[deduperkey.Key]uint64),
		reconciler:   reconciler,
	}
}

// SetDeduperState should be used when the Deduper State message is received from central, and it should be called
// only once per active connection. Any new state received will overwrite the existing state in this component.
func (c *DeduperStateProcessor) SetDeduperState(state map[deduperkey.Key]uint64) {
	if len(c.deduperState) != 0 {
		log.Warnf("SetDeduperState called but current deduperState is not empty (%d entries being overwritten)", len(state))
	}
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.deduperState = state
	c.stateReceived.Store(true)
}

// Notify processes sensor component event messages.
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
			msg := msg
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
	c.deduperState = make(map[deduperkey.Key]uint64)
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

// Start sensor component.
func (c *DeduperStateProcessor) Start() error {
	c.currentContext, c.cancelContext = context.WithCancel(context.Background())
	return nil
}

// Stop sensor component.
func (c *DeduperStateProcessor) Stop(_ error) {}

// Capabilities returns the set of features supported by this sensor.
func (c *DeduperStateProcessor) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ProcessMessage processes messages coming from central.
func (c *DeduperStateProcessor) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

// ResponsesC returns the channel where Sensor messages are written to.
func (c *DeduperStateProcessor) ResponsesC() <-chan *message.ExpiringMessage {
	return c.responseC
}
