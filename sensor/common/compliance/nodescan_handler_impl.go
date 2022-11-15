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
	errInputChanClosed   = errors.New("channel receiving node scans v2 is closed")
	errStartMoreThanOnce = errors.New("unable to start the component more than once")
)

type nodeScanHandlerImpl struct {
	inventories <-chan *storage.NodeInventory
	toCentral   <-chan *central.MsgFromSensor

	// lock prevents the race condition between Start() [writer] and ResponsesC() [reader]
	lock *sync.Mutex
	// stopC is a command that tells this component to stop
	stopC concurrency.ErrorSignal
	// stoppedC is signaled when the goroutine inside of run() finishes
	stoppedC concurrency.ErrorSignal
}

func (c *nodeScanHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedC
}

func (c *nodeScanHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.NodeScanningCap}
}

// ResponsesC returns a channel with messages to Central. It must be called after Start() for the channel to be not nil
func (c *nodeScanHandlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral == nil {
		log.Panic("Start must be called before ResponsesC")
	}
	return c.toCentral
}

func (c *nodeScanHandlerImpl) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral != nil {
		return errStartMoreThanOnce
	}
	c.toCentral = c.run()
	return nil
}

func (c *nodeScanHandlerImpl) Stop(err error) {
	c.stopC.SignalWithError(err)
}

func (c *nodeScanHandlerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	// This component doesn't actually process or handle any messages sent from Central to Sensor (yet).
	// It uses the sensor component so that the lifecycle (start, stop) can be handled when Sensor starts up.
	return nil
}

// run handles the messages from Compliance and forwards them to Central
// This is the only goroutine that writes into the toCentral channel, thus it is responsible for creating and closing that chan
func (c *nodeScanHandlerImpl) run() <-chan *central.MsgFromSensor {
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
			case scan, ok := <-c.inventories:
				if !ok {
					c.stopC.SignalWithError(errInputChanClosed)
					return
				}
				// TODO(ROX-12943): Merge with the Node and send to Central
				c.handleNodeInventory(toC, scan)
			}
		}
	}()
	return toC
}

func (c *nodeScanHandlerImpl) handleNodeInventory(toC chan *central.MsgFromSensor, scan *storage.NodeInventory) {
	c.fakeAndSendToCentral(toC, scan)
}

func (c *nodeScanHandlerImpl) fakeAndSendToCentral(toC chan *central.MsgFromSensor, scan *storage.NodeInventory) {
	if scan == nil {
		return
	}
	creation := scan.ScanTime
	nodeResource := &storage.Node{
		Id:                      uuid.NewV4().String(),
		Name:                    scan.GetNodeName(),
		Taints:                  nil,
		NodeInventory:           scan,
		Labels:                  map[string]string{"fakeLK": "fakeLV"},
		Annotations:             map[string]string{"fakeAK": "fakeAV"},
		JoinedAt:                &types.Timestamp{Seconds: creation.Seconds, Nanos: creation.Nanos},
		InternalIpAddresses:     []string{"192.168.255.254"},
		ExternalIpAddresses:     []string{"10.10.255.254"},
		ContainerRuntime:        k8sutil.ParseContainerRuntimeVersion("docker://20.10.0-hardcoded"),
		ContainerRuntimeVersion: "docker://20.10.0-hardcoded-deprecated",
		KernelVersion:           "v1.2.4-hardcoded",
		OperatingSystem:         "RedHat-CoreOS-hardcoded",
		OsImage:                 "RedHat CoreOS v6.6.6-hardcoded",
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
