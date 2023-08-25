package tests

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/indexer/mocks"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	prefix1 = []byte("cluster")
	prefix2 = []byte("namespace")
	prefix3 = []byte("deployment")
)

func TestIndexer(t *testing.T) {
	suite.Run(t, new(IndexerTestSuite))
}

type IndexerTestSuite struct {
	suite.Suite

	mockCtrl    *gomock.Controller
	mockWrapper *mocks.MockWrapper
}

func (suite *IndexerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockWrapper = mocks.NewMockWrapper(suite.mockCtrl)
}

func (suite *IndexerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *IndexerTestSuite) TestIndexer() {
	suite.mockWrapper.EXPECT().Wrap(dbhelper.GetBucketKey(prefix1, []byte("id1")), (proto.Message)(nil)).Return("id1", nil)
	suite.mockWrapper.EXPECT().Wrap(dbhelper.GetBucketKey(prefix2, []byte("id2")), (proto.Message)(nil)).Return("id2", nil)
	suite.mockWrapper.EXPECT().Wrap(dbhelper.GetBucketKey(prefix3, []byte("id3")), (proto.Message)(nil)).Return("id3", nil)

	registry := indexer.NewWrapperRegistry()
	registry.RegisterWrapper(prefix1, suite.mockWrapper)
	registry.RegisterWrapper(prefix2, suite.mockWrapper)
	registry.RegisterWrapper(prefix3, suite.mockWrapper)

	key, _ := registry.Wrap(dbhelper.GetBucketKey(prefix1, []byte("id1")), (proto.Message)(nil))
	suite.Equal("id1", key)
	key, _ = registry.Wrap(dbhelper.GetBucketKey(prefix2, []byte("id2")), (proto.Message)(nil))
	suite.Equal("id2", key)
	key, _ = registry.Wrap(dbhelper.GetBucketKey(prefix3, []byte("id3")), (proto.Message)(nil))
	suite.Equal("id3", key)
}
