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
)

func TestUpdater(t *testing.T) {
	suite.Run(t, new(UpdaterTestSuite))
}

type UpdaterTestSuite struct {
	suite.Suite

	client  *fake.Clientset
	updater common.SensorComponent
}

func (s *UpdaterTestSuite) SetupTest() {
	s.client = fake.NewSimpleClientset()
	s.updater = NewUpdater(s.client, updateInterval)
}

func (s *UpdaterTestSuite) TestHappyCase() {
	ds := makeDaemonSet()
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertVersion(health, "v456")
	s.assertTotalDesiredPods(health, 6)
	s.assertTotalReadyPods(health, 4)
	s.assertTotalRegisteredNodes(health, 7)
	s.assertNoStatusErrors(health)
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

	s.assertStatusErrors(health, "collector version")
	s.assertVersion(health, "")

	s.assertTotalDesiredPods(health, 6)
	s.assertTotalReadyPods(health, 4)
	s.assertTotalRegisteredNodes(health, 7)
}

func (s *UpdaterTestSuite) TestWithoutDaemonSet() {
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertStatusErrors(health, "collector DaemonSet")
	s.assertVersion(health, "")
	s.assertTotalRegisteredNodes(health, 7)
	s.assertTotalDesiredPods(health, -1)
	s.assertTotalDesiredPods(health, -1)
}

func (s *UpdaterTestSuite) TestWithoutNodes() {
	ds := makeDaemonSet()
	s.addDaemonSet(ds)
	// No nodes get added.

	health := s.getHealthInfo(1)

	s.assertVersion(health, "v456")
	s.assertTotalDesiredPods(health, 6)
	s.assertTotalReadyPods(health, 4)
	s.assertTotalRegisteredNodes(health, 0)
	s.assertNoStatusErrors(health)
}

func (s *UpdaterTestSuite) TestVersionWithoutTag() {
	ds := makeDaemonSet()
	ds.Spec.Template.Spec.Containers[0].Image = "blah/without/tags"
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertVersion(health, "blah/without/tags")
	s.assertTotalDesiredPods(health, 6)
	s.assertTotalReadyPods(health, 4)
	s.assertTotalRegisteredNodes(health, 7)
	s.assertNoStatusErrors(health)
}

func (s *UpdaterTestSuite) TestCanSendMultipleUpdates() {
	s.addDaemonSet(makeDaemonSet())
	s.addNodes(7)

	health := s.getHealthInfo(5)

	s.NotNil(health)
}

func (s *UpdaterTestSuite) getHealthInfo(times int) *storage.CollectorHealthInfo {
	timer := time.NewTimer(updateTimeout)

	err := s.updater.Start()
	s.Require().NoError(err)
	defer s.updater.Stop(nil)

	var healthInfo *storage.CollectorHealthInfo

	for i := 0; i < times; i++ {
		select {
		case response := <-s.updater.ResponsesC():
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
			Namespace: namespaces.StackRox,
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
	_, err := s.client.AppsV1().DaemonSets(namespaces.StackRox).Create(context.Background(), &ds, metaV1.CreateOptions{})
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

func (s *UpdaterTestSuite) assertNoStatusErrors(health *storage.CollectorHealthInfo) {
	s.assertStatusErrors(health)
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
