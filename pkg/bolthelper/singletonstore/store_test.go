package singletonstore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestSingletonStore(t *testing.T) {
	db := testutils.DBForT(t)
	defer testutils.TearDownDB(db)

	store := New(db, []byte("blah"), func() protocompat.Message {
		return &storage.Cluster{}
	}, "objectName")
	got, err := store.Get()
	assert.NoError(t, err)
	assert.Nil(t, got)

	testCluster := &storage.Cluster{Id: "asfafs"}
	assert.NoError(t, store.Upsert(testCluster))

	got, err = store.Get()
	assert.NoError(t, err)
	assert.Equal(t, testCluster, got.(*storage.Cluster))
}
