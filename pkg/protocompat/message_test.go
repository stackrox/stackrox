package protocompat

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stretchr/testify/assert"
)

func TestClone(t *testing.T) {
	m1 := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		Name:        "Namespace A",
		ClusterId:   testconsts.Cluster1,
		ClusterName: "Cluster 1",
	}

	cloned := Clone(m1)

	assert.True(t, Equal(m1, cloned))

	// Change a field value to ensure the clone does not point
	// to the original struct.
	clonedNamespace, casted := cloned.(*storage.NamespaceMetadata)
	assert.True(t, casted)
	clonedNamespace.Name = "Namespace AA"
	assert.False(t, Equal(m1, cloned))
}

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

func TestMarshal(t *testing.T) {
	msg := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		Name:        "Namespace A",
		ClusterId:   testconsts.Cluster1,
		ClusterName: "Cluster 1",
	}
	bytes, err := Marshal(msg)
	assert.NoError(t, err)
	expectedBytes := []byte{
		'\x0a', '\x0a', '\x6e', '\x61', '\x6d', '\x65', '\x73', '\x70',
		'\x61', '\x63', '\x65', '\x41', '\x12', '\x0b', '\x4e', '\x61',
		'\x6d', '\x65', '\x73', '\x70', '\x61', '\x63', '\x65', '\x20',
		'\x41', '\x1a', '\x24', '\x61', '\x61', '\x61', '\x61', '\x61',
		'\x61', '\x61', '\x61', '\x2d', '\x62', '\x62', '\x62', '\x62',
		'\x2d', '\x34', '\x30', '\x31', '\x31', '\x2d', '\x30', '\x30',
		'\x30', '\x30', '\x2d', '\x31', '\x31', '\x31', '\x31', '\x31',
		'\x31', '\x31', '\x31', '\x31', '\x31', '\x31', '\x31', '\x22',
		'\x09', '\x43', '\x6c', '\x75', '\x73', '\x74', '\x65', '\x72',
		'\x20', '\x31',
	}
	assert.Equal(t, expectedBytes, bytes)
}

func TestMarshalTextString(t *testing.T) {
	msg := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		Name:        "Namespace A",
		ClusterId:   testconsts.Cluster1,
		ClusterName: "Cluster 1",
	}
	asString := MarshalTextString(msg)

	// String output is not guarantied.
	// Info: https://pkg.go.dev/google.golang.org/protobuf@v1.34.1/encoding/prototext#Format
	// There is randomization added to output to ensure that library users
	// are not relaying on stable output format.
	// Info: https://go-review.googlesource.com/c/protobuf/+/151340
	expectedRegex := `id: +"namespaceA"
` + `name: +"Namespace A"
` + `cluster_id: +"aaaaaaaa-bbbb-4011-0000-111111111111"
` + `cluster_name: +"Cluster 1"
`
	assert.Regexp(t, expectedRegex, asString)
}

func TestUnmarshal(t *testing.T) {
	msg := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		Name:        "Namespace A",
		ClusterId:   testconsts.Cluster1,
		ClusterName: "Cluster 1",
	}
	data, err := proto.Marshal(msg)
	assert.NoError(t, err)

	decoded := &storage.NamespaceMetadata{}
	err = Unmarshal(data, decoded)
	assert.NoError(t, err)
	assert.True(t, Equal(msg, decoded))
}

func TestMerge(t *testing.T) {
	msgDst := &storage.NamespaceMetadata{
		Id:          testconsts.NamespaceA,
		ClusterName: "Cluster 1",
	}

	msgSrc := &storage.NamespaceMetadata{
		Name:        "Namespace A",
		ClusterName: "Cluster 2",
	}

	Merge(msgDst, msgSrc)

	assert.Equal(t, testconsts.NamespaceA, msgDst.GetId())
	assert.Equal(t, "Namespace A", msgDst.GetName())
	assert.Equal(t, "Cluster 2", msgDst.GetClusterName())
	assert.Equal(t, "", msgDst.GetClusterId())
}
