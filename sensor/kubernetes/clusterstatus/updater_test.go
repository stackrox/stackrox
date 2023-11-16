package clusterstatus

import (
	"context"
	"testing"
	"time"

	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	operatorVersioned "github.com/openshift/client-go/operator/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type updaterSuite struct {
	suite.Suite
	updater common.SensorComponent
}

func TestClusterStatusUpdater(t *testing.T) {
	suite.Run(t, new(updaterSuite))
}

type fakeClientSet struct {
	k8s kubernetes.Interface
}

func (c *fakeClientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

func (c *fakeClientSet) Dynamic() dynamic.Interface {
	return nil
}

func (c *fakeClientSet) OpenshiftApps() appVersioned.Interface {
	return nil
}

func (c *fakeClientSet) OpenshiftConfig() configVersioned.Interface {
	return nil
}

func (c *fakeClientSet) OpenshiftRoute() routeVersioned.Interface {
	return nil
}

func (c *fakeClientSet) OpenshiftOperator() operatorVersioned.Interface {
	return nil
}

func (s *updaterSuite) createUpdater(getProviders func(context.Context) *storage.ProviderMetadata) {
	s.updater = NewUpdater(&fakeClientSet{
		k8s: fake.NewSimpleClientset(),
	})
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
