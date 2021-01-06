package manager

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
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
	clusterID      string
	namespace      string
	deploymentName string

	observationPeriodEnd timestamp.MicroTS
	userLocked           bool
	baselinePeers        map[peer]struct{}
	forbiddenPeers       map[peer]struct{}
}

type manager struct {
	ds              datastore.DataStore
	networkEntities networkEntityDS.EntityDataStore

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
		peers, err := convertPeersToProto(baselineInfo.baselinePeers)
		if err != nil {
			return err
		}
		forbiddenPeers, err := convertPeersToProto(baselineInfo.forbiddenPeers)
		if err != nil {
			return err
		}
		baselines = append(baselines, &storage.NetworkBaseline{
			DeploymentId:         deploymentID,
			ClusterId:            baselineInfo.clusterID,
			Namespace:            baselineInfo.namespace,
			Peers:                peers,
			ForbiddenPeers:       forbiddenPeers,
			ObservationPeriodEnd: baselineInfo.observationPeriodEnd.GogoProtobuf(),
			Locked:               baselineInfo.userLocked,
			DeploymentName:       baselineInfo.deploymentName,
		})
	}
	return m.ds.UpsertNetworkBaselines(managerCtx, baselines)
}

func (m *manager) lookUpPeerName(entity networkgraph.Entity) string {
	switch entity.Type {
	case storage.NetworkEntityInfo_DEPLOYMENT:
		// If the peer is a deployment, just look it up from the baselines
		peerBaseline, ok := m.baselinesByDeploymentID[entity.ID]
		if !ok {
			// Unexpected but the chance of this happening should be very slim.
			// - created deployment A and B
			// - created baseline for A
			// - add flow called on dep A <====== only happens in this case
			// - created baseline for B
			// Returning an empty string with a log
			log.Warnf("baseline for deployment peer does not exist: %q", entity.ID)
			return ""
		}
		return peerBaseline.deploymentName
	case storage.NetworkEntityInfo_EXTERNAL_SOURCE:
		// Look it up from datastore since as of now the external source name can change without ID changing.
		networkEntity, found, err := m.networkEntities.GetEntity(managerCtx, entity.ID)
		if err != nil {
			log.Warnf("failed to get network entity for its name: %v", err)
			return ""
		}
		if !found {
			// Unexpected. Network entity can only be captured in a flow when it is in the DS
			log.Warnf("network entity peer %q not found", entity.ID)
			return ""
		}
		return networkEntity.GetInfo().GetExternalSource().GetName()
	case storage.NetworkEntityInfo_INTERNET:
		return networkgraph.InternetExternalSourceName
	default:
		// Unsupported type.
		log.Warnf("unsupported entity type in network baseline: %v", entity)
		return ""
	}
}

func (m *manager) processFlowUpdate(flows map[networkgraph.NetworkConnIndicator]timestamp.MicroTS) error {
	modifiedDeploymentIDs := set.NewStringSet()
	for conn, updateTS := range flows {
		if !m.shouldUpdate(&conn, updateTS) {
			continue
		}
		if conn.SrcEntity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
			peerName := m.lookUpPeerName(conn.DstEntity)
			if peerName != "" {
				m.maybeAddPeer(conn.SrcEntity.ID, &peer{
					isIngress: false,
					entity:    conn.DstEntity,
					name:      peerName,
					dstPort:   conn.DstPort,
					protocol:  conn.Protocol,
				}, modifiedDeploymentIDs)
			}
		}
		if conn.DstEntity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
			peerName := m.lookUpPeerName(conn.SrcEntity)
			if peerName != "" {
				m.maybeAddPeer(conn.DstEntity.ID, &peer{
					isIngress: true,
					entity:    conn.SrcEntity,
					name:      peerName,
					dstPort:   conn.DstPort,
					protocol:  conn.Protocol,
				}, modifiedDeploymentIDs)
			}
		}
	}
	return m.persistNetworkBaselines(modifiedDeploymentIDs)
}

