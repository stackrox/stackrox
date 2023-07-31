package clusterhealth

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/suite"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	// Max time to receive health info status. You may want to increase it if you plan to step through the code with debugger.
	updateTimeout = 3 * time.Second
	// How frequently should updater provide health info during tests.
	updateInterval = 1 * time.Millisecond
	// Environment variable to hold pod namespace. In actual k8s deployment it is set by helm/yaml file.
	namespaceVar = "POD_NAMESPACE"
)

func TestUpdater(t *testing.T) {
	suite.Run(t, new(UpdaterTestSuite))
}

type UpdaterTestSuite struct {
	suite.Suite

	client *fake.Clientset
}

type expectedHealthInfo struct {
	version               string
	desired, ready, nodes int32
	errors                []string
}

func (s *UpdaterTestSuite) SetupTest() {
	s.client = fake.NewSimpleClientset()
	s.T().Setenv(namespaceVar, "stackrox-mock-ns")
}

func (s *UpdaterTestSuite) TestHappyCase() {
	ds := makeDaemonSet()
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *UpdaterTestSuite) TestSlimSuffixTrimmed() {
	ds := makeDaemonSet()
	ds.Spec.Template.Spec.Containers[0].Image = "mock/image:v5.0.1fat-slim"
	s.addDaemonSet(ds)

	health := s.getHealthInfo(1)

	s.assertVersion(health, "v5.0.1fat")
}

func (s *UpdaterTestSuite) TestLatestSuffixTrimmed() {
	ds := makeDaemonSet()
	ds.Spec.Template.Spec.Containers[0].Image = "mock/image:v5.0.2old-latest"
	s.addDaemonSet(ds)

	health := s.getHealthInfo(1)

	s.assertVersion(health, "v5.0.2old")
}

func (s *UpdaterTestSuite) TestDaemonSetWithoutContainerSpec() {
	ds := makeDaemonSet()
	ds.Spec = appsV1.DaemonSetSpec{} // Erase containers information.
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "", desired: 6, ready: 4, nodes: 7, errors: []string{"collector version"},
	})
}

func (s *UpdaterTestSuite) TestWithoutDaemonSet() {
	// No DaemonSet added.
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "", desired: -1, ready: -1, nodes: 7, errors: []string{"collector DaemonSet"},
	})
}

func (s *UpdaterTestSuite) TestWithoutNodes() {
	ds := makeDaemonSet()
	s.addDaemonSet(ds)
	// No nodes get added.

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 0, errors: nil,
	})
}

func (s *UpdaterTestSuite) TestVersionWithoutTag() {
	ds := makeDaemonSet()
	ds.Spec.Template.Spec.Containers[0].Image = "blah/without/tags"
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "blah/without/tags", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *UpdaterTestSuite) TestCanSendMultipleUpdates() {
	s.addDaemonSet(makeDaemonSet())
	s.addNodes(7)

	health := s.getHealthInfo(5)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *UpdaterTestSuite) TestCustomNamespaceHappyCase() {
	const customNs = "custom-test-ns"
	s.T().Setenv(namespaceVar, customNs)

	ds := makeDaemonSet()
	ds.ObjectMeta.Namespace = customNs
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *UpdaterTestSuite) TestNamespaceFallback() {
	s.T().Setenv(namespaceVar, "")
	ds := makeDaemonSet()
	ds.ObjectMeta.Namespace = namespaces.StackRox
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *UpdaterTestSuite) TestNamespaceMismatch() {
	s.T().Setenv(namespaceVar, "where-things-should-be")

	ds := makeDaemonSet()
	ds.ObjectMeta.Namespace = "where-things-are"
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "", desired: -1, ready: -1, nodes: 7, errors: []string{"unable to find collector DaemonSet in namespace \"where-things-should-be\""},
	})
}

func (s *UpdaterTestSuite) TestOfflineMode() {
	states := []common.SensorComponentEvent{
		common.SensorComponentEventCentralReachable,
		common.SensorComponentEventOfflineMode,
		common.SensorComponentEventCentralReachable,
	}
	s.addDaemonSet(makeDaemonSet())
	s.addNodes(4)
	s.addDeployment(makeAdmissionControlDeployment())
	updater := s.createNewUpdater(updateInterval)
	s.Require().NoError(updater.Start())
	defer updater.Stop(nil)
	var expiredMessages []*message.ExpiringMessage
	for _, state := range states {
		updater.Notify(state)
		if expiredMsg := s.assertOfflineMode(state, updater, updateInterval); expiredMsg != nil {
			expiredMessages = append(expiredMessages, expiredMsg)
		}
	}
	for _, msg := range expiredMessages {
		select {
		case <-msg.Context.Done():
			continue
		case <-time.After(time.Second):
			s.Fail("the messages that were attempted to be sent while offline should be expired")
		}
	}
}

