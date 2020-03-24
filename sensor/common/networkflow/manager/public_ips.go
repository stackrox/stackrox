package manager

import (
	"encoding/binary"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities"
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

	publicIPListProtoStream *concurrency.ValueStream
}

func newPublicIPsManager() *publicIPsManager {
	return &publicIPsManager{
		publicIPsUpdateSig: concurrency.NewSignal(),
		publicIPs:          make(map[net.IPAddress]struct{}),
		publicIPDeletions:  make(map[net.IPAddress]time.Time),

		publicIPListProtoStream: concurrency.NewValueStream(nil),
	}
}

func (m *publicIPsManager) Run(ctx concurrency.Waitable, clusterEntities *clusterentities.Store) {
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

	m.publicIPs[ip] = struct{}{}
	delete(m.publicIPDeletions, ip) // undo a pending deletion, if any
	m.publicIPsUpdateSig.Signal()
}

func (m *publicIPsManager) OnRemoved(ip net.IPAddress) {
	now := time.Now()

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.publicIPDeletions[ip] = now
}

func (m *publicIPsManager) regenerateAndPushPublicIPsProto() {
	now := time.Now()

	m.mutex.Lock()
	defer m.mutex.Unlock()

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
		} else if netIP != nil && len(netIP) == 16 { // Genuine IPv6 address
			high := binary.BigEndian.Uint64(netIP[:8])
			low := binary.BigEndian.Uint64(netIP[8:])
			publicIPsList.Ipv6Addresses = append(publicIPsList.Ipv6Addresses, high, low)
		}
	}

	m.publicIPListProtoStream.Push(publicIPsList)
	m.publicIPsUpdateSig.Reset()
}

func (m *publicIPsManager) PublicIPsProtoStream() concurrency.ReadOnlyValueStream {
	return m.publicIPListProtoStream
}
