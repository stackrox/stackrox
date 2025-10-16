package compliance

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/pkg/rhctag"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/detector/metrics"
	"github.com/stackrox/rox/sensor/common/message"
	"google.golang.org/protobuf/proto"
)

var (
	errInventoryInputChanClosed = errors.New("channel receiving node inventories is closed")
	errIndexInputChanClosed     = errors.New("channel receiving node indexes is closed")
	errStartMoreThanOnce        = errors.New("unable to start the component more than once")
)

const (
	rhcosFullName = "Red Hat Enterprise Linux CoreOS"

	goldenKey = rhcc.RepositoryKey
)

var (
	goldenName = rhcc.GoldRepo.Name
	goldenURI  = rhcc.GoldRepo.URI
)

type nodeInventoryHandlerImpl struct {
	inventories  <-chan *storage.NodeInventory
	reportWraps  <-chan *index.IndexReportWrap
	toCentral    <-chan *message.ExpiringMessage
	centralReady concurrency.Signal
	// acksFromCentral is for connecting the replies from Central with the toCompliance chan
	acksFromCentral  chan common.MessageToComplianceWithAddress
	toCompliance     chan common.MessageToComplianceWithAddress
	nodeMatcher      NodeIDMatcher
	nodeRHCOSMatcher NodeRHCOSMatcher
	// lock prevents the race condition between Start() [writer] and ResponsesC() [reader]
	lock    *sync.Mutex
	stopper concurrency.Stopper
	// archCache stores an architecture per node, so that it can be used in the index report for
	// the 'rhcos' package. The arch is discovered once and then reused for subsequent scans.
	archCache map[string]string
}

func (c *nodeInventoryHandlerImpl) Name() string {
	return "compliance.nodeInventoryHandlerImpl"
}

func (c *nodeInventoryHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return c.stopper.Client().Stopped()
}

func (c *nodeInventoryHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ResponsesC returns a channel with messages to Central. It must be called after Start() for the channel to be not nil
func (c *nodeInventoryHandlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral == nil {
		log.Panic("Start must be called before ResponsesC")
	}
	return c.toCentral
}

// ComplianceC returns a channel with messages to Compliance
func (c *nodeInventoryHandlerImpl) ComplianceC() <-chan common.MessageToComplianceWithAddress {
	return c.toCompliance
}

func (c *nodeInventoryHandlerImpl) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral != nil || c.toCompliance != nil {
		return errStartMoreThanOnce
	}
	c.toCompliance = make(chan common.MessageToComplianceWithAddress)
	c.toCentral = c.run()
	return nil
}

func (c *nodeInventoryHandlerImpl) Stop() {
	if !c.stopper.Client().Stopped().IsDone() {
		defer utils.IgnoreError(c.stopper.Client().Stopped().Wait)
	}
	c.stopper.Client().Stop()
}

func (c *nodeInventoryHandlerImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		c.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		// As Compliance enters a retry loop when it is not receiving an ACK,
		// there is no need to do anything when entering offline mode
		c.centralReady.Reset()
	}
}

func (c *nodeInventoryHandlerImpl) Accepts(msg *central.MsgToSensor) bool {
	return msg.GetNodeInventoryAck() != nil
}

