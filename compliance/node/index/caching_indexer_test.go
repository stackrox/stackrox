package index

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/utils"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/suite"
)

type mockNodeIndexer struct{}

func (m mockNodeIndexer) IndexNode(_ context.Context) (*v4.IndexReport, error) {
	ir := &v4.IndexReport{}
	ir.SetHashId("mockIndexerTestReport")
	return ir, nil
}

func (m mockNodeIndexer) GetIntervals() *utils.NodeScanIntervals {
	return utils.NewNodeScanInterval(1*time.Minute, 0.1, 2*time.Minute)
}

type TestCachingIndexerSuite struct {
	suite.Suite
}

func TestCachingIndexer(t *testing.T) {
	suite.Run(t, new(TestCachingIndexerSuite))
}

func (s *TestCachingIndexerSuite) writeWrap(wrap *reportWrap, path string) {
	if wrap == nil {
		return
	}
	jsonWrap, err := json.Marshal(&wrap)
	s.NoError(err)
	err = os.WriteFile(path, jsonWrap, 0600)
	s.NoError(err)
}

func (s *TestCachingIndexerSuite) TestCachedIndexNode() {
	cases := map[string]struct {
		cachedWrap    *reportWrap
		expectedIndex *v4.IndexReport
	}{
		"cached index should be returned on success": {
			cachedWrap: &reportWrap{
				CacheValidUntil: time.Now().Add(1 * time.Minute),
				Report:          v4.IndexReport_builder{HashId: "cached"}.Build(),
			},
			expectedIndex: v4.IndexReport_builder{HashId: "cached"}.Build(),
		},
		"fresh index is generated if cached report is too old": {
			cachedWrap: &reportWrap{
				CacheValidUntil: time.Now().Add(-10 * time.Hour),
				Report:          v4.IndexReport_builder{HashId: "cached"}.Build(),
			},
			expectedIndex: v4.IndexReport_builder{
				HashId: "mockIndexerTestReport",
			}.Build(),
		},
		"fresh index is generated if cached report is broken": {
			cachedWrap: &reportWrap{
				CacheValidUntil: time.Now().Add(1 * time.Minute),
				Report:          nil,
			},
			expectedIndex: v4.IndexReport_builder{
				HashId: "mockIndexerTestReport",
			}.Build(),
		},
		"fresh index is generated if no cache exists": {
			cachedWrap: nil,
			expectedIndex: v4.IndexReport_builder{
				HashId: "mockIndexerTestReport",
			}.Build(),
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			path := fmt.Sprintf("%s/index-cache", s.T().TempDir())
			s.writeWrap(c.cachedWrap, path)
			mockIndexer := mockNodeIndexer{}
			ci := cachingNodeIndexer{indexer: mockIndexer, cachePath: path}

			actual, err := ci.IndexNode(context.TODO())

			s.NoError(err)
			protoassert.Equal(s.T(), c.expectedIndex, actual)
		})
	}
}

func (s *TestCachingIndexerSuite) TestCachedIndexDurationSetting() {
	path := fmt.Sprintf("%s/index-cache", s.T().TempDir())
	mockIndexer := mockNodeIndexer{}
	ci := cachingNodeIndexer{indexer: mockIndexer, cacheDuration: 42 * time.Minute, cachePath: path}
	s.NoFileExists(path)

	actual, err := ci.IndexNode(context.TODO())
	s.NoError(err)

	// Verify that the cache was written and contains the correct information
	s.FileExists(path)
	wrap, err := loadCachedWrap(path)
	s.NoError(err)
	protoassert.Equal(s.T(), actual, wrap.Report)
	s.WithinDuration(time.Now().Add(42*time.Minute), wrap.CacheValidUntil, 10*time.Second)
}

func (s *TestCachingIndexerSuite) TestCachedIndexNodeIllegalJSON() {
	path := fmt.Sprintf("%s/index-cache", s.T().TempDir())
	content := []byte("{some_json:value}")
	err := os.WriteFile(path, content, 0600)
	s.NoError(err)
	mockIndexer := mockNodeIndexer{}
	ci := cachingNodeIndexer{indexer: mockIndexer, cachePath: path}

	actual, err := ci.IndexNode(context.TODO())

	s.NoError(err)
	expected := &v4.IndexReport{}
	expected.SetHashId("mockIndexerTestReport")
	protoassert.Equal(s.T(), expected, actual)
}

func (s *TestCachingIndexerSuite) TestIllegalPathReturnsReport() {
	path := "/definitelynotexisting/index-report"
	mockIndexer := mockNodeIndexer{}
	ci := cachingNodeIndexer{indexer: mockIndexer, cachePath: path}

	actual, err := ci.IndexNode(context.TODO())

	s.NoError(err)
	s.NoFileExists(path)
	expected := &v4.IndexReport{}
	expected.SetHashId("mockIndexerTestReport")
	protoassert.Equal(s.T(), expected, actual)
}

func (s *TestCachingIndexerSuite) TestGetIntervals() {
	mockIndexer := mockNodeIndexer{}
	ci := cachingNodeIndexer{indexer: mockIndexer}

	actual := ci.GetIntervals()

	s.NotNil(actual)
}

func (s *TestCachingIndexerSuite) TestFailingNodeIndexer() {
	path := fmt.Sprintf("%s/index-cache", s.T().TempDir())
	fi := failingIndexer{}
	ci := cachingNodeIndexer{indexer: fi, cachePath: path}

	actual, err := ci.IndexNode(context.TODO())

	s.Nil(actual)
	s.ErrorContains(err, "Mock Failure")
}

type failingIndexer struct{}

func (m failingIndexer) IndexNode(_ context.Context) (*v4.IndexReport, error) {
	return nil, errors.New("Mock Failure")
}

func (m failingIndexer) GetIntervals() *utils.NodeScanIntervals {
	return nil
}
