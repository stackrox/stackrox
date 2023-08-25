package manager

import (
	"testing"

	"github.com/stackrox/rox/central/externalbackups/manager/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestWatchHandler(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(WatchHandlerTestSuite))
}

type WatchHandlerTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	mgr      *mocks.MockManager
}

func (s *WatchHandlerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mgr = mocks.NewMockManager(s.mockCtrl)
}

func (s *WatchHandlerTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *WatchHandlerTestSuite) TestOnStableUpdate() {
	watch := &watchHandler{
		mgr:  s.mgr,
		id:   "wooo",
		file: "test_data",
	}
	testGcsConfig := &storage.ExternalBackup{
		Name: "test",
		Type: "gcs",
	}

	s.mgr.EXPECT().Upsert(gomock.Any(), testGcsConfig).Return(nil)
	watch.OnStableUpdate(testGcsConfig, nil)
	s.NotEmpty(testGcsConfig.GetId())
}

func (s *WatchHandlerTestSuite) TestOnChange() {
	watch := &watchHandler{
		mgr:  s.mgr,
		id:   "wooo",
		file: "test_data",
	}

	testConfig, err := watch.OnChange("./testdata")
	s.NoError(err)
	s.NotNil(testConfig)
}