func (c *nodeInventoryHandlerImpl) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	ackMsg := msg.GetNodeInventoryAck()
	if ackMsg == nil {
		return nil
	}
	log.Debugf("Received node-scanning-ACK message of type %s, action %s for node %s",
		ackMsg.GetMessageType(), ackMsg.GetAction(), ackMsg.GetNodeName())
	metrics.ObserveNodeScanningAck(ackMsg.GetNodeName(),
		ackMsg.GetAction().String(),
		ackMsg.GetMessageType().String(),
		metrics.AckOperationReceive,
		"", metrics.AckOriginSensor)
	switch ackMsg.GetAction() {
	case central.NodeInventoryACK_ACK:
		switch ackMsg.GetMessageType() {
		case central.NodeInventoryACK_NodeIndexer:
			c.sendAckToCompliance(ackMsg.GetNodeName(),
				sensor.MsgToCompliance_NodeInventoryACK_ACK,
				sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer, metrics.AckReasonForwardingFromCentral)
		default:
			// If Central version is behind Sensor, then MessageType field will be unset - then default to NodeInventory.
			c.sendAckToCompliance(ackMsg.GetNodeName(),
				sensor.MsgToCompliance_NodeInventoryACK_ACK,
				sensor.MsgToCompliance_NodeInventoryACK_NodeInventory, metrics.AckReasonForwardingFromCentral)
		}
	case central.NodeInventoryACK_NACK:
		switch ackMsg.GetMessageType() {
		case central.NodeInventoryACK_NodeIndexer:
			c.sendAckToCompliance(ackMsg.GetNodeName(),
				sensor.MsgToCompliance_NodeInventoryACK_NACK,
				sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer, metrics.AckReasonForwardingFromCentral)
		default:
			// If Central version is behind Sensor, then MessageType field will be unset - then default to NodeInventory.
			c.sendAckToCompliance(ackMsg.GetNodeName(),
				sensor.MsgToCompliance_NodeInventoryACK_NACK,
				sensor.MsgToCompliance_NodeInventoryACK_NodeInventory, metrics.AckReasonForwardingFromCentral)
		}
	}
	return nil
}

// run handles the messages from Compliance and forwards them to Central
// This is the only goroutine that writes into the toCentral channel, thus it is responsible for creating and closing that chan
func (c *nodeInventoryHandlerImpl) run() (toCentral <-chan *message.ExpiringMessage) {
	ch2Central := make(chan *message.ExpiringMessage)
	go func() {
		defer func() {
			c.stopper.Flow().ReportStopped()
			close(ch2Central)
		}()
		log.Debugf("NodeInventory/NodeIndex handler is running")
		for {
			select {
			case <-c.stopper.Flow().StopRequested():
				return
			case inventory, ok := <-c.inventories:
				if !ok {
					c.stopper.Flow().StopWithError(errInventoryInputChanClosed)
					return
				}
				c.handleNodeInventory(inventory, ch2Central)
			case wrap, ok := <-c.reportWraps:
				if !ok {
					c.stopper.Flow().StopWithError(errIndexInputChanClosed)
					return
				}
				c.handleNodeIndex(wrap, ch2Central)
			}
		}
	}()
	return ch2Central
}

func (c *nodeInventoryHandlerImpl) handleNodeInventory(
	inventory *storage.NodeInventory,
	toCentral chan *message.ExpiringMessage,
) {
	log.Debugf("Handling NodeInventory...")
	if inventory == nil {
		log.Warn("Received nil node inventory: not sending to Central")
		metrics.ObserveNodeScan("nil", metrics.NodeScanTypeNodeInventory, metrics.NodeScanOperationReceive)
		return
	}
	metrics.ObserveNodeScan(inventory.GetNodeName(), metrics.NodeScanTypeNodeInventory, metrics.NodeScanOperationReceive)
	if !c.centralReady.IsDone() {
		log.Warn("Received NodeInventory but Central is not reachable. Requesting Compliance to resend NodeInventory later")
		c.sendAckToCompliance(inventory.GetNodeName(),
			sensor.MsgToCompliance_NodeInventoryACK_NACK,
			sensor.MsgToCompliance_NodeInventoryACK_NodeInventory, metrics.AckReasonCentralUnreachable)
		return
	}

	if nodeID, err := c.nodeMatcher.GetNodeID(inventory.GetNodeName()); err != nil {
		log.Warnf("Node %q unknown to Sensor. Requesting Compliance to resend NodeInventory later", inventory.GetNodeName())
		c.sendAckToCompliance(inventory.GetNodeName(),
			sensor.MsgToCompliance_NodeInventoryACK_NACK,
			sensor.MsgToCompliance_NodeInventoryACK_NodeInventory,
			metrics.AckReasonNodeUnknown)

	} else {
		inventory.SetNodeId(nodeID)
		log.Debugf("Mapping NodeInventory name '%s' to Node ID '%s'", inventory.GetNodeName(), nodeID)
		c.sendNodeInventory(toCentral, inventory)
	}
}

