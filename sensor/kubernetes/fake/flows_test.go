package fake

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stretchr/testify/suite"
)

type flowsSuite struct {
	suite.Suite
	containerID string
	endpointKey string
	processPool *ProcessPool
}

func TestFlowsSuite(t *testing.T) {
	suite.Run(t, new(flowsSuite))
}

func (s *flowsSuite) SetupSuite() {
	// containerID must be 12 chars due to impl details of `getActiveProcesses`.
	s.containerID = "container123"
	s.endpointKey = "10.0.0.1:8080"
	s.processPool = newProcessPool()
	for _, process := range getActiveProcesses(s.containerID) {
		s.processPool.add(process)
	}
}

func (s *flowsSuite) TestGetRandomInternalExternalIP() {
	w := &WorkloadManager{
		endpointPool:   newEndpointPool(),
		processPool:    s.processPool,
		ipPool:         newPool(),
		externalIpPool: newPool(),
		containerPool:  newPool(),
	}

	_, _, ok := w.getRandomSrcDst()
	s.False(ok)

	for range 1000 {
		generateAndAddIPToPool(w.ipPool)
	}

	generateExternalIPPool(w.externalIpPool)

	for range 1000 {
		ip, internal, ok := w.getRandomInternalExternalIP()
		s.True(ok)
		s.Equal(internal, !net.ParseIP(ip).IsPublic())
	}

	for range 1000 {
		src, dst, ok := w.getRandomSrcDst()
		// At least one has to be internal
		s.True(!net.ParseIP(src).IsPublic() || !net.ParseIP(dst).IsPublic())
		s.True(ok)
	}
}

func (s *flowsSuite) TestOriginatorCache_BasicCaching() {
	cache := NewOriginatorCache()

	// Manually seed the cache with a known originator
	seedOriginator := &storage.NetworkProcessUniqueKey{
		ProcessName:         "cached-process",
		ProcessExecFilePath: "/usr/bin/cached-process",
		ProcessArgs:         "cached args",
	}
	cache.cache[s.endpointKey] = seedOriginator

	// With 0.0 probability, it should return the cached originator
	for range 10 {
		originator := cache.GetOrSetOriginator(s.endpointKey, s.containerID, 0.0, s.processPool)
		s.NotNil(originator)
		s.Equal("cached-process", originator.GetProcessName())
		s.Equal("/usr/bin/cached-process", originator.GetProcessExecFilePath())
		s.Equal("cached args", originator.GetProcessArgs())
	}
}

func (s *flowsSuite) TestOriginatorCache_ProbabilityCaching() {
	cache := NewOriginatorCache()

	seedOriginator := &storage.NetworkProcessUniqueKey{
		ProcessName:         "cached-process",
		ProcessExecFilePath: "/usr/bin/cached-process",
		ProcessArgs:         "cached args",
	}
	cache.cache[s.endpointKey] = seedOriginator
	numCacheMisses := 0

	s.T().Logf("Testing with endpoint key: %s, number of processes in pool: %d", s.endpointKey, len(s.processPool.Processes[s.containerID]))

	for range 100_000 {
		originator := cache.GetOrSetOriginator(s.endpointKey, s.containerID, 0.05, s.processPool)
		s.Require().NotNil(originator)
		if originator != seedOriginator {
			numCacheMisses++
		}
	}
	s.Len(cache.cache, 1, "Cache should have 1 entry")
	got := float64(numCacheMisses) / 100_000
	s.T().Logf("Observed probability of reusing port: %f", got)
	s.InDelta(0.05, got, 0.02, "Cache miss rate should be close to 0.05 (range <0.03,0.07>)")
}
