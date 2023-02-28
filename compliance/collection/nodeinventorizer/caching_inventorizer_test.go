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
	s.sleeper = mockSleeper{callCount: 0}
	s.cs = *NewCachingScanner(
		fmt.Sprintf("%s/inventory-cache", s.T().TempDir()),
		fmt.Sprintf("%s/inventory-backoff", s.T().TempDir()),
		s.sleeper.mockWaitCallback,
	)
}

func (s *TestComplianceCachingSuite) TestValidateBackoff() {
	d, _ := time.ParseDuration("42s")
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), "1s")

	currentBackoff := validateBackoff(d)

	s.Equal(d, currentBackoff)
}

func (s *TestComplianceCachingSuite) TestValidateBackoffMaxOnBigValue() {
	d, _ := time.ParseDuration("100h")
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), "1s")
	s.T().Setenv(env.NodeScanMaxBackoff.EnvVar(), "42s")

	currentBackoff := validateBackoff(d)

	s.Equal(env.NodeScanMaxBackoff.DurationSetting(), currentBackoff)
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoff() {
	s.T().Setenv(env.NodeScanBackoffIncrement.EnvVar(), "24s")
	baseBackoff, _ := time.ParseDuration("10s")
	expectedBackoff, _ := time.ParseDuration("34s")

	newBackoff := calcNextBackoff(baseBackoff)

	s.Equal(expectedBackoff, newBackoff)
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoffUpperBoundary() {
	s.T().Setenv(env.NodeScanMaxBackoff.EnvVar(), "5s")
	s.T().Setenv(env.NodeScanBackoffIncrement.EnvVar(), "24s")
	baseBackoff, _ := time.ParseDuration("10s")
	expectedBackoff, _ := time.ParseDuration("5s")

	newBackoff := calcNextBackoff(baseBackoff)

	s.Equal(expectedBackoff, newBackoff)
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryHonorBackoff() {
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), "1s")
	s.T().Setenv(env.NodeScanBackoffIncrement.EnvVar(), "3s")

	d, _ := time.ParseDuration("8s")
	w := inventoryWrap{
		ValidUntil:      time.Time{},
		BackoffDuration: 8000000000, // 8 seconds
		Inventory:       nil,
	}
	jsonWrap, e := json.Marshal(&w)
	s.NoError(e)
	e = os.WriteFile(s.cs.InventoryCachePath, jsonWrap, 0600)
	s.NoError(e)
	c := NewCachingScanner(s.cs.InventoryCachePath, s.cs.BackoffFilePath, s.cs.BackoffWaitCallback)

	_, err := c.Scan("testme")

	s.NoError(err)
	s.Equal(d, s.sleeper.receivedDuration)
	s.Equal(1, s.sleeper.callCount)
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryWithoutResultCache() {
	nodeName := "testme"
	c := NewCachingScanner(s.cs.InventoryCachePath, s.cs.BackoffFilePath, s.cs.BackoffWaitCallback)

	actual, e := c.Scan(nodeName)

	s.NoError(e)
	s.Equal(nodeName, actual.GetNodeName())
}
