package manager

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	managerCtx = sac.WithAllAccess(context.Background())

	networkBaselineSAC = sac.ForResource(resources.NetworkBaseline)

	log = logging.LoggerForModule()
)

type baselineInfo struct {
	// Metadata that doesn't change.
	clusterID string
	namespace string

	observationPeriodEnd timestamp.MicroTS
	userLocked           bool
	baselinePeers        map[peer]struct{}
	forbiddenPeers       map[peer]struct{}
}

type manager struct {
	ds datastore.DataStore

	baselinesByDeploymentID map[string]baselineInfo

	lock sync.Mutex
}

var (
	validBaselinePeerEntityTypes = map[storage.NetworkEntityInfo_Type]struct{}{
		storage.NetworkEntityInfo_DEPLOYMENT:      {},
		storage.NetworkEntityInfo_EXTERNAL_SOURCE: {},
		storage.NetworkEntityInfo_INTERNET:        {},
	}
)

func (m *manager) shouldUpdate(conn *networkgraph.NetworkConnIndicator, updateTS timestamp.MicroTS) bool {
	var atLeastOneBaselineInObservationPeriod bool
	for _, entity := range []*networkgraph.Entity{&conn.SrcEntity, &conn.DstEntity} {
		if _, valid := validBaselinePeerEntityTypes[entity.Type]; !valid {
			return false
		}
		if entity.ID == "" {
			return false
		}
		if entity.Type != storage.NetworkEntityInfo_DEPLOYMENT {
			continue
		}
		baselineInfo, found := m.baselinesByDeploymentID[entity.ID]
		// This likely means the deployment has been deleted.
		// We can avoid processing this flow altogether.
		if !found {
			return false
		}
		// If even one baseline is user-locked, no updates based on flows.
		if baselineInfo.userLocked {
			return false
		}
		if baselineInfo.observationPeriodEnd.After(updateTS) {
			atLeastOneBaselineInObservationPeriod = true
		}
	}
	return atLeastOneBaselineInObservationPeriod
}

func (m *manager) maybeAddPeer(deploymentID string, p *peer, modifiedDeploymentIDs set.StringSet) {
	_, isForbidden := m.baselinesByDeploymentID[deploymentID].forbiddenPeers[*p]
	if isForbidden {
		return
	}
	_, alreadyInBaseline := m.baselinesByDeploymentID[deploymentID].baselinePeers[*p]
	if alreadyInBaseline {
		return
	}
	m.baselinesByDeploymentID[deploymentID].baselinePeers[*p] = struct{}{}
	modifiedDeploymentIDs.Add(deploymentID)
}

func (m *manager) persistNetworkBaselines(deploymentIDs set.StringSet) error {
	if len(deploymentIDs) == 0 {
		return nil
	}
	baselines := make([]*storage.NetworkBaseline, 0, len(deploymentIDs))
	for deploymentID := range deploymentIDs {
		baselineInfo := m.baselinesByDeploymentID[deploymentID]
		baselines = append(baselines, &storage.NetworkBaseline{
			DeploymentId:         deploymentID,
			ClusterId:            baselineInfo.clusterID,
			Namespace:            baselineInfo.namespace,
			Peers:                convertPeersToProto(baselineInfo.baselinePeers),
			ForbiddenPeers:       convertPeersToProto(baselineInfo.forbiddenPeers),
			ObservationPeriodEnd: baselineInfo.observationPeriodEnd.GogoProtobuf(),
			Locked:               baselineInfo.userLocked,
		})
	}
	return m.ds.UpsertNetworkBaselines(managerCtx, baselines)
}

func (m *manager) processFlowUpdate(flows map[networkgraph.NetworkConnIndicator]timestamp.MicroTS) error {
	modifiedDeploymentIDs := set.NewStringSet()
	for conn, updateTS := range flows {
		if !m.shouldUpdate(&conn, updateTS) {
			continue
		}
		if conn.SrcEntity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
			m.maybeAddPeer(conn.SrcEntity.ID, &peer{
				isIngress: false,
				entity:    conn.DstEntity,
				dstPort:   conn.DstPort,
				protocol:  conn.Protocol,
			}, modifiedDeploymentIDs)
		}
		if conn.DstEntity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
			m.maybeAddPeer(conn.DstEntity.ID, &peer{
				isIngress: true,
				entity:    conn.SrcEntity,
				dstPort:   conn.DstPort,
				protocol:  conn.Protocol,
			}, modifiedDeploymentIDs)
		}
	}
	return m.persistNetworkBaselines(modifiedDeploymentIDs)
}

func (m *manager) processDeploymentCreate(deploymentID, clusterID, namespace string) error {
	if _, exists := m.baselinesByDeploymentID[deploymentID]; exists {
		return nil
	}

	m.baselinesByDeploymentID[deploymentID] = baselineInfo{
		clusterID:            clusterID,
		namespace:            namespace,
		observationPeriodEnd: timestamp.Now().Add(env.NetworkBaselineObservationPeriod.DurationSetting()),
		userLocked:           false,
		baselinePeers:        make(map[peer]struct{}),
		forbiddenPeers:       make(map[peer]struct{}),
	}
	return m.persistNetworkBaselines(set.NewStringSet(deploymentID))
}

func (m *manager) ProcessDeploymentCreate(deploymentID, clusterID, namespace string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.processDeploymentCreate(deploymentID, clusterID, namespace)
}

func (m *manager) ProcessFlowUpdate(flows map[networkgraph.NetworkConnIndicator]timestamp.MicroTS) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.processFlowUpdate(flows)
}

