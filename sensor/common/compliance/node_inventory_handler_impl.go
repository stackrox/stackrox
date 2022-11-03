package compliance

import (
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	errInputChanClosed   = errors.New("channel receiving node inventories is closed")
	errStartMoreThanOnce = errors.New("unable to start the component more than once")
)

type nodeInventoryHandlerImpl struct {
	inventories <-chan *storage.NodeInventory
	toCentral   <-chan *central.MsgFromSensor

	// lock prevents the race condition between Start() [writer] and ResponsesC() [reader]
	lock *sync.Mutex
	// stopC is a command that tells this component to stop
	stopC concurrency.ErrorSignal
	// stoppedC is signaled when the goroutine inside of run() finishes
	stoppedC concurrency.ErrorSignal
}

func (c *nodeInventoryHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedC
}

func (c *nodeInventoryHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.NodeScanningCap}
}

// ResponsesC returns a channel with messages to Central. It must be called after Start() for the channel to be not nil
func (c *nodeInventoryHandlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral == nil {
		log.Panic("Start must be called before ResponsesC")
	}
	return c.toCentral
}

func (c *nodeInventoryHandlerImpl) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral != nil {
		return errStartMoreThanOnce
	}
	c.toCentral = c.run()
	return nil
}

func (c *nodeInventoryHandlerImpl) Stop(err error) {
	c.stopC.SignalWithError(err)
}

func (c *nodeInventoryHandlerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	// This component doesn't actually process or handle any messages sent from Central to Sensor (yet).
	// It uses the sensor component so that the lifecycle (start, stop) can be handled when Sensor starts up.
	return nil
}

// run handles the messages from Compliance and forwards them to Central
// This is the only goroutine that writes into the toCentral channel, thus it is responsible for creating and closing that chan
func (c *nodeInventoryHandlerImpl) run() <-chan *central.MsgFromSensor {
	toC := make(chan *central.MsgFromSensor)
	go func() {
		defer func() {
			c.stoppedC.SignalWithError(c.stopC.Err())
		}()
		defer close(toC)
		for !c.stopC.IsDone() {
			select {
			case <-c.stopC.Done():
				return
			case inventory, ok := <-c.inventories:
				if !ok {
					c.stopC.SignalWithError(errInputChanClosed)
					return
				}
				// TODO(ROX-12943): Merge with the Node and send to Central
				c.handleNodeInventory(toC, inventory)
			}
		}
	}()
	return toC
}

func (c *nodeInventoryHandlerImpl) handleNodeInventory(toC chan *central.MsgFromSensor, inventory *storage.NodeInventory) {
	c.fakeAndSendToCentral(toC, inventory)
}

func (c *nodeInventoryHandlerImpl) fakeAndSendToCentral(toC chan *central.MsgFromSensor, inventory *storage.NodeInventory) {
	if inventory == nil {
		return
	}
	creation := inventory.ScanTime
	nodeResource := &storage.Node{
		Id:                      uuid.NewV4().String(),
		Name:                    inventory.GetNodeName(),
		Taints:                  nil,
		NodeInventory:           inventory,
		Labels:                  map[string]string{"fakeLK": "fakeLV"},
		Annotations:             map[string]string{"fakeAK": "fakeAV"},
		JoinedAt:                &types.Timestamp{Seconds: creation.Seconds, Nanos: creation.Nanos},
		InternalIpAddresses:     []string{"192.168.255.254"},
		ExternalIpAddresses:     []string{"10.10.255.254"},
		ContainerRuntime:        k8sutil.ParseContainerRuntimeVersion("v1.2.3-hardcoded"),
		ContainerRuntimeVersion: "v1.2.3-hardcoded",
		KernelVersion:           "v1.2.4-hardcoded",
		OperatingSystem:         "RHCOS-hardcoded",
		OsImage:                 "v1.2.5-hardcoded",
		KubeletVersion:          "v1.2.6-hardcoded",
		KubeProxyVersion:        "v1.2.7-hardcoded",
		K8SUpdated:              types.TimestampNow(),
	}

	select {
	case <-c.stopC.Done():
		return
	case toC <- &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     nodeResource.GetId(),
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_Node{
					Node: nodeResource,
				},
			},
		},
	}:
	}
}
