package networkbaseline

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
)

// BaselineInfo is a in-memory representation of a network baseline
type BaselineInfo struct {
	// Metadata that doesn't change.
	ClusterID      string
	Namespace      string
	DeploymentName string

	ObservationPeriodEnd timestamp.MicroTS
	UserLocked           bool
	BaselinePeers        map[Peer]struct{}
	ForbiddenPeers       map[Peer]struct{}
}

// GetPeer returns the Peer struct if found in BaselinePeers.
// Return first parameter (ok) false and empty peer if not found.
func (i *BaselineInfo) GetPeer(id string) (bool, Peer) {
	if i == nil {
		return false, Peer{}
	}
	return i.getPeerFrom(id, i.BaselinePeers)
}

// GetForbiddenPeer returns the Peer struct if found in ForbiddenPeers.
// Return first parameter (ok) false and empty peer if not found.
func (i *BaselineInfo) GetForbiddenPeer(id string) (bool, Peer) {
	if i == nil {
		return false, Peer{}
	}
	return i.getPeerFrom(id, i.ForbiddenPeers)
}

func (i *BaselineInfo) getPeerFrom(id string, from map[Peer]struct{}) (bool, Peer) {
	for peer := range from {
		if peer.Entity.ID == id {
			return true, peer
		}
	}
	return false, Peer{}
}

// ConvertBaselineInfoFromProto converts proto NetworkBaseline to its in memory representation
func ConvertBaselineInfoFromProto(protoBaseline *storage.NetworkBaseline) (*BaselineInfo, error) {
	peers, err := ConvertPeersFromProto(protoBaseline.GetPeers())
	if err != nil {
		return nil, err
	}
	forbiddenPeers, err := ConvertPeersFromProto(protoBaseline.GetForbiddenPeers())
	if err != nil {
		return nil, err
	}
	return &BaselineInfo{
		ClusterID:            protoBaseline.GetClusterId(),
		Namespace:            protoBaseline.GetNamespace(),
		DeploymentName:       protoBaseline.GetDeploymentName(),
		ObservationPeriodEnd: timestamp.FromProtobuf(protoBaseline.GetObservationPeriodEnd()),
		UserLocked:           protoBaseline.GetLocked(),
		BaselinePeers:        peers,
		ForbiddenPeers:       forbiddenPeers,
	}, nil
}