func (c *nodeInventoryHandlerImpl) handleNodeIndex(
	index *index.IndexReportWrap,
	toCentral chan *message.ExpiringMessage,
) {
	if index == nil || index.IndexReport == nil {
		log.Warn("Received nil index report: not sending to Central")
		metrics.ObserveNodeScan("nil", metrics.NodeScanTypeNodeIndex, metrics.NodeScanOperationReceive)
		return
	}
	metrics.ObserveNodeScan(index.NodeName, metrics.NodeScanTypeNodeIndex, metrics.NodeScanOperationReceive)
	if !c.centralReady.IsDone() {
		log.Warn("Received IndexReport but Central is not reachable. Requesting Compliance to resend later.")
		c.sendAckToCompliance(index.NodeName,
			sensor.MsgToCompliance_NodeInventoryACK_NACK,
			sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer,
			metrics.AckReasonCentralUnreachable)
		return
	}

	if nodeID, err := c.nodeMatcher.GetNodeID(index.NodeName); err != nil {
		log.Warnf("Received Index Report from Node %q that is unknown to Sensor. Requesting Compliance to resend later.", index.NodeName)
		c.sendAckToCompliance(index.NodeName,
			sensor.MsgToCompliance_NodeInventoryACK_NACK,
			sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer,
			metrics.AckReasonNodeUnknown)
	} else {
		index.NodeID = nodeID
		log.Debugf("Mapping IndexReport name '%s' to Node ID '%s'", index.NodeName, nodeID)
		c.sendNodeIndex(toCentral, index)
	}
}

func (c *nodeInventoryHandlerImpl) sendAckToCompliance(
	nodeName string,
	action sensor.MsgToCompliance_NodeInventoryACK_Action,
	messageType sensor.MsgToCompliance_NodeInventoryACK_MessageType,
	reason metrics.AckReason,
) {
	select {
	case <-c.stopper.Flow().StopRequested():
	case c.toCompliance <- common.MessageToComplianceWithAddress{
		Msg: sensor.MsgToCompliance_builder{
			Ack: sensor.MsgToCompliance_NodeInventoryACK_builder{
				Action:      action,
				MessageType: messageType,
			}.Build(),
		}.Build(),
		Hostname:  nodeName,
		Broadcast: nodeName == "",
	}:
	}
	metrics.ObserveNodeScanningAck(nodeName,
		action.String(),
		messageType.String(),
		metrics.AckOperationSend,
		reason, metrics.AckOriginSensor)
}

func (c *nodeInventoryHandlerImpl) sendNodeInventory(toC chan<- *message.ExpiringMessage, inventory *storage.NodeInventory) {
	if inventory == nil {
		return
	}
	select {
	case <-c.stopper.Flow().StopRequested():
		se := &central.SensorEvent{}
		se.SetId(inventory.GetNodeId())
		// ResourceAction_UNSET_ACTION_RESOURCE is the only one supported by Central 4.6 and older.
		// This can be changed to CREATE or UPDATE for Sensor 4.8 or when Central 4.6 is out of support.
		se.SetAction(central.ResourceAction_UNSET_ACTION_RESOURCE)
		se.SetNodeInventory(proto.ValueOrDefault(inventory))
	case toC <- message.New(central.MsgFromSensor_builder{
		Event: se,
	}.Build()):
		metrics.ObserveReceivedNodeInventory(inventory) // keeping for compatibility with 4.6. Remove in 4.8
		metrics.ObserveNodeScan(inventory.GetNodeName(), metrics.NodeScanTypeNodeInventory, metrics.NodeScanOperationSendToCentral)
	}
}

