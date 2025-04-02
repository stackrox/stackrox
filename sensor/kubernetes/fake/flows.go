package fake

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ipPool         = newPool()
	externalIpPool = newPool()
	containerPool  = newPool()
	endpointPool   = newEndpointPool()

	registeredHostConnections []manager.HostNetworkInfo
)

type pool struct {
	pool set.StringSet
	lock sync.RWMutex
}

func newPool() *pool {
	return &pool{
		pool: set.NewStringSet(),
	}
}

func (p *pool) add(val string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	if added := p.pool.Add(val); !added {
		return false
	}
	return true
}

func (p *pool) remove(val string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.pool.Remove(val)
	processPool.remove(val)
	endpointPool.remove(val)
}

func (p *pool) randomElem() (string, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	val := p.pool.GetArbitraryElem()
	return val, val != ""
}

func (p *pool) mustGetRandomElem() string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	val := p.pool.GetArbitraryElem()
	if val == "" {
		panic("not expecting an empty pool")
	}
	return val
}

// EndpointPool stores endpoints by containerID using a map
type EndpointPool struct {
	Endpoints           map[string][]*sensor.NetworkEndpoint
	EndpointsToBeClosed []*sensor.NetworkEndpoint
	Capacity            int
	Size                int
	lock                sync.RWMutex
}

func newEndpointPool() *EndpointPool {
	return &EndpointPool{
		Endpoints:           make(map[string][]*sensor.NetworkEndpoint),
		EndpointsToBeClosed: make([]*sensor.NetworkEndpoint, 0),
		Capacity:            10000,
		Size:                0,
	}
}

func (p *EndpointPool) add(val *sensor.NetworkEndpoint) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.Size < p.Capacity {
		p.Endpoints[val.ContainerId] = append(p.Endpoints[val.ContainerId], val)
		p.Size++
	}
}

func (p *EndpointPool) remove(containerID string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.EndpointsToBeClosed = append(p.EndpointsToBeClosed, p.Endpoints[containerID]...)
	p.Size -= len(p.Endpoints[containerID])
	delete(p.Endpoints, containerID)
}

func (p *EndpointPool) clearEndpointsToBeClosed() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.EndpointsToBeClosed = []*sensor.NetworkEndpoint{}
}

func generateIP() string {
	return fmt.Sprintf("10.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256))
}

// Generate IP addresses from 11.0.0.0 to 99.255.255.255 which are all public
func generateExternalIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(89)+11, rand.Intn(256), rand.Intn(256), rand.Intn(256))
}

// We want to reuse some external IPs, so we test the cases where multiple
// entities connect to the same external IP, but we also want many external IPs
// that are only used once.
func generateExternalIPPool() {
	ip := []int{11, 0, 0, 0}
	for range 1000 {
		for j := 3; j >= 0; j-- {
			ip[j]++
			if ip[j] > 255 {
				ip[j] = 0
			} else {
				break
			}
		}
		ipString := fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
		externalIpPool.add(ipString)
	}
}

func generateAndAddIPToPool() string {
	ip := generateIP()
	for !ipPool.add(ip) {
		ip = generateIP()
	}
	return ip
}

func (w *WorkloadManager) getRandomHostConnection(ctx context.Context) (manager.HostNetworkInfo, bool) {
	// Return false if the network manager hasn't been initialized yet
	if !w.servicesInitialized.IsDone() {
		return nil, false
	}
	if len(registeredHostConnections) == 0 {
		// Initialize the host connections
		nodeResp, err := w.fakeClient.CoreV1().Nodes().List(ctx, v1.ListOptions{})
		if err != nil {
			log.Errorf("error listing nodes: %v", err)
			return nil, false
		}
		for _, node := range nodeResp.Items {
			info, _ := w.networkManager.RegisterCollector(node.Name)
			registeredHostConnections = append(registeredHostConnections, info)
		}
	}
	return registeredHostConnections[rand.Intn(len(registeredHostConnections))], true
}

func getNetworkProcessUniqueKeyFromProcess(process *storage.ProcessSignal) *storage.NetworkProcessUniqueKey {
	if process != nil {
		return &storage.NetworkProcessUniqueKey{
			ProcessName:         process.Name,
			ProcessExecFilePath: process.ExecFilePath,
			ProcessArgs:         process.Args,
		}
	}

	return nil
}

func getRandomOriginator(containerID string) *storage.NetworkProcessUniqueKey {
	var process *storage.ProcessSignal
	var percentMatchedProcess float32 = 0.5
	p := rand.Float32()
	if p < percentMatchedProcess {
		// There is a chance that the process has been filtered out or hasn't gotten to
		// the central-db for some other reason so this is not a guarantee that the
		// process is in the central-db
		process = processPool.getRandomProcess(containerID)
	} else {
		process = getGoodProcess(containerID)
	}

	return getNetworkProcessUniqueKeyFromProcess(process)
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
		ip, ok = ipPool.randomElem()
	} else {
		if rand.Intn(100) < 50 {
			ip, ok = externalIpPool.randomElem()
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
		dst, ok = ipPool.randomElem()
		if !ok {
			log.Error("Found no IPs in the internal pool")
		}
	}

	return src, dst, ok
}

func (w *WorkloadManager) getRandomNetworkEndpoint(containerID string) (*sensor.NetworkEndpoint, bool) {
	originator := getRandomOriginator(containerID)

	ip, ok := ipPool.randomElem()
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

		containerID, ok := containerPool.randomElem()
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
		if endpointPool.Size < endpointPool.Capacity {
			endpointPool.add(networkEndpoint)
		}
		networkEndpoints = append(networkEndpoints, networkEndpoint)
	}

	for _, endpoint := range endpointPool.EndpointsToBeClosed {
		networkEndpoint := endpoint
		closeTS, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Add(-5 * time.Second))
		if err != nil {
			log.Errorf("Unable to set CloseTimestamp for endpoint %+v", err)
		} else {
			networkEndpoint.CloseTimestamp = closeTS
			networkEndpoints = append(networkEndpoints, networkEndpoint)
		}
	}

	endpointPool.clearEndpointsToBeClosed()

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

	generateExternalIPPool()

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
