package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/compliance/collection/nodeinventorizer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/suite"
)

type mockSleeper struct {
	receivedDuration int64
	callCount        int
}

func (ms *mockSleeper) mockWaitCallback(d int64) int64 {
	ms.receivedDuration = d
	ms.callCount++
	return 4243
}

type TestComplianceCachingSuite struct {
	suite.Suite
	mockScanOpts *cachedScanOpts
	sleeper      mockSleeper
}

func TestComplianceCaching(t *testing.T) {
	suite.Run(t, new(TestComplianceCachingSuite))
}

// run before each test
func (s *TestComplianceCachingSuite) SetupTest() {
	s.sleeper = mockSleeper{callCount: 0}
	s.mockScanOpts = &cachedScanOpts{
		nodeName:            "testme",
		scanner:             &nodeinventorizer.FakeNodeInventorizer{},
		inventoryCachePath:  s.T().TempDir(),
		backoffWaitCallback: s.sleeper.mockWaitCallback,
	}
}

func (s *TestComplianceCachingSuite) TestCachingScanNode() {
	result, e := runCachedScan(s.mockScanOpts)
	s.FileExists(fmt.Sprintf("%s/last_scan", s.mockScanOpts.inventoryCachePath))
	s.NotNil(result)
	s.NoError(e)
}

func (s *TestComplianceCachingSuite) TestCaching() {
	// these two vars are set in the module and directly effect the tested code
	s.T().Setenv(env.NodeInventoryCacheDuration.EnvVar(), "1m")

	unix42, _ := timestamp.TimestampProto(time.Unix(42, 0))
	twoMinutesBefore, _ := timestamp.TimestampProto(time.Now().Add(-time.Minute * 2))
	testCases := map[string]struct {
		inputInventory   *storage.NodeInventory
		expectedNodeName string
	}{
		"cachedResult": {
			inputInventory: &storage.NodeInventory{
				NodeName: "cachedNode",
				ScanTime: timestamp.TimestampNow(),
			},
			expectedNodeName: "cachedNode",
		},
		"cacheTooOld": {
			inputInventory: &storage.NodeInventory{
				NodeName: "cachedNode",
				ScanTime: twoMinutesBefore,
			},
			expectedNodeName: "testme",
		},
		"cacheVeryOld": {
			inputInventory: &storage.NodeInventory{
				NodeName: "cachedNode",
				ScanTime: unix42,
			},
			expectedNodeName: "testme",
		},
	}

	for caseName, testCase := range testCases {
		s.Run(caseName, func() {
			minv, _ := proto.Marshal(testCase.inputInventory)
			err := os.WriteFile(fmt.Sprintf("%s/last_scan", s.mockScanOpts.inventoryCachePath), minv, 0600)
			s.NoError(err)

			actual, e := cachedScanNode(s.mockScanOpts)
			s.NoError(e)
			s.Equal(testCase.expectedNodeName, actual.GetNode())
		})
	}
}

func (s *TestComplianceCachingSuite) TestBackoffNoFile() {
	_, _ = scanNodeWithBackoff(s.mockScanOpts)

	// This file mustn't exist after a successful run
	_, err := os.Stat(fmt.Sprintf("%s/inventory-backoff", s.mockScanOpts.inventoryCachePath))
	s.ErrorIs(err, os.ErrNotExist)

	// No sleep should have been called
	s.Equal(0, s.sleeper.callCount)
}

func (s *TestComplianceCachingSuite) TestBackoffWithFile() {
	sleepTime := 32 * time.Second
	err := os.WriteFile(fmt.Sprintf("%s/inventory-backoff", s.mockScanOpts.inventoryCachePath), []byte(fmt.Sprintf("%d", int64(sleepTime))), 0600)
	s.NoError(err)

	_, _ = scanNodeWithBackoff(s.mockScanOpts)

	// This file mustn't exist after a successful run
	_, err = os.Stat(fmt.Sprintf("%s/inventory-backoff", s.mockScanOpts.inventoryCachePath))
	s.ErrorIs(err, os.ErrNotExist)

	s.Equal(1, s.sleeper.callCount)
	s.Equal(int64(sleepTime/time.Second), s.sleeper.receivedDuration)
}

func (s *TestComplianceCachingSuite) TestBackoffUpperBoundary() {
	s.T().Setenv(env.NodeInventoryMaxBackoff.EnvVar(), "10ms")
	s.T().Setenv(env.NodeInventoryBackoffIncrement.EnvVar(), "3s")

	nextInterval := waitAndIncreaseBackoff(4)

	s.Equal(int64(3), nextInterval)
}
