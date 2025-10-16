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
	"google.golang.org/protobuf/proto"
)

func AnyFlow(toID string, toType storage.NetworkEntityInfo_Type, fromID string, fromType storage.NetworkEntityInfo_Type) *storage.NetworkFlow {
	nei := &storage.NetworkEntityInfo{}
	nei.SetType(fromType)
	nei.SetId(fromID)
	nei2 := &storage.NetworkEntityInfo{}
	nei2.SetType(toType)
	nei2.SetId(toID)
	nfp := &storage.NetworkFlowProperties{}
	nfp.SetSrcEntity(nei)
	nfp.SetDstEntity(nei2)
	nf := &storage.NetworkFlow{}
	nf.SetProps(nfp)
	return nf
}

func ExtFlow(toID, fromID string) *storage.NetworkFlow {
	return AnyFlow(toID, storage.NetworkEntityInfo_EXTERNAL_SOURCE, fromID, storage.NetworkEntityInfo_DEPLOYMENT)
}

func DepFlow(toID, fromID string) *storage.NetworkFlow {
	return AnyFlow(toID, storage.NetworkEntityInfo_DEPLOYMENT, fromID, storage.NetworkEntityInfo_DEPLOYMENT)
}

func ListenFlow(depID string, port uint32) *storage.NetworkFlow {
	nei := &storage.NetworkEntityInfo{}
	nei.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei.SetId(depID)
	nei2 := &storage.NetworkEntityInfo{}
	nei2.SetType(storage.NetworkEntityInfo_LISTEN_ENDPOINT)
	nfp := &storage.NetworkFlowProperties{}
	nfp.SetSrcEntity(nei)
	nfp.SetDstEntity(nei2)
	nfp.SetDstPort(port)
	nfp.SetL4Protocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	nf := &storage.NetworkFlow{}
	nf.SetProps(nfp)
	return nf
}

func ExtIdFromIPv4(clusterId string, packedIp uint32) (sac.ResourceID, error) {
	bs := [4]byte{}
	binary.BigEndian.PutUint32(bs[:], packedIp)
	ip := netip.AddrFrom4(bs)
	return externalsrcs.NewClusterScopedID(clusterId, fmt.Sprintf("%s/32", ip.String()))
}

// GetDeploymentNetworkEntity returns a deployment type network entity.
func GetDeploymentNetworkEntity(id, name string) *storage.NetworkEntityInfo {
	nd := &storage.NetworkEntityInfo_Deployment{}
	nd.SetName(name)
	nei := &storage.NetworkEntityInfo{}
	nei.SetId(id)
	nei.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei.SetDeployment(proto.ValueOrDefault(nd))
	return nei
}

// GetExtSrcNetworkEntity returns a external source typed *storage.NetworkEntity object.
func GetExtSrcNetworkEntity(id, name, cidr string, isDefault bool, clusterID string, isDiscovered bool) *storage.NetworkEntity {
	ns := &storage.NetworkEntity_Scope{}
	ns.SetClusterId(clusterID)
	ne := &storage.NetworkEntity{}
	ne.SetInfo(GetExtSrcNetworkEntityInfo(id, name, cidr, isDefault, isDiscovered))
	ne.SetScope(ns)
	return ne
}

// GetExtSrcNetworkEntityInfo returns a external source typed *storage.NetworkEntityInfo object.
func GetExtSrcNetworkEntityInfo(id, name, cidr string, isDefault bool, isDiscovered bool) *storage.NetworkEntityInfo {
	nei := &storage.NetworkEntityInfo{}
	nei.SetId(id)
	nei.SetType(storage.NetworkEntityInfo_EXTERNAL_SOURCE)
	nei.SetExternalSource(storage.NetworkEntityInfo_ExternalSource_builder{
		Name:       name,
		Cidr:       proto.String(cidr),
		Default:    isDefault,
		Discovered: isDiscovered,
	}.Build())
	return nei
}

// GetNetworkFlow returns a network flow constructed from supplied data.
func GetNetworkFlow(src, dst *storage.NetworkEntityInfo, port int, protocol storage.L4Protocol, ts *time.Time) *storage.NetworkFlow {
	nfp := &storage.NetworkFlowProperties{}
	nfp.SetSrcEntity(src)
	nfp.SetDstEntity(dst)
	nfp.SetDstPort(uint32(port))
	nfp.SetL4Protocol(protocol)
	nf := &storage.NetworkFlow{}
	nf.SetProps(nfp)
	nf.SetLastSeenTimestamp(protocompat.ConvertTimeToTimestampOrNil(ts))
	return nf
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
