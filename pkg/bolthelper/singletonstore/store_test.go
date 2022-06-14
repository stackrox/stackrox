package singletonstore

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestSingletonStore(t *testing.T) {
	db := testutils.DBForT(t)
	defer testutils.TearDownDB(db)

	store := New(db, []byte("blah"), func() proto.Message {
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
