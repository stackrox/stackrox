package manager

import (
	"context"

	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/networkbaseline"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	managerCtx = sac.WithAllAccess(context.Background())

	networkBaselineSAC = sac.ForResource(resources.NetworkBaseline)

	log = logging.LoggerForModule()
)

type manager struct {
	ds                datastore.DataStore
	networkEntities   networkEntityDS.EntityDataStore
	deploymentDS      deploymentDS.DataStore
	networkPolicyDS   networkPolicyDS.DataStore
	connectionManager connection.Manager

	baselinesByDeploymentID map[string]*networkbaseline.BaselineInfo
	seenNetworkPolicies     set.Uint64Set
	lock                    sync.Mutex
}

func getNewObservationPeriodEnd() timestamp.MicroTS {
	return timestamp.Now().Add(env.NetworkBaselineObservationPeriod.DurationSetting())
}

func (m *manager) shouldUpdate(conn *networkgraph.NetworkConnIndicator, updateTS timestamp.MicroTS) bool {
	var atLeastOneBaselineInObservationPeriod bool
	for _, entity := range []*networkgraph.Entity{&conn.SrcEntity, &conn.DstEntity} {
		if _, valid := networkbaseline.ValidBaselinePeerEntityTypes[entity.Type]; !valid {
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
		if baselineInfo.UserLocked {
			return false
		}
		if baselineInfo.ObservationPeriodEnd.After(updateTS) {
			atLeastOneBaselineInObservationPeriod = true
		}
	}
	return atLeastOneBaselineInObservationPeriod
}

func (m *manager) maybeAddPeer(deploymentID string, p *networkbaseline.Peer, modifiedDeploymentIDs set.StringSet) {
	_, isForbidden := m.baselinesByDeploymentID[deploymentID].ForbiddenPeers[*p]
	if isForbidden {
		return
	}
	_, alreadyInBaseline := m.baselinesByDeploymentID[deploymentID].BaselinePeers[*p]
	if alreadyInBaseline {
		return
	}
	m.baselinesByDeploymentID[deploymentID].BaselinePeers[*p] = struct{}{}
	modifiedDeploymentIDs.Add(deploymentID)
}

func (m *manager) persistNetworkBaselines(deploymentIDs set.StringSet, baselinesUnlocked set.StringSet) error {
	if len(deploymentIDs) == 0 {
		return nil
	}
	baselines := make([]*storage.NetworkBaseline, 0, len(deploymentIDs))
	for deploymentID := range deploymentIDs {
		baselineInfo := m.baselinesByDeploymentID[deploymentID]
		peers, err := networkbaseline.ConvertPeersToProto(baselineInfo.BaselinePeers)
		if err != nil {
			return err
		}
		forbiddenPeers, err := networkbaseline.ConvertPeersToProto(baselineInfo.ForbiddenPeers)
		if err != nil {
			return err
		}
		baselines = append(baselines, &storage.NetworkBaseline{
			DeploymentId:         deploymentID,
			ClusterId:            baselineInfo.ClusterID,
			Namespace:            baselineInfo.Namespace,
			Peers:                peers,
			ForbiddenPeers:       forbiddenPeers,
			ObservationPeriodEnd: baselineInfo.ObservationPeriodEnd.GogoProtobuf(),
			Locked:               baselineInfo.UserLocked,
			DeploymentName:       baselineInfo.DeploymentName,
		})
	}
	err := m.ds.UpsertNetworkBaselines(managerCtx, baselines)
	if err != nil {
		return errors.Wrap(err, "upserting network baselines in manager")
	}
	m.sendNetworkBaselinesToSensor(baselines, baselinesUnlocked)
	return nil
}

func (m *manager) sendNetworkBaselinesToSensor(baselines []*storage.NetworkBaseline, baselinesUnlocked set.StringSet) {
	// First map baselines by clusters
	clusterIDToBaselines := make(map[string][]*storage.NetworkBaseline, len(baselines))
	for _, b := range baselines {
		// Don't sync baselines if they were in the unlocked state before persist.
		if b.GetLocked() || baselinesUnlocked.Contains(b.GetDeploymentId()) {
			clusterIDToBaselines[b.GetClusterId()] = append(clusterIDToBaselines[b.GetClusterId()], b)
		}
	}
	for clusterID, clusterBaselines := range clusterIDToBaselines {
		err := m.connectionManager.SendMessage(clusterID, &central.MsgToSensor{
			Msg: &central.MsgToSensor_NetworkBaselineSync{
				NetworkBaselineSync: &central.NetworkBaselineSync{
					NetworkBaselines: clusterBaselines,
				},
			},
		})
		if err != nil {
			log.Errorf("error sending network baselines to cluster %q: %v", clusterID, err)
		}
	}
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
		return peerBaseline.DeploymentName
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
				m.maybeAddPeer(conn.SrcEntity.ID, &networkbaseline.Peer{
					IsIngress: false,
					Entity:    conn.DstEntity,
					Name:      peerName,
					DstPort:   conn.DstPort,
					Protocol:  conn.Protocol,
				}, modifiedDeploymentIDs)
			}
		}
		if conn.DstEntity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
			peerName := m.lookUpPeerName(conn.SrcEntity)
			if peerName != "" {
				m.maybeAddPeer(conn.DstEntity.ID, &networkbaseline.Peer{
					IsIngress: true,
					Entity:    conn.SrcEntity,
					Name:      peerName,
					DstPort:   conn.DstPort,
					Protocol:  conn.Protocol,
				}, modifiedDeploymentIDs)
			}
		}
	}
	return m.persistNetworkBaselines(modifiedDeploymentIDs, nil)
}

