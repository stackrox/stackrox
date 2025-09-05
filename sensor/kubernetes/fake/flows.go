package fake

import (
	"context"
	"math/rand"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (w *WorkloadManager) getRandomHostConnection(ctx context.Context) (manager.HostNetworkInfo, bool) {
	// Return false if the network manager hasn't been initialized yet
	if !w.servicesInitialized.IsDone() {
		return nil, false
	}
	if len(w.registeredHostConnections) == 0 {
		// Initialize the host connections
		nodeResp, err := w.fakeClient.CoreV1().Nodes().List(ctx, v1.ListOptions{})
		if err != nil {
			log.Errorf("error listing nodes: %v", err)
			return nil, false
		}
		for _, node := range nodeResp.Items {
			info, _ := w.networkManager.RegisterCollector(node.Name)
			w.registeredHostConnections = append(w.registeredHostConnections, info)
		}
	}
	return w.registeredHostConnections[rand.Intn(len(w.registeredHostConnections))], true
}

func makeNetworkConnection(src string, dst string, containerID string, closeTimestamp time.Time) *sensor.NetworkConnection {
	closeTS, err := protocompat.ConvertTimeToTimestampOrError(closeTimestamp)
	if err != nil {
		log.Errorf("Unable to set closeTS %+v", err)
	}

	return &sensor.NetworkConnection{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		LocalAddress: &sensor.NetworkAddress{
			AddressData: net.ParseIP(src).AsNetIP(),
			Port:        rand.Uint32() % 63556,
		},
		RemoteAddress: &sensor.NetworkAddress{
			AddressData: net.ParseIP(dst).AsNetIP(),
			Port:        rand.Uint32() % 63556,
		},
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		Role:           sensor.ClientServerRole_ROLE_CLIENT,
		ContainerId:    containerID,
		CloseTimestamp: closeTS,
	}
}

// Randomly decide to get an interal or external IP, with an 80% chance of the IP
// being internal and 20% of being external. If the IP is external randomly decide
// to pick it from a pool of external IPs or a new external IP, with a 50/50 chance
// of being from the pool or a newly generated IP address. We want to have cases
// where multiple different entities connect to the same external IP, but we also
// want a large number of unique external IPs.
func (w *WorkloadManager) getRandomInternalExternalIP() (string, bool, bool) {
	ip := ""
	var ok bool

	internal := rand.Intn(100) < 80
	if internal {
		ip, ok = w.ipPool.randomElem()
	} else {
		if rand.Intn(100) < 50 {
			ip, ok = w.externalIpPool.randomElem()
		} else {
			ip = generateExternalIP()
			ok = true
		}
	}

	if !ok {
		log.Errorf("Found no IPs in the %s pool", map[bool]string{true: "internal", false: "external"}[internal])
	}

	return ip, internal, ok
}

func (w *WorkloadManager) getRandomSrcDst() (string, string, bool) {
	src, internal, ok := w.getRandomInternalExternalIP()
	if !ok {
		return "", "", false
	}
	var dst string
	// If the src is internal, the dst can be internal or external, but
	// if the src is external, the dst must be internal.
	if internal {
		dst, _, ok = w.getRandomInternalExternalIP()
	} else {
		dst, ok = w.ipPool.randomElem()
		if !ok {
			log.Error("Found no IPs in the internal pool")
		}
	}

	return src, dst, ok
}

func (w *WorkloadManager) getRandomNetworkEndpoint(containerID string) (*sensor.NetworkEndpoint, bool) {
	originator := getRandomOriginator(containerID, w.processPool)

	ip, ok := w.ipPool.randomElem()
	if !ok {
		return nil, false
	}

	networkEndpoint := &sensor.NetworkEndpoint{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
		ListenAddress: &sensor.NetworkAddress{
			AddressData: net.ParseIP(ip).AsNetIP(),
			Port:        rand.Uint32() % 63556,
		},
		ContainerId:    containerID,
		CloseTimestamp: nil,
		Originator:     originator,
	}

	return networkEndpoint, ok
}

func (w *WorkloadManager) getFakeNetworkConnectionInfo(workload NetworkWorkload) *sensor.NetworkConnectionInfo {
	conns := make([]*sensor.NetworkConnection, 0, workload.BatchSize)
	networkEndpoints := make([]*sensor.NetworkEndpoint, 0, workload.BatchSize)
	for i := 0; i < workload.BatchSize; i++ {
		src, dst, ok := w.getRandomSrcDst()
		if !ok {
			continue
		}

		containerID, ok := w.containerPool.randomElem()
		if !ok {
			log.Error("Found no containers in pool")
			continue
		}

		conn := makeNetworkConnection(src, dst, containerID, time.Now().Add(-5*time.Second))
		networkEndpoint, ok := w.getRandomNetworkEndpoint(containerID)
		if !ok {
			log.Error("Found no IPs in the internal pool")
			continue
		}

		conns = append(conns, conn)
		if w.endpointPool.Size < w.endpointPool.Capacity {
			w.endpointPool.add(networkEndpoint)
		}
		if workload.GenerateUnclosedEndpoints {
			// These endpoints will not be closed - i.e., CloseTimestamp will be always nil.
			networkEndpoints = append(networkEndpoints, networkEndpoint)
		}
	}

	for _, endpoint := range w.endpointPool.EndpointsToBeClosed {
		networkEndpoint := endpoint
		closeTS, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Add(-5 * time.Second))
		if err != nil {
			log.Errorf("Unable to set CloseTimestamp for endpoint %+v", err)
		} else {
			networkEndpoint.CloseTimestamp = closeTS
			networkEndpoints = append(networkEndpoints, networkEndpoint)
		}
	}

	w.endpointPool.clearEndpointsToBeClosed()

	return &sensor.NetworkConnectionInfo{
		UpdatedConnections: conns,
		UpdatedEndpoints:   networkEndpoints,
		Time:               protocompat.TimestampNow(),
	}
}

// manageFlows should be called via `go manageFlows` as it will run forever
func (w *WorkloadManager) manageFlows(ctx context.Context, workload NetworkWorkload) {
	if workload.FlowInterval == 0 {
		return
	}
	// Pick a valid pod
	ticker := time.NewTicker(workload.FlowInterval)
	defer ticker.Stop()

	generateExternalIPPool(w.externalIpPool)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		networkConnectionInfo := w.getFakeNetworkConnectionInfo(workload)
		hostConn, ok := w.getRandomHostConnection(ctx)
		if !ok {
			continue
		}

		err := hostConn.Process(networkConnectionInfo, timestamp.Now(), 1)
		if err != nil {
			log.Error(err)
		}
	}
}
