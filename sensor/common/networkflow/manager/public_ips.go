package manager

import (
	"encoding/binary"
	"sort"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	publicIPAfterglowPeriod = 1 * time.Minute // wait this long to propagate any deletion of a public IP address
)

// publicIPsManager tracks the addition and deletion of public IP addresses, for downstream consumption by sensors.
// In order to minimize the duration of the critical section and to avoid sending too many messages, as well as to avoid
// false negatives on the collector side because of the delayed transmission of network flows, deletions (a) are not
// propagated immediately but only after an "afterglow" period of 1 minute, and (b) only additions of IP addresses
// may trigger a state update (cause all pending deletions to finally take effect).
type publicIPsManager struct {
	mutex sync.Mutex

	publicIPsUpdateSig concurrency.Signal
	publicIPs          map[net.IPAddress]struct{}
	publicIPDeletions  map[net.IPAddress]time.Time

	publicIPListProtoStream *concurrency.ValueStream[*sensor.IPAddressList]

	lastSentIPAddrList *sensor.IPAddressList
}

func newPublicIPsManager() *publicIPsManager {
	return &publicIPsManager{
		publicIPsUpdateSig: concurrency.NewSignal(),
		publicIPs:          make(map[net.IPAddress]struct{}),
		publicIPDeletions:  make(map[net.IPAddress]time.Time),

		publicIPListProtoStream: concurrency.NewValueStream[*sensor.IPAddressList](nil),
	}
}

func (m *publicIPsManager) Run(ctx concurrency.Waitable, clusterEntities EntityStore) {
	clusterEntities.RegisterPublicIPsListener(m)
	defer clusterEntities.UnregisterPublicIPsListener(m)

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.publicIPsUpdateSig.Done():
			m.regenerateAndPushPublicIPsProto()
		}
	}
}

func (m *publicIPsManager) OnAdded(ip net.IPAddress) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Debugf("OnAdded: IP %s", ip.String())

	m.publicIPs[ip] = struct{}{}
	delete(m.publicIPDeletions, ip) // undo a pending deletion, if any
	m.publicIPsUpdateSig.Signal()
}

func (m *publicIPsManager) OnRemoved(ip net.IPAddress) {
	now := time.Now()

	m.mutex.Lock()
	defer m.mutex.Unlock()
	log.Debugf("OnRemoved: IP %s", ip.String())

	m.publicIPDeletions[ip] = now
}

func (m *publicIPsManager) regenerateAndPushPublicIPsProto() {
	now := time.Now()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	defer m.publicIPsUpdateSig.Reset()

	for deletedIP, deletionTS := range m.publicIPDeletions {
		if now.Sub(deletionTS) >= publicIPAfterglowPeriod {
			// Have the deletion take effect.
			delete(m.publicIPDeletions, deletedIP)
			delete(m.publicIPs, deletedIP)
		}
	}

	publicIPsList := &sensor.IPAddressList{}

	for publicIP := range m.publicIPs {
		netIP := publicIP.AsNetIP()

		if ipV4 := netIP.To4(); ipV4 != nil {
			publicIPsList.Ipv4Addresses = append(publicIPsList.Ipv4Addresses, binary.BigEndian.Uint32(ipV4))
		} else if len(netIP) == 16 { // Genuine IPv6 address
			high := binary.BigEndian.Uint64(netIP[:8])
			low := binary.BigEndian.Uint64(netIP[8:])
			publicIPsList.Ipv6Addresses = append(publicIPsList.Ipv6Addresses, high, low)
		}
	}

	normalizeIPsList(publicIPsList)

	if m.lastSentIPAddrList != nil && ipsListsEqual(publicIPsList, m.lastSentIPAddrList) {
		return
	}

	m.publicIPListProtoStream.Push(publicIPsList)
	m.lastSentIPAddrList = publicIPsList
}

func (m *publicIPsManager) PublicIPsProtoStream() concurrency.ReadOnlyValueStream[*sensor.IPAddressList] {
	return m.publicIPListProtoStream
}

type sortableIPv6Slice []uint64

func (s sortableIPv6Slice) Len() int {
	return len(s) / 2
}

func (s sortableIPv6Slice) Less(i, j int) bool {
	if s[2*i] != s[2*j] {
		return s[2*i] < s[2*j]
	}
	return s[2*i+1] < s[2*j+1]
}

func (s sortableIPv6Slice) Swap(i, j int) {
	s[2*i], s[2*j] = s[2*j], s[2*i]
	s[2*i+1], s[2*j+1] = s[2*j+1], s[2*i+1]
}

func normalizeIPsList(listProto *sensor.IPAddressList) {
	sliceutils.NaturalSort(listProto.Ipv4Addresses)
	sort.Sort(sortableIPv6Slice(listProto.Ipv6Addresses))
}

func ipsListsEqual(a, b *sensor.IPAddressList) bool {
	return sliceutils.Equal(a.GetIpv4Addresses(), b.GetIpv4Addresses()) &&
		sliceutils.Equal(a.GetIpv6Addresses(), b.GetIpv6Addresses())
}