func (m *manager) processDeploymentCreate(deploymentID, deploymentName, clusterID, namespace string) error {
	if _, exists := m.baselinesByDeploymentID[deploymentID]; exists {
		return nil
	}

	m.baselinesByDeploymentID[deploymentID] = &networkbaseline.BaselineInfo{
		ClusterID:            clusterID,
		Namespace:            namespace,
		DeploymentName:       deploymentName,
		ObservationPeriodEnd: getNewObservationPeriodEnd(),
		UserLocked:           false,
		BaselinePeers:        make(map[networkbaseline.Peer]struct{}),
		ForbiddenPeers:       make(map[networkbaseline.Peer]struct{}),
	}
	return m.persistNetworkBaselines(set.NewStringSet(deploymentID), nil)
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
	for peer := range deletingBaseline.BaselinePeers {
		// Delete the edge from that deployment to this deployment
		peerBaseline, peerFound := m.baselinesByDeploymentID[peer.Entity.ID]
		if !peerFound {
			// Probably the peer is not a deployment
			continue
		}
		reversedPeer := networkbaseline.ReversePeerView(deploymentID, deletingBaseline.DeploymentName, &peer)
		delete(peerBaseline.BaselinePeers, reversedPeer)
		modifiedDeployments.Add(peer.Entity.ID)
	}
	// For now delete this deployment record from the forbidden peers as well. If we need
	// the records to be sticky for any reason, remove the following lines
	for forbiddenPeer := range deletingBaseline.ForbiddenPeers {
		forbiddenPeerBaseline, found := m.baselinesByDeploymentID[forbiddenPeer.Entity.ID]
		if !found {
			// Probably the forbidden peer is not a deployment
			continue
		}
		reversedPeer := networkbaseline.ReversePeerView(deploymentID, deletingBaseline.DeploymentName, &forbiddenPeer)
		delete(forbiddenPeerBaseline.ForbiddenPeers, reversedPeer)
		modifiedDeployments.Add(forbiddenPeer.Entity.ID)
	}

	// Delete the records from other baselines first, then delete the wanted baseline after
	err := m.persistNetworkBaselines(modifiedDeployments, nil)
	if err != nil {
		return errors.Wrapf(err, "deleting baseline of deployment %q", deletingBaseline.DeploymentName)
	}

	err = m.ds.DeleteNetworkBaseline(managerCtx, deploymentID)
	if err != nil {
		return errors.Wrapf(err, "deleting baseline of deployment %q", deletingBaseline.DeploymentName)
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
		if _, valid := networkbaseline.ValidBaselinePeerEntityTypes[entity.GetType()]; !valid {
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
		return errors.Wrap(errox.InvalidArgs, errorList.String())
	}
	return nil
}

func (m *manager) ProcessBaselineStatusUpdate(ctx context.Context, modifyRequest *v1.ModifyBaselineStatusForPeersRequest) error {
	deploymentID := modifyRequest.GetDeploymentId()
	m.lock.Lock()
	defer m.lock.Unlock()

	baseline, found := m.baselinesByDeploymentID[deploymentID]
	if !found {
		return errors.Wrapf(errox.InvalidArgs, "no baseline found for deployment id %q", deploymentID)
	}
	if err := m.validatePeers(modifyRequest.GetPeers()); err != nil {
		return err
	}

	// It's not ideal to have to duplicate this check, but we do the permission check here upfront so that we know for sure
	// what the end state of the in-memory data structures should be. Otherwise, if there is a permission denied error,
	// we will need to come back and undo the in-memory changes, which is more complex.
	if ok, err := networkBaselineSAC.WriteAllowed(ctx, sac.ClusterScopeKey(baseline.ClusterID), sac.NamespaceScopeKey(baseline.Namespace)); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	modifiedDeploymentIDs := set.NewStringSet()
	for _, peerAndStatus := range modifyRequest.GetPeers() {
		v1Peer := peerAndStatus.GetPeer()
		peerName := m.lookUpPeerName(
			networkgraph.Entity{
				Type: v1Peer.GetEntity().GetType(),
				ID:   v1Peer.GetEntity().GetId(),
			})
		peer := networkbaseline.PeerFromV1Peer(v1Peer, peerName)
		_, inBaseline := baseline.BaselinePeers[peer]
		_, inForbidden := baseline.ForbiddenPeers[peer]
		switch peerAndStatus.GetStatus() {
		case v1.NetworkBaselinePeerStatus_BASELINE:
			if inBaseline && !inForbidden {
				// We wouldn't make any modifications in this case.
				continue
			}
			baseline.BaselinePeers[peer] = struct{}{}
			delete(baseline.ForbiddenPeers, peer)
			modifiedDeploymentIDs.Add(deploymentID)
			if peer.Entity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
				reversePeer := networkbaseline.ReversePeerView(deploymentID, baseline.DeploymentName, &peer)

				otherBaseline := m.baselinesByDeploymentID[peer.Entity.ID]
				otherBaseline.BaselinePeers[reversePeer] = struct{}{}
				delete(otherBaseline.ForbiddenPeers, reversePeer)
				modifiedDeploymentIDs.Add(peer.Entity.ID)
			}
		case v1.NetworkBaselinePeerStatus_ANOMALOUS:
			if !inBaseline && inForbidden {
				// We wouldn't make any modifications in this case.
				continue
			}
			delete(baseline.BaselinePeers, peer)
			baseline.ForbiddenPeers[peer] = struct{}{}
			modifiedDeploymentIDs.Add(deploymentID)
			if peer.Entity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
				reversePeer := networkbaseline.ReversePeerView(deploymentID, baseline.DeploymentName, &peer)

				otherBaseline := m.baselinesByDeploymentID[peer.Entity.ID]
				delete(otherBaseline.BaselinePeers, reversePeer)
				otherBaseline.ForbiddenPeers[reversePeer] = struct{}{}
				modifiedDeploymentIDs.Add(peer.Entity.ID)
			}
		default:
			return utils.Should(errors.Errorf("unknown status: %v", peerAndStatus.GetStatus()))
		}
	}
	if err := m.persistNetworkBaselines(modifiedDeploymentIDs, nil); err != nil {
		return errors.Errorf("failed to persist baseline to store: %v", err)
	}
	return nil
}