func (c *nodeInventoryHandlerImpl) sendNodeIndex(toC chan<- *message.ExpiringMessage, indexWrap *index.IndexReportWrap) {
	if indexWrap == nil || indexWrap.IndexReport == nil {
		log.Debugf("Empty IndexReport - not sending to Central")
		return
	}

	isRHCOS, version, err := c.nodeRHCOSMatcher.GetRHCOSVersion(indexWrap.NodeName)
	if err != nil {
		log.Warnf("Unable to determine RHCOS version for node %q: %v", indexWrap.NodeName, err)
		isRHCOS = false
	}
	log.Debugf("Node=%q discovered RHCOS=%t rhcos-version=%q", indexWrap.NodeName, isRHCOS, version)

	select {
	case <-c.stopper.Flow().StopRequested():
	default:
		defer func() {
			log.Debugf("Sent IndexReport to Central")
			metrics.ObserveReceivedNodeIndex(indexWrap.NodeName) // keeping for compatibility with 4.6. Remove in 4.8
			metrics.ObserveNodeScan(indexWrap.NodeName, metrics.NodeScanTypeNodeIndex, metrics.NodeScanOperationSendToCentral)
		}()
		irWrapperFunc := noop
		arch := c.archCache[indexWrap.NodeName]
		if isRHCOS {
			if _, ok := c.archCache[indexWrap.NodeName]; !ok {
				arch = extractArch(indexWrap.IndexReport)
				c.archCache[indexWrap.NodeName] = arch
			}
			log.Debugf("Attaching OCI entry for 'rhcos' to index-report for node %s: version=%s, arch=%s", indexWrap.NodeName, version, arch)
			irWrapperFunc = attachRPMtoRHCOS
		}
		se := &central.SensorEvent{}
		se.SetId(indexWrap.NodeID)
		// ResourceAction_UNSET_ACTION_RESOURCE is the only one supported by Central 4.6 and older.
		// This can be changed to CREATE or UPDATE for Sensor 4.8 or when Central 4.6 is out of support.
		se.SetAction(central.ResourceAction_UNSET_ACTION_RESOURCE)
		se.SetIndexReport(proto.ValueOrDefault(irWrapperFunc(version, arch, indexWrap.IndexReport)))
		toC <- message.New(central.MsgFromSensor_builder{
			Event: se,
		}.Build())
	}
}

func normalizeVersion(version string) []int32 {
	rhctagVersion, err := rhctag.Parse(version)
	if err != nil {
		return []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	}
	m := rhctagVersion.MinorStart()
	v := m.Version(true).V
	// Only two first fields matter for the initial db query that matches the vulnerabilities.
	// The results of that query will be further filtered using the string value of the Version field.
	return []int32{v[0], v[1], 0, 0, 0, 0, 0, 0, 0, 0}
}

func noop(_, _ string, rpm *v4.IndexReport) *v4.IndexReport {
	return rpm
}

func idTaken[T any](m map[string]T, id int) bool {
	_, exists := m[strconv.Itoa(id)]
	return exists
}

// extractArch deduces the architecture of the node OS based on the index report containing rpm packages.
func extractArch(rpm *v4.IndexReport) string {
	for _, distro := range rpm.GetContents().GetDistributions() {
		if distro.GetArch() != "" && distro.GetArch() != "noarch" {
			return distro.GetArch()
		}
	}
	for _, distro := range rpm.GetContents().GetDistributionsDEPRECATED() {
		if distro.GetArch() != "" && distro.GetArch() != "noarch" {
			return distro.GetArch()
		}
	}
	for _, p := range rpm.GetContents().GetPackages() {
		if p.GetArch() != "" && p.GetArch() != "noarch" {
			return p.GetArch()
		}
	}
	for _, p := range rpm.GetContents().GetPackagesDEPRECATED() {
		if p.GetArch() != "" && p.GetArch() != "noarch" {
			return p.GetArch()
		}
	}
	return ""
}

