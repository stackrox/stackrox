package clustermetrics

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/suite"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	// Must be larger than defaultInterval. You may want to increase it if you plan
	// to step through the code with a debugger.
	metricsTimeout = 100 * time.Millisecond
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
	defaultInterval = 1 * time.Millisecond
}

func (s *ClusterMetricsTestSuite) TestZeroNodes() {
	expected := &central.ClusterMetrics{NodeCount: 0, CpuCapacity: 0}

	metrics := s.getClusterMetrics()

	s.Equal(metrics, expected)
}

func (s *ClusterMetricsTestSuite) TestSingleNode() {
	expected := &central.ClusterMetrics{NodeCount: 1, CpuCapacity: 10}
	s.addNode("node-1", *resource.NewQuantity(expected.CpuCapacity, resource.DecimalSI))

	metrics := s.getClusterMetrics()

	s.Equal(metrics, expected)
}

func (s *ClusterMetricsTestSuite) TestMultipleNodes() {
	expected := &central.ClusterMetrics{NodeCount: 3, CpuCapacity: 10}
	s.addNode("node-1", *resource.NewQuantity(5, resource.DecimalSI))
	s.addNode("node-2", *resource.NewQuantity(3, resource.DecimalSI))
	s.addNode("node-3", *resource.NewQuantity(2, resource.DecimalSI))

	metrics := s.getClusterMetrics()

	s.Equal(metrics, expected)
}

func (s *ClusterMetricsTestSuite) getClusterMetrics() *central.ClusterMetrics {
	timer := time.NewTimer(metricsTimeout)
	clusterMetricsStream := New(s.client)

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
