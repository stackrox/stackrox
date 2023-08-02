package clustermetrics

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/suite"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	// Must be larger than defaultInterval. You may want to increase it if you plan
	// to step through the code with a debugger.
	metricsTimeout = 300 * time.Millisecond
)

func TestClusterMetrics(t *testing.T) {
	suite.Run(t, new(ClusterMetricsTestSuite))
}

type ClusterMetricsTestSuite struct {
	suite.Suite

	client *fake.Clientset
}

func (s *ClusterMetricsTestSuite) SetupTest() {
	s.client = fake.NewSimpleClientset()
	defaultInterval = 10 * time.Millisecond
}

func (s *ClusterMetricsTestSuite) TestZeroNodes() {
	expected := &central.ClusterMetrics{NodeCount: 0, CpuCapacity: 0}

	metrics := s.getClusterMetrics()

	s.Equal(expected, metrics)
}

func (s *ClusterMetricsTestSuite) TestSingleNode() {
	expected := &central.ClusterMetrics{NodeCount: 1, CpuCapacity: 10}
	s.addNode("node-1", *resource.NewQuantity(expected.CpuCapacity, resource.DecimalSI))

	metrics := s.getClusterMetrics()

	s.Equal(expected, metrics)
}

func (s *ClusterMetricsTestSuite) TestMultipleNodes() {
	expected := &central.ClusterMetrics{NodeCount: 3, CpuCapacity: 10}
	s.addNode("node-1", *resource.NewQuantity(5, resource.DecimalSI))
	s.addNode("node-2", *resource.NewQuantity(3, resource.DecimalSI))
	s.addNode("node-3", *resource.NewQuantity(2, resource.DecimalSI))

	metrics := s.getClusterMetrics()

	s.Equal(expected, metrics)
}

func (s *ClusterMetricsTestSuite) TestOfflineMode() {
	states := []common.SensorComponentEvent{
		common.SensorComponentEventCentralReachable,
		common.SensorComponentEventOfflineMode,
		common.SensorComponentEventCentralReachable,
	}
	metrics := s.createNewClusterMetrics(time.Millisecond)
	s.Require().NoError(metrics.Start())
	defer metrics.Stop(nil)
	// Read the first message. This is needed because we call runPipeline before entering the ticker loop.
	// This first call will block the goroutine until the message is read.
	select {
	case <-metrics.ResponsesC():
		break
	case <-time.After(metricsTimeout):
		s.Fail("timeout waiting for the first message")
	}
	for _, state := range states {
		metrics.Notify(state)
		s.assertOfflineMode(state, metrics)
	}
}

func (s *ClusterMetricsTestSuite) createNewClusterMetrics(interval time.Duration) *clusterMetricsImpl {
	metricsComponent := NewWithInterval(s.client, interval)
	metrics, ok := metricsComponent.(*clusterMetricsImpl)
	s.Require().True(ok, "New should return a struct of type *clusterMetricsImpl")
	return metrics
}

func (s *ClusterMetricsTestSuite) assertOfflineMode(state common.SensorComponentEvent, metrics *clusterMetricsImpl) {
	switch state {
	case common.SensorComponentEventCentralReachable:
		select {
		case <-time.After(metricsTimeout):
			s.Fail("timeout waiting for the pollTicker to tick")
		case <-metrics.pollTicker.C:
			return
		}
	case common.SensorComponentEventOfflineMode:
		select {
		case <-time.After(2 * metrics.pollingInterval):
			return
		case <-metrics.pollTicker.C:
			s.Fail("the pollTicker should not tick in offline mode")
		}
	}
}

func (s *ClusterMetricsTestSuite) getClusterMetrics() *central.ClusterMetrics {
	timer := time.NewTimer(metricsTimeout)
	clusterMetricsStream := New(s.client)

	clusterMetricsStream.Notify(common.SensorComponentEventCentralReachable)
	err := clusterMetricsStream.Start()
	s.Require().NoError(err)
	defer clusterMetricsStream.Stop(nil)

	select {
	case response := <-clusterMetricsStream.ResponsesC():
		metrics := response.GetClusterMetrics()
		return metrics
	case <-timer.C:
		s.Fail("Timed out while waiting for cluster metrics.")
	}
	return nil
}

func (s *ClusterMetricsTestSuite) addNode(name coreV1.ResourceName, cpu resource.Quantity) {
	_, err := s.client.CoreV1().Nodes().Create(context.Background(), &coreV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name.String(),
		},
		Status: coreV1.NodeStatus{
			Capacity: coreV1.ResourceList{"cpu": cpu},
		},
	}, metaV1.CreateOptions{})
	s.Require().NoError(err)
}
