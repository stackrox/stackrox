package m43tom44

import (
	"fmt"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecbot/gorocksdb"
)

func TestSubjectMigration(t *testing.T) {
	rocksDB, dir, err := rocksdb.NewTemp(t.Name())
	require.NoError(t, err)
	func() { _ = os.RemoveAll(dir) }()

	wb := gorocksdb.NewWriteBatch()
	bindings := make([]*storage.K8SRoleBinding, 0, 10000)
	for i := 0; i < 10000; i++ {
		binding := &storage.K8SRoleBinding{
			Id:          fmt.Sprintf("binding%d", i),
			ClusterId:   "clusterid",
			ClusterName: "clustername",
		}
		for j := 0; j < 3; j++ {
			binding.Subjects = []*storage.Subject{
				{
					Name: fmt.Sprintf("subject%d", j),
				},
			}
		}

		data, err := proto.Marshal(binding)
		require.NoError(t, err)

		wb.Put([]byte(fmt.Sprintf("%s%s", bindingPrefix, binding.GetId())), data)

		bindings = append(bindings, binding)
	}
	require.NoError(t, rocksDB.Write(gorocksdb.NewDefaultWriteOptions(), wb))
	assert.NoError(t, runEnrichSubjects(rocksDB.DB))

	readOpts := gorocksdb.NewDefaultReadOptions()
	for i := 0; i < 10000; i++ {
		expectedBinding := bindings[i]
		enrichSubjects(expectedBinding)

		slice, err := rocksDB.Get(readOpts, []byte(fmt.Sprintf("%sbinding%d", bindingPrefix, i)))
		require.NoError(t, err)
		require.True(t, slice.Exists())

		var retrievedBinding storage.K8SRoleBinding
		require.NoError(t, proto.Unmarshal(slice.Data(), &retrievedBinding))

		assert.Equal(t, expectedBinding, &retrievedBinding)
	}
}
