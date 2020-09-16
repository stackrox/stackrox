package externalsrcs

import (
	"net"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

// Store is a store for network graph external sources.
type Store interface {
	ExternalSrcsValueStream() concurrency.ReadOnlyValueStream
}

// Handler forwards the external network entities received from Central to Collectors.
type Handler interface {
	common.SensorComponent
}

type handlerImpl struct {
	stopSig   concurrency.Signal
	updateSig concurrency.Signal

	// `entities` store the CIDR string to network entity mapping.
	entities         map[string]*storage.NetworkEntityInfo
	lastRequestSeqID int64
	lastSeenList     *sensor.IPNetworkList

	ipNetworkListProtoStream *concurrency.ValueStream

	lock sync.Mutex
}

func (h *handlerImpl) Start() error {
	go h.run()
	return nil
}

func (h *handlerImpl) Stop(_ error) {
	h.stopSig.Signal()
}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.NetworkGraphExternalSrcsCap}
}

func (h *handlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	request := msg.GetPushNetworkEntitiesRequest()
	if request == nil {
		return nil
	}
	select {
	case <-h.stopSig.Done():
		return errors.New("could not process external network entities request")
	default:
		h.lock.Lock()
		defer h.lock.Unlock()

		if request.GetSeqID() < h.lastRequestSeqID {
			return nil
		}
		h.lastRequestSeqID = request.GetSeqID()

		h.saveEntitiesNoLock(request.GetEntities())
		h.updateSig.Signal()
		return nil
	}
}

func (h *handlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func (h *handlerImpl) ExternalSrcsValueStream() concurrency.ReadOnlyValueStream {
	return h.ipNetworkListProtoStream
}

func (h *handlerImpl) run() {
	for {
		select {
		case <-h.updateSig.Done():
			h.regenerateAndPushExternalSrcsToValueStream()
		case <-h.stopSig.Done():
			return
		}
	}
}

func (h *handlerImpl) saveEntitiesNoLock(entities []*storage.NetworkEntityInfo) {
	// We assume that the network entity validation is already performed by Central.
	h.entities = make(map[string]*storage.NetworkEntityInfo)
	for _, entity := range entities {
		h.entities[entity.GetExternalSource().GetCidr()] = entity
	}
}

func (h *handlerImpl) regenerateAndPushExternalSrcsToValueStream() {
	h.lock.Lock()
	defer h.lock.Unlock()

	defer h.updateSig.Reset()

	ipNetworkList := &sensor.IPNetworkList{}

	var errList errorhelpers.ErrorList
	for cidr := range h.entities {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			errList.AddError(err)
			continue
		}

		if ipV4 := ipNet.IP.To4(); ipV4 != nil {
			ones, _ := ipNet.Mask.Size()
			ipNetworkList.Ipv4Networks = append(ipNetworkList.Ipv4Networks, ipV4...)
			ipNetworkList.Ipv4Networks = append(ipNetworkList.Ipv4Networks, byte(uint8(ones)))
		} else if ipV6 := ipNet.IP.To16(); ipV6 != nil {
			ones, _ := ipNet.Mask.Size()
			ipNetworkList.Ipv6Networks = append(ipNetworkList.Ipv6Networks, ipV6...)
			ipNetworkList.Ipv6Networks = append(ipNetworkList.Ipv6Networks, byte(uint8(ones)))
		}
	}

	normalizeNetworkList(ipNetworkList)

	if h.lastSeenList != nil && networkListsEqual(ipNetworkList, h.lastSeenList) {
		return
	}

	h.ipNetworkListProtoStream.Push(ipNetworkList)
	h.lastSeenList = ipNetworkList
}

func normalizeNetworkList(listProto *sensor.IPNetworkList) {
	sort.Sort(sortableIPv4NetworkSlice(listProto.GetIpv4Networks()))
	sort.Sort(sortableIPv6NetworkSlice(listProto.GetIpv6Networks()))
}

func networkListsEqual(a, b *sensor.IPNetworkList) bool {
	return sliceutils.ByteEqual(a.GetIpv4Networks(), b.GetIpv4Networks()) &&
		sliceutils.ByteEqual(a.GetIpv6Networks(), b.GetIpv6Networks())
}
