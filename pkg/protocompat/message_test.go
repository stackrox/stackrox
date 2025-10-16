package protocompat

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClone(t *testing.T) {
	m1 := &storage.NamespaceMetadata{}
	m1.SetId(testconsts.NamespaceA)
	m1.SetName("Namespace A")
	m1.SetClusterId(testconsts.Cluster1)
	m1.SetClusterName("Cluster 1")

	cloned := Clone(m1)

	assert.True(t, m1.EqualVT(cloned.(*storage.NamespaceMetadata)))

	// Change a field value to ensure the clone does not point
	// to the original struct.
	clonedNamespace, casted := cloned.(*storage.NamespaceMetadata)
	assert.True(t, casted)
	clonedNamespace.SetName("Namespace AA")
	assert.False(t, m1.EqualVT(cloned.(*storage.NamespaceMetadata)))
}

func TestMarshalTextString(t *testing.T) {
	msg := &storage.NamespaceMetadata{}
	msg.SetId(testconsts.NamespaceA)
	msg.SetName("Namespace A")
	msg.SetClusterId(testconsts.Cluster1)
	msg.SetClusterName("Cluster 1")
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

func TestMerge(t *testing.T) {
	msgDst := &storage.NamespaceMetadata{}
	msgDst.SetId(testconsts.NamespaceA)
	msgDst.SetClusterName("Cluster 1")

	msgSrc := &storage.NamespaceMetadata{}
	msgSrc.SetName("Namespace A")
	msgSrc.SetClusterName("Cluster 2")

	Merge(msgDst, msgSrc)

	assert.Equal(t, testconsts.NamespaceA, msgDst.GetId())
	assert.Equal(t, "Namespace A", msgDst.GetName())
	assert.Equal(t, "Cluster 2", msgDst.GetClusterName())
	assert.Equal(t, "", msgDst.GetClusterId())
}

func TestMarshalMap(t *testing.T) {
	expected := map[string]interface{}{
		"clusterId":   "aaaaaaaa-bbbb-4011-0000-111111111111",
		"clusterName": "Cluster 1",
		"id":          "namespaceA",
		"name":        "Namespace A",
	}

	msg := &storage.NamespaceMetadata{}
	msg.SetId(testconsts.NamespaceA)
	msg.SetName("Namespace A")
	msg.SetClusterId(testconsts.Cluster1)
	msg.SetClusterName("Cluster 1")

	marshalled, err := MarshalMap(msg)
	require.NoError(t, err)
	assert.Equal(t, expected, marshalled)

	// Test with nil value
	marshalled, err = MarshalMap(nil)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{}, marshalled)
}
