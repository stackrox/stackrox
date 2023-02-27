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
	mockInventoryScanOpts *CachingScannerOpts
	sleeper               mockSleeper
}

func TestComplianceCaching(t *testing.T) {
	suite.Run(t, new(TestComplianceCachingSuite))
}

// run before each test
func (s *TestComplianceCachingSuite) SetupTest() {
	s.sleeper = mockSleeper{callCount: 0}
	s.mockInventoryScanOpts = &CachingScannerOpts{
		InventoryCachePath:  fmt.Sprintf("%s/inventory-cache", s.T().TempDir()),
		BackoffFilePath:     fmt.Sprintf("%s/inventory-backoff", s.T().TempDir()),
		BackoffWaitCallback: s.sleeper.mockWaitCallback,
	}
}

func (s *TestComplianceCachingSuite) TestGetCurrentBackoff() {
	d, _ := time.ParseDuration("42s")
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), "1s")
	err := os.WriteFile(s.mockInventoryScanOpts.BackoffFilePath, []byte(d.String()), 0600)
	s.NoError(err)

	currentBackoff := readBackoff(s.mockInventoryScanOpts.BackoffFilePath)

	s.Equal(d, currentBackoff)
}

func (s *TestComplianceCachingSuite) TestGetCurrentBackoffReturnMaxOnError() {
	s.T().Setenv(env.NodeScanInitialBackoff.EnvVar(), "1s")
	s.T().Setenv(env.NodeScanMaxBackoff.EnvVar(), "42s")
	err := os.WriteFile(s.mockInventoryScanOpts.BackoffFilePath, []byte("notADuration"), 0600)
	s.NoError(err)

	currentBackoff := readBackoff(s.mockInventoryScanOpts.BackoffFilePath)

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
	e := os.WriteFile(s.mockInventoryScanOpts.BackoffFilePath, []byte(d.String()), 0600)
	s.NoError(e)
	c := NewCachingScanner(s.mockInventoryScanOpts.InventoryCachePath, s.mockInventoryScanOpts.BackoffFilePath, s.mockInventoryScanOpts.BackoffWaitCallback)

	_, err := c.Scan("testme")

	s.NoError(err)
	s.Equal(d, s.sleeper.receivedDuration)
	s.Equal(1, s.sleeper.callCount)
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryWithoutResultCache() {
	nodeName := "testme"
	c := NewCachingScanner(s.mockInventoryScanOpts.InventoryCachePath, s.mockInventoryScanOpts.BackoffFilePath, s.mockInventoryScanOpts.BackoffWaitCallback)

	actual, e := c.Scan(nodeName)

	s.NoError(e)
	s.Equal(nodeName, actual.GetNodeName())
}

func (s *TestComplianceCachingSuite) TestIsCachedInventoryValid() {
	s.T().Setenv(env.NodeScanCacheDuration.EnvVar(), "1m")

	tests := map[string]struct {
		created        time.Time
		expectedResult bool
	}{
		"cachedResult": {
			created:        time.Now(),
			expectedResult: true,
		},
		"cacheTooOld": {
			created:        time.Now().Add(-2 * time.Minute),
			expectedResult: false,
		},
		"cacheVeryOld": {
			created:        time.Unix(42, 0),
			expectedResult: false,
		},
	}

	for name, t := range tests {
		s.Run(name, func() {
			actual := isCachedInventoryValid(t.created)
			s.Equal(t.expectedResult, actual)
		})
	}
}
