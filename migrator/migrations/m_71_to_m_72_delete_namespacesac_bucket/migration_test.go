package m71tom72

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecbot/gorocksdb"
)

func TestMigration(t *testing.T) {
	writeOpts := gorocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()
	readOpts := gorocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	var (
		deploymentBucket = []byte("deployment")
		clusterBucket    = []byte("xluster") // ensure it appears after namespacesSAC
	)
	db := rocksdbtest.RocksDBForT(t)
	defer db.Close()

	nsSksBefore := SortedCopy([][]byte{
		prefixKey(deploymentBucket, []byte("deployment1")),
		prefixKey(clusterBucket, []byte("cluster1")),
		prefixKey(nsSACBucketName, []byte("ns1")),
	})

	nsSACSksBefore := SortedCopy([][]byte{
		prefixKey(nsBucketName, []byte("ns1")),
	})

	require.NoError(t, db.Put(writeOpts, getGraphKey(prefixKey(nsBucketName, []byte("ns1"))), nsSksBefore.Marshal()))
	require.NoError(t, db.Put(writeOpts, getGraphKey(prefixKey(nsSACBucketName, []byte("ns1"))), nsSACSksBefore.Marshal()))

	assert.NoError(t, deleteNamespaceSACBucketAndEdges(db.DB))

	val, err := db.GetBytes(readOpts, getGraphKey(prefixKey(nsSACBucketName, []byte("ns1"))))
	require.NoError(t, err)
	assert.Nil(t, val)

	val, err = db.GetBytes(readOpts, getGraphKey(prefixKey(nsBucketName, []byte("ns1"))))
	require.NoError(t, err)
	nsSksAfter, err := Unmarshal(val)
	require.NoError(t, err)

	expectedNsSksAfter := SortedCopy([][]byte{
		prefixKey(deploymentBucket, []byte("deployment1")),
		prefixKey(clusterBucket, []byte("cluster1")),
	})

	require.Len(t, nsSksAfter, len(expectedNsSksAfter))
	for i, sk := range nsSksAfter {
		assert.Equal(t, expectedNsSksAfter[i], sk)
	}
}
