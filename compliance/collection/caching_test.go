package main

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/compliance/collection/nodeinventorizer"
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