func (m *manager) processNetworkPolicyUpdate(
	ctx context.Context,
	action central.ResourceAction,
	policy *storage.NetworkPolicy,
) error {
	shouldIgnore, hash, err := m.shouldIgnoreNetworkPolicy(action, policy)
	if err != nil || shouldIgnore {
		return err
	}

	// This is a network policy we have not yet processed before. Process it.
	// First get all the relevant deployments
	deploymentIDs, err := m.getDeploymentIDsAffectedByNetworkPolicy(ctx, policy)
	if err != nil {
		return err
	}

	// For each of the affected deployments, reset the corresponding baseline's observation period
	modifiedDeploymentIDs := set.NewStringSet()
	for _, deploymentID := range deploymentIDs {
		baseline, found := m.baselinesByDeploymentID[deploymentID]
		if !found {
			// Maybe somehow the network policy update came first before the deployment create event.
			// In this case do nothing and trust that the deployment create flow should just
			// take care of setting the observation period
			continue
		}
		baseline.ObservationPeriodEnd = getNewObservationPeriodEnd()
		modifiedDeploymentIDs.Add(deploymentID)
	}

	err = m.persistNetworkBaselines(modifiedDeploymentIDs, nil)
	if err != nil {
		return err
	}
	// After everything, persist the hash of this network policy
	m.seenNetworkPolicies.Add(hash)
	return nil
}

