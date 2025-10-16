package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetDeploymentIDsFromKey(t *testing.T) {
	nei := &storage.NetworkEntityInfo{}
	nei.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei.SetId("id1")
	nei2 := &storage.NetworkEntityInfo{}
	nei2.SetType(storage.NetworkEntityInfo_INTERNET)
	nei2.SetId("id2")
	nfp := &storage.NetworkFlowProperties{}
	nfp.SetSrcEntity(nei)
	nfp.SetDstEntity(nei2)
	nfp.SetDstPort(8080)
	nfp.SetL4Protocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	id := GetID(nfp)

	id1, id2 := GetDeploymentIDsFromKey(id)
	assert.Equal(t, []byte("id1"), id1)
	assert.Equal(t, []byte("id2"), id2)
}
