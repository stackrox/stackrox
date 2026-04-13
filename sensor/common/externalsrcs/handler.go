package externalsrcs

import (
	"bytes"
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
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

	// pendingEntities holds the raw entity list from Central until first lookup.
	// This lazy-loads the entity index: if no policy evaluation ever queries
	// external entities, the ~16 MB CIDR index is never allocated.
	pendingEntities []*storage.NetworkEntityInfo
	indexed         bool

	// entities stores the IPNetwork to entity object mappings. We allow only unique CIDRs in a cluster, which could
	// be overlapping or not. Populated lazily on first lookup.
	entities map[pkgNet.IPNetwork]*storage.NetworkEntityInfo
	// entitiesById is used for easy lookups during network flow policy evaluation
	entitiesByID     map[string]*storage.NetworkEntityInfo
	lastRequestSeqID int64
	// lastSeenList stores the networks in descending lexical byte order. Since, the host identifier bits are all set
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

func (h *handlerImpl) Stop() {
	h.stopSig.Signal()
}

func (h *handlerImpl) Name() string {
	return "externalsrcs.handlerImpl"
}

func (h *handlerImpl) Notify(common.SensorComponentEvent) {}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.NetworkGraphExternalSrcsCap}
}

func (h *handlerImpl) Accepts(msg *central.MsgToSensor) bool {
	return msg.GetPushNetworkEntitiesRequest() != nil
}

func (h *handlerImpl) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
	if env.SensorLite.BooleanSetting() {
		// In lite mode, skip loading network entity knowledge base.
		// Saves ~16 MB. Network flows show raw IPs instead of cloud labels.
		return nil
	}
	request := msg.GetPushNetworkEntitiesRequest()
	if request == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return errors.Wrapf(ctx.Err(), "message processing in component %s", h.Name())
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
	// Store the raw entity list without building the CIDR index.
	// The index is built lazily on first LookupByNetwork/LookupByID call.
	// This saves ~16 MB on clusters where no network policy references external entities.
	h.pendingEntities = entities
	h.indexed = false
	h.entities = nil
	h.entitiesByID = nil
}

// ensureIndexedNoLock builds the CIDR → entity maps from pendingEntities on first access.
func (h *handlerImpl) ensureIndexedNoLock() {
	if h.indexed {
		return
	}
	h.indexed = true

	h.entities = make(map[pkgNet.IPNetwork]*storage.NetworkEntityInfo, len(h.pendingEntities))
	h.entitiesByID = make(map[string]*storage.NetworkEntityInfo, len(h.pendingEntities))
	var errList errorhelpers.ErrorList
	for _, entity := range h.pendingEntities {
		ipNet := pkgNet.IPNetworkFromCIDR(entity.GetExternalSource().GetCidr())
		if !ipNet.IsValid() {
			errList.AddStringf("%s (cidr=%s) ", entity.GetId(), entity.GetExternalSource().GetCidr())
			continue
		}
		h.entities[ipNet] = entity
		h.entitiesByID[entity.GetId()] = entity
	}
	// Release the raw list — the maps now own the data.
	h.pendingEntities = nil

	if err := errList.ToError(); err != nil {
		log.Errorf("could not process some external sources received from Central: %v", err)
	}
	log.Infof("Lazy-indexed %d external network entities on first lookup", len(h.entities))
}

func (h *handlerImpl) regenerateAndPushExternalSrcsToValueStream() {
	h.lock.Lock()
	defer h.lock.Unlock()

	defer h.updateSig.Reset()

	// Build the IP network list for collectors from raw entities.
	// This does NOT trigger the full CIDR index build — it just extracts
	// the IP/prefix pairs which is lightweight (~bytes, not ~16 MB maps).
	ipNetworkList := &sensor.IPNetworkList{}

	entities := h.pendingEntities
	if h.indexed {
		// If already indexed, iterate the map keys
		for ipNet := range h.entities {
			appendIPNet(ipNetworkList, ipNet)
		}
	} else {
		// Not yet indexed — parse CIDRs from raw entities without building maps
		for _, entity := range entities {
			ipNet := pkgNet.IPNetworkFromCIDR(entity.GetExternalSource().GetCidr())
			if ipNet.IsValid() {
				appendIPNet(ipNetworkList, ipNet)
			}
		}
	}

	normalizeNetworkList(ipNetworkList)

	if h.lastSeenList != nil && networkListsEqual(ipNetworkList, h.lastSeenList) {
		return
	}

	h.ipNetworkListProtoStream.Push(ipNetworkList)
	h.lastSeenList = ipNetworkList
}

func appendIPNet(list *sensor.IPNetworkList, ipNet pkgNet.IPNetwork) {
	if ipV4 := ipNet.IP().AsNetIP().To4(); ipV4 != nil {
		list.Ipv4Networks = append(list.Ipv4Networks, ipV4...)
		list.Ipv4Networks = append(list.Ipv4Networks, ipNet.PrefixLen())
	} else if ipV6 := ipNet.IP().AsNetIP().To16(); ipV6 != nil {
		list.Ipv6Networks = append(list.Ipv6Networks, ipV6...)
		list.Ipv6Networks = append(list.Ipv6Networks, ipNet.PrefixLen())
	}
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

	h.ensureIndexedNoLock()
	return h.entities[ipNet]
}

func (h *handlerImpl) LookupByID(id string) *storage.NetworkEntityInfo {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.ensureIndexedNoLock()
	return h.entitiesByID[id]
}
