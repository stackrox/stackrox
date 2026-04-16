package fake

import (
	"fmt"
	"math/rand"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// pool is a thread-safe string pool using set.StringSet
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
	if val == "" {
		return "", false
	}
	return val, true
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
		p.Endpoints[val.GetContainerId()] = append(p.Endpoints[val.GetContainerId()], val)
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

// IP generation and pool manipulation functions

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
func generateExternalIPPool(pool *pool) {
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
		pool.add(ipString)
	}
}

func generateAndAddIPToPool(ipPool *pool) string {
	ip := generateIP()
	for !ipPool.add(ip) {
		ip = generateIP()
	}
	return ip
}

func getRandomOriginator(containerID string, pool *ProcessPool) *storage.NetworkProcessUniqueKey {
	var process *storage.ProcessSignal
	var percentMatchedProcess float32 = 0.5
	p := rand.Float32()
	if p < percentMatchedProcess {
		// There is a chance that the process has been filtered out or hasn't gotten to
		// the central-db for some other reason so this is not a guarantee that the
		// process is in the central-db
		process = pool.getRandomProcess(containerID)
	} else {
		process = getGoodProcess(containerID)
	}

	return getNetworkProcessUniqueKeyFromProcess(process)
}

func getNetworkProcessUniqueKeyFromProcess(process *storage.ProcessSignal) *storage.NetworkProcessUniqueKey {
	if process != nil {
		return &storage.NetworkProcessUniqueKey{
			ProcessName:         process.GetName(),
			ProcessExecFilePath: process.GetExecFilePath(),
			ProcessArgs:         process.GetArgs(),
		}
	}

	return nil
}
