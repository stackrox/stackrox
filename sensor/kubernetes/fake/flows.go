package fake

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/net"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/timestamp"
	"github.com/stackrox/stackrox/pkg/utils"
	"github.com/stackrox/stackrox/sensor/common/networkflow/manager"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ipPool        = newPool()
	containerPool = newPool()

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

		conns := make([]*sensor.NetworkConnection, 0, workload.BatchSize)
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
			utils.CrashOnError(err)

			conn := &sensor.NetworkConnection{
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
			conns = append(conns, conn)
		}
		hostConn, ok := w.getRandomHostConnection(ctx)
		if !ok {
			continue
		}
		err := hostConn.Process(&sensor.NetworkConnectionInfo{
			UpdatedConnections: conns,
			Time:               types.TimestampNow(),
		}, timestamp.Now(), 1)
		if err != nil {
			log.Error(err)
		}
	}
}
