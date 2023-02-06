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
	inventoryCacheSeconds = 60 * time.Second.Seconds()

	unix42, _ := timestamp.TimestampProto(time.Unix(42, 0))
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
