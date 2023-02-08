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

type TestComplianceCachingSuite struct {
	suite.Suite
}

func TestComplianceCaching(t *testing.T) {
	suite.Run(t, new(TestComplianceCachingSuite))
}

func (s *TestComplianceCachingSuite) TestCachingScanNode() {
	tmpDir := s.T().TempDir()
	inventoryCachePath = tmpDir

	result, e := runCachedScan("nodename", &nodeinventorizer.FakeNodeInventorizer{})
	s.FileExists(fmt.Sprintf("%s/last_scan", tmpDir))
	s.NotNil(result)
	s.NoError(e)
}

func (s *TestComplianceCachingSuite) TestCaching() {
	tmpDir := s.T().TempDir()
	// these two vars are set in the module and directly effect the tested code
	inventoryCachePath = tmpDir
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
			expectedNodeName: "newNode",
		},
		"cacheVeryOld": {
			inputInventory: &storage.NodeInventory{
				NodeName: "cachedNode",
				ScanTime: unix42,
			},
			expectedNodeName: "newNode",
		},
	}

	for caseName, testCase := range testCases {
		s.Run(caseName, func() {
			minv, _ := proto.Marshal(testCase.inputInventory)
			err := os.WriteFile(fmt.Sprintf("%s/last_scan", inventoryCachePath), minv, 0600)
			s.NoError(err)

			actual, e := scanNode("newNode", &nodeinventorizer.FakeNodeInventorizer{})
			s.NoError(e)
			s.Equal(testCase.expectedNodeName, actual.GetNode())
		})
	}
}

type mockSleeper struct {
	receivedDuration time.Duration
	callCount        int
}

func (ms *mockSleeper) Sleep(d time.Duration) {
	ms.receivedDuration = d
	ms.callCount++
}

func (s *TestComplianceCachingSuite) TestBackoffNoFile() {
	m := mockSleeper{callCount: 0}
	inventorySleeper = m.Sleep
	tmpDir := s.T().TempDir()
	inventoryCachePath = tmpDir

	_, _ = scanNodeBacked("testname", &nodeinventorizer.FakeNodeInventorizer{})

	// This file mustn't exist after a successful run
	_, err := os.Stat(fmt.Sprintf("%s/backoff", inventoryCachePath))
	s.ErrorIs(err, os.ErrNotExist)

	// No sleep should have been called
	s.Equal(0, m.callCount)
}

func (s *TestComplianceCachingSuite) TestBackoffWithFile() {
	m := mockSleeper{callCount: 0}
	inventorySleeper = m.Sleep
	tmpDir := s.T().TempDir()
	inventoryCachePath = tmpDir

	err := os.WriteFile(fmt.Sprintf("%s/backoff", inventoryCachePath), []byte(fmt.Sprintf("%d", 421)), 0600)
	s.NoError(err)

	_, _ = scanNodeBacked("testname", &nodeinventorizer.FakeNodeInventorizer{})

	// This file mustn't exist after a successful run
	_, err = os.Stat(fmt.Sprintf("%s/backoff", inventoryCachePath))
	s.ErrorIs(err, os.ErrNotExist)

	s.Equal(1, m.callCount)
	s.Equal(m.receivedDuration, time.Duration(421)*time.Second)
}

type mockInventoryErr struct {
}

func (mi *mockInventoryErr) Scan(nodeName string) (*storage.NodeInventory, error) {
	return nil, fmt.Errorf("This is a failure on node %s", nodeName)
}

func (s *TestComplianceCachingSuite) TestBackoffFailedRun() {
	tmpDir := s.T().TempDir()
	inventoryCachePath = tmpDir
	inventoryInitialBackoff = 4221

	_, _ = scanNodeBacked("testname", &mockInventoryErr{})

	// Even if a scan fails, it should still leave no state file behind
	_, err := os.Stat(fmt.Sprintf("%s/backoff", inventoryCachePath))
	s.ErrorIs(err, os.ErrNotExist)
}
