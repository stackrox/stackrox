package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeSimpleNamespace() *NamespaceMetadata {
	labels := map[string]string{
		"app":  "test",
		"env":  "prod",
		"team": "platform",
	}

	return &NamespaceMetadata{
		Id:           uuid.NewString(),
		Name:         "test-namespace",
		ClusterId:    uuid.NewString(),
		ClusterName:  "test-cluster",
		Labels:       labels,
		Annotations:  map[string]string{"owner": "team-a"},
		CreationTime: timestamppb.New(time.Now()),
		Priority:     100,
	}
}

func BenchmarkNamespaceCloneVT(b *testing.B) {
	ns := makeSimpleNamespace()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ns.CloneVT()
	}
}

func BenchmarkNamespaceReadFields(b *testing.B) {
	ns := makeSimpleNamespace()
	b.ResetTimer()
	b.ReportAllocs()

	var clusterId, name string
	for i := 0; i < b.N; i++ {
		clusterId = ns.GetClusterId()
		name = ns.GetName()
	}
	_ = clusterId
	_ = name
}

func BenchmarkClusterCloneVT(b *testing.B) {
	cluster := &Cluster{
		Id:             uuid.NewString(),
		Name:           "test-cluster",
		Type:           ClusterType_KUBERNETES_CLUSTER,
		MainImage:      "stackrox.io/main:latest",
		CollectorImage: "stackrox.io/collector:latest",
		Priority:       100,
	}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cluster.CloneVT()
	}
}

func BenchmarkClusterReadFields(b *testing.B) {
	cluster := &Cluster{
		Id:             uuid.NewString(),
		Name:           "test-cluster",
		Type:           ClusterType_KUBERNETES_CLUSTER,
		MainImage:      "stackrox.io/main:latest",
		CollectorImage: "stackrox.io/collector:latest",
		Priority:       100,
	}
	b.ResetTimer()
	b.ReportAllocs()

	var id, name string
	for i := 0; i < b.N; i++ {
		id = cluster.GetId()
		name = cluster.GetName()
	}
	_ = id
	_ = name
}
