package common

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetDeploymentIDsFromKey(t *testing.T) {
	id := GetID(&storage.NetworkFlowProperties{
		SrcEntity: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			Id:   "id1",
		},
		DstEntity: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_INTERNET,
			Id:   "id2",
		},
		DstPort:    8080,
		L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	})

	id1, id2 := GetDeploymentIDsFromKey(id)
	assert.Equal(t, []byte("id1"), id1)
	assert.Equal(t, []byte("id2"), id2)
}