func (s *UpdaterTestSuite) TestNotExpiredMessage() {
	s.addDaemonSet(makeDaemonSet())
	s.addNodes(4)
	s.addDeployment(makeAdmissionControlDeployment())

	updater := s.createNewUpdater(updateInterval)
	fakeTicker := make(chan time.Time)
	defer close(fakeTicker)
	go updater.run(fakeTicker)
	updater.Notify(common.SensorComponentEventCentralReachable)
	fakeTicker <- time.Now()
	select {
	case msg := <-updater.ResponsesC():
		select {
		case <-msg.Context.Done():
			s.Fail("the message in ResponseC should not be cancelled")
		case <-time.After(10 * updateInterval):
			break
		}
	case <-time.After(10 * time.Second):
		s.Fail("timeout waiting for sensor message")
	}
}

func (s *UpdaterTestSuite) TestExpiredMessage() {
	s.addDaemonSet(makeDaemonSet())
	s.addNodes(4)
	s.addDeployment(makeAdmissionControlDeployment())

	updater := s.createNewUpdater(updateInterval)
	fakeTicker := make(chan time.Time)
	defer close(fakeTicker)
	go updater.run(fakeTicker)
	updater.Notify(common.SensorComponentEventCentralReachable)
	fakeTicker <- time.Now()
	var msg *message.ExpiringMessage
	select {
	case msg = <-updater.ResponsesC():
		break
	case <-time.After(10 * time.Second):
		s.Fail("timeout waiting for sensor message")
	}
	updater.Notify(common.SensorComponentEventOfflineMode)
	updater.Notify(common.SensorComponentEventCentralReachable)
	select {
	case <-msg.Context.Done():
		break
	case <-time.After(10 * updateInterval):
		s.Fail("the message in ResponseC should be cancelled")
	}
}

func (s *UpdaterTestSuite) createNewUpdater(interval time.Duration) *updaterImpl {
	updaterComponent := NewUpdater(s.client, interval)
	updater, ok := updaterComponent.(*updaterImpl)
	s.Require().True(ok, "NewUpdater should return a struct of type *updaterImpl")
	return updater
}

func (s *UpdaterTestSuite) assertOfflineMode(state common.SensorComponentEvent, updater *updaterImpl, interval time.Duration) *message.ExpiringMessage {
	switch state {
	case common.SensorComponentEventCentralReachable:
		select {
		case <-time.After(10 * time.Second):
			s.Fail("timeout waiting for sensor message")
		case <-updater.ResponsesC():
			return nil
		}
	case common.SensorComponentEventOfflineMode:
		select {
		case <-time.After(10 * interval):
			return nil
		case msg := <-updater.ResponsesC():
			return msg
		}
	}
	return nil
}

func (s *UpdaterTestSuite) getHealthInfo(times int) *storage.CollectorHealthInfo {
	timer := time.NewTimer(updateTimeout)
	updater := NewUpdater(s.client, updateInterval)

	updater.Notify(common.SensorComponentEventCentralReachable)
	err := updater.Start()
	s.Require().NoError(err)
	defer updater.Stop(nil)

	var healthInfo *storage.CollectorHealthInfo

	for i := 0; i < times; i++ {
		select {
		case response := <-updater.ResponsesC():
			healthInfo = response.Msg.(*central.MsgFromSensor_ClusterHealthInfo).ClusterHealthInfo.CollectorHealthInfo
		case <-timer.C:
			s.Fail("Timed out while waiting for cluster health update")
		}
	}

	return healthInfo
}

func makeDaemonSet() appsV1.DaemonSet {
	return appsV1.DaemonSet{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "collector",
			Namespace: "stackrox-mock-ns",
		},
		Spec: appsV1.DaemonSetSpec{
			Template: coreV1.PodTemplateSpec{
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{Name: "collector", Image: "mock/image:v456"},
					},
				},
			},
		},
		Status: appsV1.DaemonSetStatus{
			DesiredNumberScheduled: 6,
			NumberReady:            4,
		},
	}
}

