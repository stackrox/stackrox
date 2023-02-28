package nodeinventorizer

import (
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

func (s *TestComplianceCachingSuite) TestGetCurrentBackoff() {
	d, _ := time.ParseDuration("42s")
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), "1s")
	err := os.WriteFile(s.cs.BackoffFilePath, []byte(d.String()), 0600)
	s.NoError(err)

	currentBackoff := readBackoff(s.cs.BackoffFilePath)

	s.Equal(d, currentBackoff)
}

func (s *TestComplianceCachingSuite) TestGetCurrentBackoffReturnMaxOnError() {
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), "1s")
	s.T().Setenv(env.NodeScanMaxBackoff.EnvVar(), "42s")
	err := os.WriteFile(s.cs.BackoffFilePath, []byte("notADuration"), 0600)
	s.NoError(err)

	currentBackoff := readBackoff(s.cs.BackoffFilePath)

	s.Equal(env.NodeScanMaxBackoff.DurationSetting(), currentBackoff)
}

func (s *TestComplianceCachingSuite) TestGetCurrentBackoffReturnMaxOnBigValue() {
	d, _ := time.ParseDuration("100h")
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), "1s")
	s.T().Setenv(env.NodeScanMaxBackoff.EnvVar(), "42s")
	err := os.WriteFile(s.cs.BackoffFilePath, []byte(d.String()), 0600)
	s.NoError(err)

	currentBackoff := readBackoff(s.cs.BackoffFilePath)

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
	d, _ := time.ParseDuration("3m")
	e := os.WriteFile(s.cs.BackoffFilePath, []byte(d.String()), 0600)
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

func (s *TestComplianceCachingSuite) TestIsCachedInventoryValidSuccess() {
	s.T().Setenv(env.NodeScanCacheDuration.EnvVar(), "1m")
	t := time.Now()

	s.Equal(true, isCachedInventoryValid(t))
}

func (s *TestComplianceCachingSuite) TestIsCachedInventoryValidOutOfCache() {
	s.T().Setenv(env.NodeScanCacheDuration.EnvVar(), "1m")
	t := time.Now().Add(-5 * time.Minute) // 5 minutes ago, with cache duration at 1 minute

	s.Equal(false, isCachedInventoryValid(t))
}

func (s *TestComplianceCachingSuite) TestIsCachedInventoryValidFuture() {
	s.T().Setenv(env.NodeScanCacheDuration.EnvVar(), "1m")
	t := time.Now().Add(3 * time.Hour) // 3 hours in the future, which is considered invalid

	s.Equal(false, isCachedInventoryValid(t))
}
