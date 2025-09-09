package fake

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OriginatorCache stores originators by endpoint key (IP:port) to provide consistent
// process-to-endpoint mappings with configurable reuse probability.
// This improves test realism by simulating how processes typically bind to consistent endpoints.
//
// This cache implements probabilistic reuse to simulate:
//   - High probability case: Single process consistently binds to the endpoint; a new process can bind
//     to the same endpoint only after sending a close message.
//   - Low probability case: New process takes over the endpoint without sending a close message
//     (port is shared by multiple processes).
type OriginatorCache struct {
	cache map[string]*storage.NetworkProcessUniqueKey
	lock  sync.RWMutex
}

// NewOriginatorCache creates a new cache for storing endpoint-to-originator mappings.
func NewOriginatorCache() *OriginatorCache {
	return &OriginatorCache{
		cache: make(map[string]*storage.NetworkProcessUniqueKey),
	}
}

// GetOrSetOriginator retrieves a cached originator for the endpoint or generates a new one.
// The portSharingProbability parameter controls the probability of reusing an existing open endpoint by
// a different process versus generating a new endpoint for the process (range: 0.0-1.0).
//
// This caching mechanism improves test realism by ensuring that network endpoints
// typically close the endpoint before changing the originator (e.g., 95% default), while still allowing
// for on-the-fly process changes (e.g., 5% for multiple processes reusing the same port in parallel).
//
// Note that generating multiple 'open' endpoints with the same IP and Port but different originators
// without a 'close' message in between is not a realistic scenario and occurs very rarely in production
// (practically only when a process deliberately abuses the same IP and Port in parallel to a different process).
//
// Setting the portSharingProbability to an unrealistic value (anything higher than 0.05) would cause additional memory-pressure
// in Sensor enrichment pipeline because the deduping key for processes contains the IP, Port and the originator.
// Note that if Sensor sees an open endpoint for <container1, 1.1.1.1:80, nginx> and then another open endpoint for
// <container1, 1.1.1.1:80, apache2>, then Sensor will keep the nginx-entry forever, as there was no 'close' message in between.
//
// The probability logic is explicit and configurable for differ{ent testing scenarios.
func (oc *OriginatorCache) GetOrSetOriginator(endpointKey string, containerID string, openPortReuseProbability float64, processPool *ProcessPool) *storage.NetworkProcessUniqueKey {
	// Ensure that the probability is between 0.0 and 1.0.
	prob := math.Min(1.0, math.Max(0.0, openPortReuseProbability))
	if openPortReuseProbability < 0.0 || openPortReuseProbability > 1.0 {
		log.Warnf("Incorrect probability value %.2f for 'openPortReuseProbability', "+
			"rounding to: %.2f.", openPortReuseProbability, prob)
	}

	originator, exists := concurrency.WithRLock2(&oc.lock, func() (*storage.NetworkProcessUniqueKey, bool) {
		originator, exists := oc.cache[endpointKey]
		return originator, exists
	})

	if exists && rand.Float64() > prob {
		// Use the previously-known process for the same endpoint.
		return originator
	}
	// Generate a new originator with probability `openPortReuseProbability`.
	// This simulates when multiple processes listen on the same port
	newOriginator := getRandomOriginator(containerID, processPool)
	// We update the cache only on cache miss.
	// In case of openPortReuse, we keep the original originator in the cache, as this is a rare event.
	if !exists {
		concurrency.WithLock(&oc.lock, func() {
			oc.cache[endpointKey] = newOriginator
		})
	}

	return newOriginator
}

// Clear removes all cached originators. Used for cleanup between test runs.
func (oc *OriginatorCache) Clear() {
	concurrency.WithLock(&oc.lock, func() {
		oc.cache = make(map[string]*storage.NetworkProcessUniqueKey)
	})
}

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

// getRandomNetworkEndpoint generates a network endpoint with consistent originator caching.
// Uses probabilistic caching (configurable via workload.OpenPortReuseProbability) to simulate
// realistic process-to-endpoint binding behavior in containerized environments.
func (w *WorkloadManager) getRandomNetworkEndpoint(containerID string) (*sensor.NetworkEndpoint, bool) {
	ip, ok := w.ipPool.randomElem()
	if !ok {
		return nil, false
	}

	port := rand.Uint32() % 63556

	// Create endpoint key from IP and port for caching
	endpointKey := fmt.Sprintf("%s:%d", ip, port)

	// Get or set originator for this endpoint with configurable reuse probability
	originator := w.originatorCache.GetOrSetOriginator(endpointKey, containerID, w.workload.NetworkWorkload.OpenPortReuseProbability, w.processPool)

	networkEndpoint := &sensor.NetworkEndpoint{
		SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_IPV4,
		Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
		ListenAddress: &sensor.NetworkAddress{
			AddressData: net.ParseIP(ip).AsNetIP(),
			Port:        port,
		},
		ContainerId:    containerID,
		CloseTimestamp: nil,
		Originator:     originator,
	}

	return networkEndpoint, ok
}

func (w *WorkloadManager) getFakeNetworkConnectionInfo() *sensor.NetworkConnectionInfo {
	conns := make([]*sensor.NetworkConnection, 0, w.workload.NetworkWorkload.BatchSize)
	networkEndpoints := make([]*sensor.NetworkEndpoint, 0, w.workload.NetworkWorkload.BatchSize)
	for i := 0; i < w.workload.NetworkWorkload.BatchSize; i++ {
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
		if w.workload.NetworkWorkload.GenerateUnclosedEndpoints {
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
func (w *WorkloadManager) manageFlows(ctx context.Context) {
	if w.workload.NetworkWorkload.FlowInterval == 0 {
		return
	}
	// Pick a valid pod
	ticker := time.NewTicker(w.workload.NetworkWorkload.FlowInterval)
	defer ticker.Stop()

	generateExternalIPPool(w.externalIpPool)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		networkConnectionInfo := w.getFakeNetworkConnectionInfo()
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
