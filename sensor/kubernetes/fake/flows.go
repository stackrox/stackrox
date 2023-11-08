package fake

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ipPool        = newPool()
	containerPool = newPool()
	endpointPool  = newEndpointPool()

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

func (p *EndpointPool) getRandomEndpoint(containerID string) *sensor.NetworkEndpoint {
	p.lock.Lock()
	defer p.lock.Unlock()

	size := len(p.Endpoints[containerID])
	if size > 0 {
		randIdx := rand.Intn(size)
		return p.Endpoints[containerID][randIdx]
	}

	return nil
}

func generateIP() string {
	return fmt.Sprintf("10.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256))
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

func getNetworkEndpointFromConnectionAndOriginator(conn *sensor.NetworkConnection, originator *storage.NetworkProcessUniqueKey) *sensor.NetworkEndpoint {
	return &sensor.NetworkEndpoint{
		SocketFamily:   conn.SocketFamily,
		Protocol:       conn.Protocol,
		ListenAddress:  conn.LocalAddress,
		ContainerId:    conn.ContainerId,
		CloseTimestamp: nil,
		Originator:     originator,
	}
}

func makeNetworkConnection(src string, dst string, containerID string, closeTS *types.Timestamp) *sensor.NetworkConnection {
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

func (w *WorkloadManager) getFakeNetworkConnectionInfo(workload NetworkWorkload) *sensor.NetworkConnectionInfo {
	conns := make([]*sensor.NetworkConnection, 0, workload.BatchSize)
	networkEndpoints := make([]*sensor.NetworkEndpoint, 0, workload.BatchSize)
	for i := 0; i < workload.BatchSize; i++ {
		src, ok := ipPool.randomElem()
		if !ok {
			log.Error("found no IPs in pool")
			continue
		}
		dst, ok := ipPool.randomElem()
		if !ok {
			log.Error("found no IPs in pool")
			continue
		}

		containerID, ok := containerPool.randomElem()
		if !ok {
			log.Error("found no containers in pool")
			continue
		}

		closeTS, err := types.TimestampProto(time.Now().Add(-5 * time.Second))
		if err != nil {
			log.Errorf("Unable to set closeTS %+v", err)
		}

		conn := makeNetworkConnection(src, dst, containerID, closeTS)

		originator := getRandomOriginator(containerID)

		networkEndpoint := getNetworkEndpointFromConnectionAndOriginator(conn, originator)

		conns = append(conns, conn)
		if endpointPool.Size < endpointPool.Capacity {
			endpointPool.add(networkEndpoint)
		}
		networkEndpoints = append(networkEndpoints, networkEndpoint)
	}

	for _, endpoint := range endpointPool.EndpointsToBeClosed {
		networkEndpoint := endpoint
		closeTS, err := types.TimestampProto(time.Now().Add(-5 * time.Second))
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
		Time:               types.TimestampNow(),
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
