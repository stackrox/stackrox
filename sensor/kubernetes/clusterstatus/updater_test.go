package clusterstatus

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stretchr/testify/suite"
)

type updaterSuite struct {
	suite.Suite
	updater common.SensorComponent
}

func TestClusterStatusUpdater(t *testing.T) {
	suite.Run(t, new(updaterSuite))
}

func (s *updaterSuite) createUpdater(getProviders func(context.Context) *storage.ProviderMetadata) {
	cl := k8s.MakeFakeClient()
	s.updater = NewUpdater(cl)
	s.updater.(*updaterImpl).getProviders = getProviders
}

func (s *updaterSuite) online() {
	s.updater.Notify(common.SensorComponentEventCentralReachable)
}

func (s *updaterSuite) offline() {
	s.updater.Notify(common.SensorComponentEventOfflineMode)
}

func assertContextIsCancelled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		return errors.New("context is not cancelled")
	}
}

func (s *updaterSuite) readStatus() {
	msg, more := <-s.updater.ResponsesC()
	s.Assert().True(more, "channel should be open")
	s.Assert().False(msg.IsExpired(), "message should not be expired")
	s.Assert().NotNil(msg.GetClusterStatusUpdate().GetStatus(), "message should be ClusterStatus")
}

func (s *updaterSuite) readCancelledStatus() {
	updater, ok := s.updater.(*updaterImpl)
	s.Require().True(ok)
	select {
	case msg, more := <-s.updater.ResponsesC():
		s.Assert().True(more, "channel should be open")
		s.Assert().True(msg.IsExpired(), "message should not be expired")
		s.Assert().NotNil(msg.GetClusterStatusUpdate().GetStatus(), "message should be ClusterStatus")
	case <-time.After(10 * time.Nanosecond):
		// If context is cancelled the message might not be sent at all
		s.Assert().NoError(assertContextIsCancelled(updater.getCurrentContext()))
	}
}

func (s *updaterSuite) readDeploymentEnv() {
	msg, more := <-s.updater.ResponsesC()
	s.Assert().True(more, "channel should be open")
	s.Assert().False(msg.IsExpired(), "message should not be expired")
	s.Assert().NotNil(msg.GetClusterStatusUpdate().GetDeploymentEnvUpdate(), "message should be DeploymentEnvUpdate")
}

func (s *updaterSuite) readCancelledDeploymentEnv() {
	updater, ok := s.updater.(*updaterImpl)
	s.Require().True(ok)
	select {
	case msg, more := <-s.updater.ResponsesC():
		s.Assert().True(more, "channel should be open")
		s.Assert().True(msg.IsExpired(), "message should not be expired")
		s.Assert().NotNil(msg.GetClusterStatusUpdate().GetDeploymentEnvUpdate(), "message should be DeploymentEnvUpdate")
	case <-time.After(10 * time.Nanosecond):
		// If context is cancelled the message might not be sent at all
		s.Assert().NoError(assertContextIsCancelled(updater.getCurrentContext()))
	}
}

func mockGetMetadata(_ context.Context) *storage.ProviderMetadata {
	return &storage.ProviderMetadata{}
}

func (s *updaterSuite) Test_OfflineMode() {
	cases := map[string][]func(){
		"Online, offline, read":                           {s.online, s.offline, s.readCancelledStatus},
		"Online, read, offline, read":                     {s.online, s.readStatus, s.offline, s.readCancelledDeploymentEnv},
		"Online, read, read, offline, online, read, read": {s.online, s.readStatus, s.readDeploymentEnv, s.offline, s.online, s.readStatus, s.readDeploymentEnv},
	}
	for tName, tc := range cases {
		s.Run(tName, func() {
			s.createUpdater(mockGetMetadata)
			for _, fn := range tc {
				fn()
			}
		})
	}
}
