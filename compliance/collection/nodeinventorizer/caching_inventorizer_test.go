package nodeinventorizer

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/suite"
)

const (
	nodeScanInitialBackoff = "1s"
	nodeScanMaxBackoff     = "5s"
)

type mockSleeper struct {
	receivedDuration time.Duration
	callCount        int
}

func (ms *mockSleeper) mockWaitCallback(d time.Duration) {
	ms.receivedDuration = d
	ms.callCount++
}

type TestComplianceCachingSuite struct {
	suite.Suite
	cs      CachingScanner
	sleeper mockSleeper
}

func TestComplianceCaching(t *testing.T) {
	suite.Run(t, new(TestComplianceCachingSuite))
}

// run before each test
func (s *TestComplianceCachingSuite) SetupTest() {
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), nodeScanInitialBackoff)
	s.T().Setenv(env.NodeScanMaxBackoff.EnvVar(), nodeScanMaxBackoff)

	s.sleeper = mockSleeper{callCount: 0}
	s.cs = *NewCachingScanner(
		fmt.Sprintf("%s/inventory-cache", s.T().TempDir()),
		s.sleeper.mockWaitCallback,
	)
}

func (s *TestComplianceCachingSuite) TestValidateBackoff() {
	d, _ := time.ParseDuration("2s")

	currentBackoff := s.cs.validateBackoff(d)

	// On successful test, the duration must not be overwritten
	s.Equal(d, currentBackoff)
}

func (s *TestComplianceCachingSuite) TestValidateBackoffMaxOnBigValue() {
	d, _ := time.ParseDuration("100h")
	maxBackoff, _ := time.ParseDuration(nodeScanMaxBackoff)

	currentBackoff := s.cs.validateBackoff(d)

	s.Equal(maxBackoff, currentBackoff)
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoff() {
	baseBackoff, _ := time.ParseDuration("1s")
	expectedBackoff := baseBackoff * backoffMultiplier

	newBackoff := s.cs.calcNextBackoff(baseBackoff)

	s.Equal(expectedBackoff, newBackoff)
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoffUpperBoundary() {
	baseBackoff, _ := time.ParseDuration("10s")
	expectedBackoff, _ := time.ParseDuration(nodeScanMaxBackoff)

	newBackoff := s.cs.calcNextBackoff(baseBackoff)

	s.Equal(expectedBackoff, newBackoff)
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryHonorBackoff() {
	d, _ := time.ParseDuration("4s")
	w := inventoryWrap{
		ValidUntil:      time.Time{},
		BackoffDuration: 4000000000, // 4 seconds
		Inventory:       nil,
	}
	jsonWrap, e := json.Marshal(&w)
	s.NoError(e)
	e = os.WriteFile(s.cs.inventoryCachePath, jsonWrap, 0600)
	s.NoError(e)
	c := NewCachingScanner(s.cs.inventoryCachePath, s.cs.backoffWaitCallback)

	_, err := c.Scan("testme")

	s.NoError(err)
	s.Equal(d, s.sleeper.receivedDuration)
	s.Equal(1, s.sleeper.callCount)
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryWithoutResultCache() {
	nodeName := "testme"
	c := NewCachingScanner(s.cs.inventoryCachePath, s.cs.backoffWaitCallback)

	actual, e := c.Scan(nodeName)

	s.NoError(e)
	s.Equal(nodeName, actual.GetNodeName())
}