func (s *UpdaterTestSuite) addDaemonSet(ds appsV1.DaemonSet) {
	_, err := s.client.AppsV1().DaemonSets(ds.ObjectMeta.Namespace).Create(context.Background(), &ds, metaV1.CreateOptions{})
	s.Require().NoError(err)
}

func makeAdmissionControlDeployment() appsV1.Deployment {
	return appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "admission-control",
			Namespace: "stackrox-mock-ns",
		},
		Spec: appsV1.DeploymentSpec{
			Template: coreV1.PodTemplateSpec{
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{Name: "admission-control", Image: "mock/ac-image:v456"},
					},
				},
			},
		},
		Status: appsV1.DeploymentStatus{
			Replicas:      2,
			ReadyReplicas: 2,
		},
	}
}

func (s *UpdaterTestSuite) addDeployment(d appsV1.Deployment) {
	_, err := s.client.AppsV1().Deployments(d.ObjectMeta.Namespace).Create(context.Background(), &d, metaV1.CreateOptions{})
	s.Require().NoError(err)
}

func (s *UpdaterTestSuite) addNodes(count int) {
	for i := 0; i < count; i++ {
		_, err := s.client.CoreV1().Nodes().Create(context.Background(), &coreV1.Node{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "mock-node-" + strconv.Itoa(i),
			},
		}, metaV1.CreateOptions{})
		s.Require().NoError(err)
	}
}

func (s *UpdaterTestSuite) assertHealthInfo(actual *storage.CollectorHealthInfo, expected expectedHealthInfo) {
	s.assertVersion(actual, expected.version)
	s.assertTotalDesiredPods(actual, expected.desired)
	s.assertTotalReadyPods(actual, expected.ready)
	s.assertTotalRegisteredNodes(actual, expected.nodes)
	s.assertStatusErrors(actual, expected.errors...)
}

func (s *UpdaterTestSuite) assertVersion(health *storage.CollectorHealthInfo, expected string) {
	s.Equal(expected, health.Version)
}

func (s *UpdaterTestSuite) assertTotalDesiredPods(health *storage.CollectorHealthInfo, expected int32) {
	var actual int32
	switch v := health.TotalDesiredPodsOpt.(type) {
	case *storage.CollectorHealthInfo_TotalDesiredPods:
		actual = v.TotalDesiredPods
	case nil:
		actual = -1
	default:
		s.FailNowf("Unexpected total desired pods value type", "actual value: %#v", health.TotalDesiredPodsOpt)
	}
	s.Equalf(expected, actual, "Unexpected value of total desired pods %#v", health.TotalDesiredPodsOpt)
}

func (s *UpdaterTestSuite) assertTotalReadyPods(health *storage.CollectorHealthInfo, expected int32) {
	var actual int32
	switch v := health.TotalReadyPodsOpt.(type) {
	case *storage.CollectorHealthInfo_TotalReadyPods:
		actual = v.TotalReadyPods
	case nil:
		actual = -1
	default:
		s.FailNowf("Unexpected total ready pods value type", "actual value: %#v", health.TotalReadyPodsOpt)
	}
	s.Equalf(expected, actual, "Unexpected value of total ready pods %#v", health.TotalReadyPodsOpt)
}

func (s *UpdaterTestSuite) assertTotalRegisteredNodes(health *storage.CollectorHealthInfo, expected int32) {
	var actual int32
	switch v := health.TotalRegisteredNodesOpt.(type) {
	case *storage.CollectorHealthInfo_TotalRegisteredNodes:
		actual = v.TotalRegisteredNodes
	case nil:
		actual = -1
	default:
		s.FailNowf("Unexpected total registered nodes value type", "actual value: %#v", health.TotalRegisteredNodesOpt)
	}
	s.Equalf(expected, actual, "Unexpected value of total registered nodes %#v", health.TotalReadyPodsOpt)
}

func (s *UpdaterTestSuite) assertStatusErrors(health *storage.CollectorHealthInfo, expected ...string) {
	s.Len(health.StatusErrors, len(expected))
	for _, e := range expected {
		var found int
		for _, s := range health.StatusErrors {
			if strings.Contains(s, e) {
				found++
			}
		}
		if found != 1 {
			s.Failf(
				"Did not find expected error",
				"Expected to find exactly 1 substring %#v in %#v, found %d",
				e,
				health.StatusErrors,
				found)
		}
	}
}