func (m *manager) processDeploymentCreate(deploymentID, deploymentName, clusterID, namespace string) error {
	if _, exists := m.baselinesByDeploymentID[deploymentID]; exists {
		return nil
	}

	m.baselinesByDeploymentID[deploymentID] = baselineInfo{
		clusterID:            clusterID,
		namespace:            namespace,
		deploymentName:       deploymentName,
		observationPeriodEnd: timestamp.Now().Add(env.NetworkBaselineObservationPeriod.DurationSetting()),
		userLocked:           false,
		baselinePeers:        make(map[peer]struct{}),
		forbiddenPeers:       make(map[peer]struct{}),
	}
	return m.persistNetworkBaselines(set.NewStringSet(deploymentID))
}

func (m *manager) ProcessDeploymentCreate(deploymentID, deploymentName, clusterID, namespace string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.processDeploymentCreate(deploymentID, deploymentName, clusterID, namespace)
}

func (m *manager) processDeploymentDelete(deploymentID string) error {
	deletingBaseline, found := m.baselinesByDeploymentID[deploymentID]
	if !found {
		// Most likely a repeated call to delete. Just return
		return nil
	}

	modifiedDeployments := set.NewStringSet()
	for peer := range deletingBaseline.baselinePeers {
		// Delete the edge from that deployment to this deployment
		peerBaseline, peerFound := m.baselinesByDeploymentID[peer.entity.ID]
		if !peerFound {
			// Probably the peer is not a deployment
			continue
		}
		reversedPeer := reversePeerView(deploymentID, deletingBaseline.deploymentName, &peer)
		delete(peerBaseline.baselinePeers, reversedPeer)
		modifiedDeployments.Add(peer.entity.ID)
	}
	// For now delete this deployment record from the forbidden peers as well. If we need
	// the records to be sticky for any reason, remove the following lines
	for forbiddenPeer := range deletingBaseline.forbiddenPeers {
		forbiddenPeerBaseline, found := m.baselinesByDeploymentID[forbiddenPeer.entity.ID]
		if !found {
			// Probably the forbidden peer is not a deployment
			continue
		}
		reversedPeer := reversePeerView(deploymentID, deletingBaseline.deploymentName, &forbiddenPeer)
		delete(forbiddenPeerBaseline.forbiddenPeers, reversedPeer)
		modifiedDeployments.Add(forbiddenPeer.entity.ID)
	}

	// Delete the records from other baselines first, then delete the wanted baseline after
	err := m.persistNetworkBaselines(modifiedDeployments)
	if err != nil {
		return errors.Wrapf(err, "deleting baseline of deployment %q", deletingBaseline.deploymentName)
	}

	err = m.ds.DeleteNetworkBaseline(managerCtx, deploymentID)
	if err != nil {
		return errors.Wrapf(err, "deleting baseline of deployment %q", deletingBaseline.deploymentName)
	}

	// Clean up cache
	delete(m.baselinesByDeploymentID, deploymentID)
	return nil
}

func (m *manager) ProcessDeploymentDelete(deploymentID string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.processDeploymentDelete(deploymentID)
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
				reversePeer := reversePeerView(deploymentID, baseline.deploymentName, &peer)

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
				reversePeer := reversePeerView(deploymentID, baseline.deploymentName, &peer)

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
		peers, err := convertPeersFromProto(baseline.GetPeers())
		if err != nil {
			return err
		}
		forbiddenPeers, err := convertPeersFromProto(baseline.GetForbiddenPeers())
		if err != nil {
			return err
		}
		m.baselinesByDeploymentID[baseline.GetDeploymentId()] = baselineInfo{
			clusterID:            baseline.GetClusterId(),
			namespace:            baseline.GetNamespace(),
			deploymentName:       baseline.GetDeploymentName(),
			observationPeriodEnd: timestamp.FromProtobuf(baseline.GetObservationPeriodEnd()),
			userLocked:           baseline.GetLocked(),
			baselinePeers:        peers,
			forbiddenPeers:       forbiddenPeers,
		}

		return nil
	})
}

// New returns an initialized manager, and starts the manager's processing loop in the background.
func New(ds datastore.DataStore, networkEntities networkEntityDS.EntityDataStore) (Manager, error) {
	m := &manager{
		ds:              ds,
		networkEntities: networkEntities,
	}
	if err := m.initFromStore(); err != nil {
		return nil, err
	}
	return m, nil
}
