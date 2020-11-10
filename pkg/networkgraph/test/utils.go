package test

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// GetDeploymentNetworkEntity returns a deployment type network entity.
func GetDeploymentNetworkEntity(id, name string) *storage.NetworkEntityInfo {
	return &storage.NetworkEntityInfo{
		Id:   id,
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		Desc: &storage.NetworkEntityInfo_Deployment_{
			Deployment: &storage.NetworkEntityInfo_Deployment{
				Name: name,
			},
		},
	}
}

// GetExtSrcNetworkEntity returns a external source typed *storage.NetworkEntity object.
func GetExtSrcNetworkEntity(id, name, cidr string, isDefault bool, clusterID string) *storage.NetworkEntity {
	return &storage.NetworkEntity{
		Info: GetExtSrcNetworkEntityInfo(id, name, cidr, isDefault),
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: clusterID,
		},
	}
}

// GetExtSrcNetworkEntityInfo returns a external source type network entity.
func GetExtSrcNetworkEntityInfo(id, name, cidr string, isDefault bool) *storage.NetworkEntityInfo {
	return &storage.NetworkEntityInfo{
		Id:   id,
		Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
		Desc: &storage.NetworkEntityInfo_ExternalSource_{
			ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
				Name: name,
				Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
					Cidr: cidr,
				},
				Default: isDefault,
			},
		},
	}
}

// GetNetworkFlow returns a network flow constructed from supplied data.
func GetNetworkFlow(src, dst *storage.NetworkEntityInfo, port int, protocol storage.L4Protocol, ts *types.Timestamp) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity:  src,
			DstEntity:  dst,
			DstPort:    uint32(port),
			L4Protocol: protocol,
		},
		LastSeenTimestamp: ts,
	}
}