func attachRPMtoRHCOS(version, arch string, rpm *v4.IndexReport) *v4.IndexReport {
	idCandidate := 600 // Arbitrary selected. RHCOS has usually 520-560 rpm packages.
	envs := rpm.GetContents().GetEnvironments()
	if len(envs) == 0 {
		envs = rpm.GetContents().GetEnvironmentsDEPRECATED()
	}
	for idTaken(envs, idCandidate) {
		idCandidate++
	}
	strID := strconv.Itoa(idCandidate)
	oci := buildRHCOSIndexReport(strID, version, arch)
	for pkgID, pkg := range rpm.GetContents().GetPackages() {
		oci.GetContents().GetPackages()[pkgID] = pkg
	}
	oci.GetContents().SetPackagesDEPRECATED(append(oci.GetContents().GetPackagesDEPRECATED(), rpm.GetContents().GetPackagesDEPRECATED()...))
	for repoID, repo := range rpm.GetContents().GetRepositories() {
		oci.GetContents().GetRepositories()[repoID] = repo
	}
	oci.GetContents().SetRepositoriesDEPRECATED(append(oci.GetContents().GetRepositoriesDEPRECATED(), rpm.GetContents().GetRepositoriesDEPRECATED()...))
	for envId, list := range rpm.GetContents().GetEnvironments() {
		oci.GetContents().GetEnvironments()[envId] = list
	}
	for envId, list := range rpm.GetContents().GetEnvironmentsDEPRECATED() {
		oci.GetContents().GetEnvironmentsDEPRECATED()[envId] = list
	}
	oci.GetContents().SetDistributions(rpm.GetContents().GetDistributions())
	oci.GetContents().SetDistributionsDEPRECATED(rpm.GetContents().GetDistributionsDEPRECATED())
	return oci
}

func buildRHCOSIndexReport(Id, version, arch string) *v4.IndexReport {
	return v4.IndexReport_builder{
		// This hashId is arbitrary. The value doesn't play a role for matcher, but must be valid sha256.
		HashId:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		State:   controller.IndexFinished.String(),
		Success: true,
		Err:     "",
		Contents: v4.Contents_builder{
			Packages: map[string]*v4.Package{
				Id: v4.Package_builder{
					Id:      Id,
					Name:    "rhcos",
					Version: version,
					NormalizedVersion: v4.NormalizedVersion_builder{
						Kind: "rhctag",
						V:    normalizeVersion(version), // Only two first fields matter for the db-query.
					}.Build(),
					Kind: "binary",
					Source: v4.Package_builder{
						Id:      Id,
						Name:    "rhcos",
						Kind:    "source",
						Version: version,
						Cpe:     "cpe:2.3:*", // required to pass validation of scanner V4 API
					}.Build(),
					Arch: arch,
					Cpe:  "cpe:2.3:*", // required to pass validation of scanner V4 API
				}.Build(),
			},
			PackagesDEPRECATED: []*v4.Package{
				v4.Package_builder{
					Id:      Id,
					Name:    "rhcos",
					Version: version,
					NormalizedVersion: v4.NormalizedVersion_builder{
						Kind: "rhctag",
						V:    normalizeVersion(version), // Only two first fields matter for the db-query.
					}.Build(),
					Kind: "binary",
					Source: v4.Package_builder{
						Id:      Id,
						Name:    "rhcos",
						Kind:    "source",
						Version: version,
						Cpe:     "cpe:2.3:*", // required to pass validation of scanner V4 API
					}.Build(),
					Arch: arch,
					Cpe:  "cpe:2.3:*", // required to pass validation of scanner V4 API
				}.Build(),
			},
			Repositories: map[string]*v4.Repository{
				Id: v4.Repository_builder{
					Id:   Id,
					Name: goldenName,
					Key:  goldenKey,
					Uri:  goldenURI,
					Cpe:  "cpe:2.3:*", // required to pass validation of scanner V4 API
				}.Build(),
			},
			RepositoriesDEPRECATED: []*v4.Repository{
				v4.Repository_builder{
					Id:   Id,
					Name: goldenName,
					Key:  goldenKey,
					Uri:  goldenURI,
					Cpe:  "cpe:2.3:*", // required to pass validation of scanner V4 API
				}.Build(),
			},
			// Environments must be present for the matcher to discover records
			Environments: map[string]*v4.Environment_List{
				Id: v4.Environment_List_builder{
					Environments: []*v4.Environment{
						v4.Environment_builder{
							PackageDb: "",
							// IntroducedIn must be a valid sha256, but the value is not important.
							IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							RepositoryIds: []string{Id},
						}.Build(),
					},
				}.Build(),
			},
			EnvironmentsDEPRECATED: map[string]*v4.Environment_List{
				Id: v4.Environment_List_builder{
					Environments: []*v4.Environment{
						v4.Environment_builder{
							PackageDb: "",
							// IntroducedIn must be a valid sha256, but the value is not important.
							IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							RepositoryIds: []string{Id},
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}.Build()
}