func (m *manager) validatePeers(peers []*v1.NetworkBaselinePeerStatus) error {
	var missingDeploymentIDs []string
	var invalidPeerTypes []string
	for _, p := range peers {
		entity := p.GetPeer().GetEntity()
		if _, valid := validBaselinePeerEntityTypes[entity.GetType()]; !valid {
			invalidPeerTypes = append(invalidPeerTypes, entity.GetType().String())
		}
		if entity.GetType() == storage.NetworkEntityInfo_DEPLOYMENT {
			if _, found := m.baselinesByDeploymentID[entity.GetId()]; !found {
				missingDeploymentIDs = append(missingDeploymentIDs, entity.GetId())
			}
		}
	}
	if len(missingDeploymentIDs) > 0 || len(invalidPeerTypes) > 0 {
		errorList := errorhelpers.NewErrorList("peer validation")
		if len(missingDeploymentIDs) > 0 {
			errorList.AddStringf("no baselines found for deployment IDs %v", missingDeploymentIDs)
		}
		if len(invalidPeerTypes) > 0 {
			errorList.AddStringf("invalid types for peers: %v", invalidPeerTypes)
		}
		return status.Error(codes.InvalidArgument, errorList.String())
	}
	return nil
}

func (m *manager) ProcessBaselineStatusUpdate(ctx context.Context, modifyRequest *v1.ModifyBaselineStatusForPeersRequest) error {
	deploymentID := modifyRequest.GetDeploymentId()
	m.lock.Lock()
	defer m.lock.Unlock()

	baseline, found := m.baselinesByDeploymentID[deploymentID]
	if !found {
		return status.Errorf(codes.InvalidArgument, "no baseline found for deployment id %q", deploymentID)
	}
	if err := m.validatePeers(modifyRequest.GetPeers()); err != nil {
		return err
	}

	// It's not ideal to have to duplicate this check, but we do the permission check here upfront so that we know for sure
	// what the end state of the in-memory data structures should be. Otherwise, if there is a permission denied error,
	// we will need to come back and undo the in-memory changes, which is more complex.
	if ok, err := networkBaselineSAC.WriteAllowed(ctx, sac.ClusterScopeKey(baseline.clusterID), sac.NamespaceScopeKey(baseline.namespace)); err != nil {
		return err
	} else if !ok {
		return status.Error(codes.PermissionDenied, sac.ErrPermissionDenied.Error())
	}

	modifiedDeploymentIDs := set.NewStringSet()
	for _, peerAndStatus := range modifyRequest.GetPeers() {
		peer := peerFromV1Peer(peerAndStatus.GetPeer())
		_, inBaseline := baseline.baselinePeers[peer]
		_, inForbidden := baseline.forbiddenPeers[peer]
		switch peerAndStatus.GetStatus() {
		case v1.NetworkBaselinePeerStatus_BASELINE:
			if inBaseline && !inForbidden {
				// We wouldn't make any modifications in this case.
				continue
			}
			baseline.baselinePeers[peer] = struct{}{}
			delete(baseline.forbiddenPeers, peer)
			modifiedDeploymentIDs.Add(deploymentID)
			if peer.entity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
				reversePeer := reversePeerView(deploymentID, &peer)

				otherBaseline := m.baselinesByDeploymentID[peer.entity.ID]
				otherBaseline.baselinePeers[reversePeer] = struct{}{}
				delete(otherBaseline.forbiddenPeers, reversePeer)
				modifiedDeploymentIDs.Add(peer.entity.ID)
			}
		case v1.NetworkBaselinePeerStatus_ANOMALOUS:
			if !inBaseline && inForbidden {
				// We wouldn't make any modifications in this case.
				continue
			}
			delete(baseline.baselinePeers, peer)
			baseline.forbiddenPeers[peer] = struct{}{}
			modifiedDeploymentIDs.Add(deploymentID)
			if peer.entity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
				reversePeer := reversePeerView(deploymentID, &peer)

				otherBaseline := m.baselinesByDeploymentID[peer.entity.ID]
				delete(otherBaseline.baselinePeers, reversePeer)
				otherBaseline.forbiddenPeers[reversePeer] = struct{}{}
				modifiedDeploymentIDs.Add(peer.entity.ID)
			}
		default:
			return utils.Should(errors.Errorf("unknown status: %v", peerAndStatus.GetStatus()))
		}
	}
	if err := m.persistNetworkBaselines(modifiedDeploymentIDs); err != nil {
		return status.Errorf(codes.Internal, "failed to persist baseline to store: %v", err)
	}
	return nil
}

func (m *manager) initFromStore() error {
	m.baselinesByDeploymentID = make(map[string]baselineInfo)
	return m.ds.Walk(managerCtx, func(baseline *storage.NetworkBaseline) error {
		m.baselinesByDeploymentID[baseline.GetDeploymentId()] = baselineInfo{
			clusterID:            baseline.GetClusterId(),
			namespace:            baseline.GetNamespace(),
			observationPeriodEnd: timestamp.FromProtobuf(baseline.GetObservationPeriodEnd()),
			userLocked:           baseline.GetLocked(),
			baselinePeers:        convertPeersFromProto(baseline.GetPeers()),
			forbiddenPeers:       convertPeersFromProto(baseline.GetForbiddenPeers()),
		}
		return nil
	})
}

// New returns an initialized manager, and starts the manager's processing loop in the background.
func New(ds datastore.DataStore) (Manager, error) {
	m := &manager{
		ds: ds,
	}
	if err := m.initFromStore(); err != nil {
		return nil, err
	}
	return m, nil
}
