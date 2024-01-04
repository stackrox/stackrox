package protocompat

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {

	m1 := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		Name:        "Namespace A",
		ClusterId:   testconsts.Cluster1,
		ClusterName: "Cluster 1",
	}
	m2 := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		Name:        "Namespace A",
		ClusterId:   testconsts.Cluster1,
		ClusterName: "Cluster 1",
	}
	m3 := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		Name:        "Namespace A",
		ClusterId:   testconsts.Cluster2,
		ClusterName: "Cluster 2",
	}
	m4 := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceB,
		Name:        "Namespace B",
		ClusterId:   testconsts.Cluster2,
		ClusterName: "Cluster 2",
	}
	assert.True(t, Equal(m1, m1))
	assert.True(t, Equal(m1, m2))
	assert.False(t, Equal(m1, m3))
	assert.False(t, Equal(m1, m4))
	assert.True(t, Equal(m2, m2))
	assert.False(t, Equal(m2, m3))
	assert.False(t, Equal(m2, m4))
	assert.True(t, Equal(m3, m3))
	assert.False(t, Equal(m3, m4))
	assert.True(t, Equal(m4, m4))
}

func TestMarshalTextString(t *testing.T) {
	msg := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		Name:        "Namespace A",
		ClusterId:   testconsts.Cluster1,
		ClusterName: "Cluster 1",
	}
	asString := MarshalTextString(msg)
	expectedString := `id: "namespaceA"
` + `name: "Namespace A"
` + `cluster_id: "aaaaaaaa-bbbb-4011-0000-111111111111"
` + `cluster_name: "Cluster 1"
`
	assert.Equal(t, expectedString, asString)
}
