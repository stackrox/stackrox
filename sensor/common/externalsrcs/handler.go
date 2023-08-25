package externalsrcs

import (
	"bytes"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	log = logging.LoggerForModule()
)

// Store is a store for network graph external sources.
//
//go:generate mockgen-wrapper
type Store interface {
	ExternalSrcsValueStream() concurrency.ReadOnlyValueStream[*sensor.IPNetworkList]
	LookupByNetwork(ipNet pkgNet.IPNetwork) *storage.NetworkEntityInfo
	LookupByID(id string) *storage.NetworkEntityInfo
}

// Handler forwards the external network entities received from Central to Collectors.
type Handler interface {
	common.SensorComponent
}

type handlerImpl struct {
	stopSig   concurrency.Signal
	updateSig concurrency.Signal

	// `entities` stores the IPNetwork to entity object mappings. We allow only unique CIDRs in a cluster, which could
	// be overlapping or not.
	entities map[pkgNet.IPNetwork]*storage.NetworkEntityInfo
	// entitiesById is used for easy lookups during network flow policy evaluation
	entitiesByID     map[string]*storage.NetworkEntityInfo
	lastRequestSeqID int64
	// `lastSeenList` stores the networks in descending lexical byte order. Since, the host identifier bits are all set
	// to 0, this gives us highest-smallest to lowest-largest subnet ordering. e.g. 127.0.0.0/8, 10.10.0.0/24,
	// 10.0.0.0/24, 10.0.0.0/8. This list can be used to lookup the smallest subnet containing an IP address.
	lastSeenList             *sensor.IPNetworkList
	ipNetworkListProtoStream *concurrency.ValueStream[*sensor.IPNetworkList]

	lock sync.Mutex
}

func (h *handlerImpl) Start() error {
	go h.run()
	return nil
}

func (h *handlerImpl) Stop(_ error) {
	h.stopSig.Signal()
}

func (h *handlerImpl) Notify(common.SensorComponentEvent) {}

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

func (h *handlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
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
	// We assume that the network entity object validation is already performed by Central.
	h.entities = make(map[pkgNet.IPNetwork]*storage.NetworkEntityInfo)
	var errList errorhelpers.ErrorList
	for _, entity := range entities {
		ipNet := pkgNet.IPNetworkFromCIDR(entity.GetExternalSource().GetCidr())
		if !ipNet.IsValid() {
			errList.AddStringf("%s (cidr=%s) ", entity.GetId(), entity.GetExternalSource().GetCidr())
			continue
		}
		h.entities[ipNet] = entity
		h.entitiesByID[entity.GetId()] = entity
	}

	if err := errList.ToError(); err != nil {
		log.Errorf("could not process some external sources received from Central: %v", err)
	}
}

func (h *handlerImpl) regenerateAndPushExternalSrcsToValueStream() {
	h.lock.Lock()
	defer h.lock.Unlock()

	defer h.updateSig.Reset()

	ipNetworkList := &sensor.IPNetworkList{}

	for ipNet := range h.entities {
		if ipV4 := ipNet.IP().AsNetIP().To4(); ipV4 != nil {
			ipNetworkList.Ipv4Networks = append(ipNetworkList.Ipv4Networks, ipV4...)
			ipNetworkList.Ipv4Networks = append(ipNetworkList.Ipv4Networks, ipNet.PrefixLen())
		} else if ipV6 := ipNet.IP().AsNetIP().To16(); ipV6 != nil {
			ipNetworkList.Ipv6Networks = append(ipNetworkList.Ipv6Networks, ipV6...)
			ipNetworkList.Ipv6Networks = append(ipNetworkList.Ipv6Networks, ipNet.PrefixLen())
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
	return bytes.Equal(a.GetIpv4Networks(), b.GetIpv4Networks()) &&
		bytes.Equal(a.GetIpv6Networks(), b.GetIpv6Networks())
}

func (h *handlerImpl) ExternalSrcsValueStream() concurrency.ReadOnlyValueStream[*sensor.IPNetworkList] {
	return h.ipNetworkListProtoStream
}

func (h *handlerImpl) LookupByNetwork(ipNet pkgNet.IPNetwork) *storage.NetworkEntityInfo {
	if !ipNet.IsValid() {
		return nil
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	return h.entities[ipNet]
}

func (h *handlerImpl) LookupByID(id string) *storage.NetworkEntityInfo {
	h.lock.Lock()
	defer h.lock.Unlock()

	return h.entitiesByID[id]
}