func (m *manager) getDeploymentIDsAffectedByNetworkPolicy(
	ctx context.Context,
	policy *storage.NetworkPolicy,
) ([]string, error) {
	deploymentQuery :=
		search.
			NewQueryBuilder().
			AddExactMatches(search.Namespace, policy.GetNamespace()).
			AddExactMatches(search.ClusterID, policy.GetClusterId()).
			ProtoQuery()
	deploymentsInSameClusterAndNamespace, err := m.deploymentDS.SearchRawDeployments(managerCtx, deploymentQuery)
	if err != nil {
		return nil, err
	}

	// Filter out the deployments that we don't want. aka check the policy's pod selectors and
	// namespace selectors.
	var result []string
	for _, deployment := range deploymentsInSameClusterAndNamespace {
		if m.isDeploymentAffectedByNetworkPolicy(deployment, policy) {
			result = append(result, deployment.GetId())
		}
	}

	return result, nil
}

func (m *manager) isDeploymentAffectedByNetworkPolicy(
	deployment *storage.Deployment,
	policy *storage.NetworkPolicy,
) bool {
	// Check if policy is specified for this deployment
	// NOTE: we do not need to look into the ingress/egress rules of the policy since
	//       as long as one side of the connection is still within the observation period,
	//       the other side's network baseline also gets updated when we update the baseline
	//       for this side.
	return deployment.GetNamespace() == policy.GetNamespace() &&
		deployment.GetClusterId() == policy.GetClusterId() &&
		labels.MatchLabels(policy.GetSpec().GetPodSelector(), deployment.GetPodLabels())
}

func (m *manager) ProcessNetworkPolicyUpdate(
	ctx context.Context,
	action central.ResourceAction,
	policy *storage.NetworkPolicy,
) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.processNetworkPolicyUpdate(ctx, action, policy)
}

type clusterNamespacePair struct {
	ClusterID string
	Namespace string
}

func (m *manager) processBaselineLockUpdate(ctx context.Context, deploymentID string, lockBaseline bool) error {
	baseline, found := m.baselinesByDeploymentID[deploymentID]
	if !found {
		return errors.Wrap(errox.InvalidArgs, "no baseline with given deployment ID found")
	}
	// Permission check before modifying in-memory data structures
	if ok, err := networkBaselineSAC.WriteAllowed(ctx, sac.ClusterScopeKey(baseline.ClusterID), sac.NamespaceScopeKey(baseline.Namespace)); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// No error if already locked/unlocked
	if baseline.UserLocked == lockBaseline {
		// Already in the state which user specifies
		return nil
	}
	var baselinesUnlocked set.StringSet
	if baseline.UserLocked && !lockBaseline {
		// Baseline is currently locked but we are unlocking it. Need to sync to sensor
		baselinesUnlocked = set.NewStringSet(deploymentID)
	}

	baseline.UserLocked = lockBaseline
	return m.persistNetworkBaselines(set.NewStringSet(deploymentID), baselinesUnlocked)
}

