package testutils

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
)

func AnyFlow(toID string, toType storage.NetworkEntityInfo_Type, fromID string, fromType storage.NetworkEntityInfo_Type) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: fromType,
				Id:   fromID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: toType,
				Id:   toID,
			},
		},
	}
}

func ExtFlow(toID, fromID string) *storage.NetworkFlow {
	return AnyFlow(toID, storage.NetworkEntityInfo_EXTERNAL_SOURCE, fromID, storage.NetworkEntityInfo_DEPLOYMENT)
}

func DepFlow(toID, fromID string) *storage.NetworkFlow {
	return AnyFlow(toID, storage.NetworkEntityInfo_DEPLOYMENT, fromID, storage.NetworkEntityInfo_DEPLOYMENT)
}

func ListenFlow(depID string, port uint32) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   depID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_LISTEN_ENDPOINT,
			},
			DstPort:    port,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}
}

func ExtIdFromIPv4(clusterId string, packedIp uint32) (sac.ResourceID, error) {
	bs := [4]byte{}
	binary.BigEndian.PutUint32(bs[:], packedIp)
	ip := netip.AddrFrom4(bs)
	return externalsrcs.NewClusterScopedID(clusterId, fmt.Sprintf("%s/32", ip.String()))
}

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
func GetExtSrcNetworkEntity(id, name, cidr string, isDefault bool, clusterID string, isDiscovered bool) *storage.NetworkEntity {
	return &storage.NetworkEntity{
		Info: GetExtSrcNetworkEntityInfo(id, name, cidr, isDefault, isDiscovered),
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: clusterID,
		},
	}
}

// GetExtSrcNetworkEntityInfo returns a external source typed *storage.NetworkEntityInfo object.
func GetExtSrcNetworkEntityInfo(id, name, cidr string, isDefault bool, isDiscovered bool) *storage.NetworkEntityInfo {
	return &storage.NetworkEntityInfo{
		Id:   id,
		Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
		Desc: &storage.NetworkEntityInfo_ExternalSource_{
			ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
				Name: name,
				Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
					Cidr: cidr,
				},
				Default:    isDefault,
				Discovered: isDiscovered,
			},
		},
	}
}

// GetNetworkFlow returns a network flow constructed from supplied data.
func GetNetworkFlow(src, dst *storage.NetworkEntityInfo, port int, protocol storage.L4Protocol, ts *time.Time) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity:  src,
			DstEntity:  dst,
			DstPort:    uint32(port),
			L4Protocol: protocol,
		},
		LastSeenTimestamp: protocompat.ConvertTimeToTimestampOrNil(ts),
	}
}

// GenRandomExtSrcNetworkEntityInfo generates numNetworks number of storage.NetworkEntityInfo objects with random CIDRs.
func GenRandomExtSrcNetworkEntityInfo(family pkgNet.Family, numNetworks int) ([]*storage.NetworkEntityInfo, error) {
	nets, err := genRandomNetworks(family, numNetworks)
	if err != nil {
		return nil, err
	}

	entities := make([]*storage.NetworkEntityInfo, 0, len(nets))
	for k := range nets {
		entities = append(entities, GetExtSrcNetworkEntityInfo(k, k, k, false, false))
	}

	return entities, nil
}

// GenRandomExtSrcNetworkEntity generates numNetworks number of storage.NetworkEntity objects with random CIDRs.
func GenRandomExtSrcNetworkEntity(family pkgNet.Family, numNetworks int, clusterID string) ([]*storage.NetworkEntity, error) {
	nets, err := genRandomNetworks(family, numNetworks)
	if err != nil {
		return nil, err
	}

	entities := make([]*storage.NetworkEntity, 0, len(nets))
	for k := range nets {
		id, err := externalsrcs.NewClusterScopedID(clusterID, k)
		utils.Should(err)
		entities = append(entities, GetExtSrcNetworkEntity(id.String(), k, k, false, clusterID, false))
	}

	return entities, nil
}

func genRandomNetworks(family pkgNet.Family, numNetworks int) (map[string]struct{}, error) {
	nets := make(map[string]struct{})

	var bits int32
	if family == pkgNet.IPv4 {
		bits = 32
	} else if family == pkgNet.IPv6 {
		bits = 128
	}

	ip := make([]byte, bits/8)
	for len(nets) < numNetworks {
		if _, err := rand.Read(ip); err != nil {
			return nil, err
		}

		n, err := networkgraph.ValidateCIDR(net.IP(ip).String() + "/" + strconv.Itoa(int(1+rand.Int31n(bits))))
		if err != nil {
			continue
		}
		nets[n.String()] = struct{}{}
	}
	return nets, nil
}
