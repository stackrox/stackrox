package networkbaseline

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/timestamp"
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
