package nodeinventorizer

import (
	"fmt"
	"os"
	"testing"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
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
	mockInventoryScanOpts *InventoryScanOpts
	sleeper               mockSleeper
}

func TestComplianceCaching(t *testing.T) {
	suite.Run(t, new(TestComplianceCachingSuite))
}

// run before each test
func (s *TestComplianceCachingSuite) SetupTest() {
	s.sleeper = mockSleeper{callCount: 0}
	s.mockInventoryScanOpts = &InventoryScanOpts{
		NodeName:            "testme",
		Scanner:             &FakeNodeInventorizer{},
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

	currentBackoff, err := getCurrentBackoff(s.mockInventoryScanOpts.BackoffFilePath)

	s.NoError(err)
	s.Equal(d, *currentBackoff)
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

	_, err := TriggerNodeInventory(s.mockInventoryScanOpts)

	s.NoError(err)
	s.Equal(d, s.sleeper.receivedDuration)
	s.Equal(1, s.sleeper.callCount)
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryWithoutResultCache() {
	actual, e := TriggerNodeInventory(s.mockInventoryScanOpts)
	s.NoError(e)
	s.Equal(s.mockInventoryScanOpts.NodeName, actual.GetNode())
}

func (s *TestComplianceCachingSuite) TestIsCachedInventoryValid() {
	s.T().Setenv(env.NodeScanCacheDuration.EnvVar(), "1m")

	unix42, _ := timestamp.TimestampProto(time.Unix(42, 0))
	twoMinutesBefore, _ := timestamp.TimestampProto(time.Now().Add(-time.Minute * 2))
	testCases := map[string]struct {
		inputInventory *storage.NodeInventory
		expectedResult bool
	}{
		"cachedResult": {
			inputInventory: &storage.NodeInventory{
				NodeName: "cachedNode",
				ScanTime: timestamp.TimestampNow(),
			},
			expectedResult: true,
		},
		"cacheTooOld": {
			inputInventory: &storage.NodeInventory{
				NodeName: "cachedNode",
				ScanTime: twoMinutesBefore,
			},
			expectedResult: false,
		},
		"cacheVeryOld": {
			inputInventory: &storage.NodeInventory{
				NodeName: "cachedNode",
				ScanTime: unix42,
			},
			expectedResult: false,
		},
	}

	for caseName, testCase := range testCases {
		s.Run(caseName, func() {
			actual := isCachedInventoryValid(testCase.inputInventory)
			s.Equal(testCase.expectedResult, actual)
		})
	}
}