func (m *manager) ProcessBaselineLockUpdate(ctx context.Context, deploymentID string, lockBaseline bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.processBaselineLockUpdate(ctx, deploymentID, lockBaseline)
}

func (m *manager) processPostClusterDelete(clusterID string) error {
	deletingBaselines := set.NewStringSet()
	for deploymentID, baseline := range m.baselinesByDeploymentID {
		if baseline.ClusterID == clusterID {
			deletingBaselines.Add(deploymentID)
		}
	}

	// Clean up edges in other baselines
	modifiedBaselines := set.NewStringSet()
	for deploymentID, baseline := range m.baselinesByDeploymentID {
		if deletingBaselines.Contains(deploymentID) {
			continue
		}
		// Baselines that are not deleted. Need to update their edges in case
		// they are pointing to the deleting baselines.
		for p := range baseline.BaselinePeers {
			if deletingBaselines.Contains(p.Entity.ID) {
				delete(baseline.BaselinePeers, p)
				modifiedBaselines.Add(deploymentID)
			}
		}
		for forbiddenP := range baseline.ForbiddenPeers {
			if deletingBaselines.Contains(forbiddenP.Entity.ID) {
				delete(baseline.ForbiddenPeers, forbiddenP)
				modifiedBaselines.Add(deploymentID)
			}
		}
	}

	// Update the edges of other baselines first
	err := m.persistNetworkBaselines(modifiedBaselines, nil)
	if err != nil {
		return err
	}

	// Delete the baselines
	err = m.ds.DeleteNetworkBaselines(managerCtx, deletingBaselines.AsSlice())
	if err != nil {
		return err
	}
	// Delete from cache
	for deploymentID := range deletingBaselines {
		delete(m.baselinesByDeploymentID, deploymentID)
	}
	return nil
}

func (m *manager) ProcessPostClusterDelete(clusterID string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.processPostClusterDelete(clusterID)
}

func (m *manager) initFromStore() error {
	seenClusterAndNamespace := make(map[clusterNamespacePair]struct{})
	m.baselinesByDeploymentID = make(map[string]*networkbaseline.BaselineInfo)
	return m.ds.Walk(managerCtx, func(baseline *storage.NetworkBaseline) error {
		baselineInfo, err := networkbaseline.ConvertBaselineInfoFromProto(baseline)
		if err != nil {
			return err
		}
		m.baselinesByDeploymentID[baseline.GetDeploymentId()] = baselineInfo

		// Try loading all the network policies to build the seen network policies cache
		curPair := clusterNamespacePair{ClusterID: baseline.GetClusterId(), Namespace: baseline.GetNamespace()}
		if _, ok := seenClusterAndNamespace[curPair]; !ok {
			// Mark seen
			seenClusterAndNamespace[curPair] = struct{}{}

			policies, err := m.networkPolicyDS.GetNetworkPolicies(managerCtx, baseline.ClusterId, baseline.Namespace)
			if err != nil {
				return err
			}
			for _, policy := range policies {
				// On start treat all policies as have just been created.
				hash, err := m.getHashOfNetworkPolicyWithResourceAction(central.ResourceAction_CREATE_RESOURCE, policy)
				if err != nil {
					return err
				}
				m.seenNetworkPolicies.Add(hash)
			}
		}

		return nil
	})
}

// New returns an initialized manager, and starts the manager's processing loop in the background.
func New(
	ds datastore.DataStore,
	networkEntities networkEntityDS.EntityDataStore,
	deploymentDS deploymentDS.DataStore,
	networkPolicyDS networkPolicyDS.DataStore,
	connectionManager connection.Manager,
) (Manager, error) {
	m := &manager{
		ds:                  ds,
		networkEntities:     networkEntities,
		deploymentDS:        deploymentDS,
		networkPolicyDS:     networkPolicyDS,
		connectionManager:   connectionManager,
		seenNetworkPolicies: set.NewUint64Set(),
	}
	if err := m.initFromStore(); err != nil {
		return nil, err
	}
	return m, nil
}
