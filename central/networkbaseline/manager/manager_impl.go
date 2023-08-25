package manager

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/queue"
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	networkFlowDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
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
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	baselineFlushTickerDuration = 5 * time.Second
)

var (
	managerCtx = sac.WithAllAccess(context.Background())

	deploymentExtensionSAC = sac.ForResource(resources.DeploymentExtension)

	log = logging.LoggerForModule()

	observationDuration = env.NetworkBaselineObservationPeriod.DurationSetting()
)

type manager struct {
	ds                datastore.DataStore
	networkEntities   networkEntityDS.EntityDataStore
	deploymentDS      deploymentDS.DataStore
	networkPolicyDS   networkPolicyDS.DataStore
	clusterFlows      networkFlowDS.ClusterDataStore
	connectionManager connection.Manager

	baselinesByDeploymentID map[string]*networkbaseline.BaselineInfo
	seenNetworkPolicies     set.Set[uint64]
	lock                    sync.Mutex

	deploymentObservationQueue queue.DeploymentObservationQueue
	baselineFlushTicker        *time.Ticker
}

func getNewObservationPeriodEnd() timestamp.MicroTS {
	return timestamp.Now().Add(observationDuration)
}

// shouldUpdate -- looks at the baselines and flows to determine if the flow should be added to the baseline.
func (m *manager) shouldUpdate(conn *networkgraph.NetworkConnIndicator, updateTS timestamp.MicroTS, initialLoad bool) bool {
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
		// This case occurs when the deployment has been deleted OR the deployment is being controlled by the
		// observation queue and as such we have not created a baseline for it yet.
		// We can avoid processing this flow altogether at this time.
		if !found || baselineInfo == nil {
			return false
		}

		// If even one baseline is user-locked, no updates based on flows.
		if baselineInfo.UserLocked {
			return false
		}

		// It is possible that the last time stamp on the flow is nil if the connection is initial and still open
		// In those cases updateTS will be 0 because that is the nil value of a MicroTS.  So all flows in such a state
		// would think the baseline is out of observation or not when we compare the time as we do below.  To resolve
		// these cases we compare to now if the timestamp is 0 to ensure we don't add a flow to a baseline that is
		// no longer being observed.
		compareTime := updateTS
		if compareTime == 0 {
			compareTime = timestamp.Now()
		}

		// If the last time the flow was seen is nil, then updateTS will be 0 and thus
		// the baseline will always report being in observation.
		// Additionally, we could be in an initial load state where Deployment Observation expired
		// OR user requested a baseline; so we shouldUpdate the deployments involved.
		// Additionally, it is possible that by the time we get the flow from sensor that the observation window
		// has ended.  So we still need to compare based on the time of the flow vs just checking the observation queue.
		if baselineInfo.ObservationPeriodEnd.After(compareTime) || initialLoad {
			atLeastOneBaselineInObservationPeriod = true
		}
	}
	return atLeastOneBaselineInObservationPeriod
}

func (m *manager) maybeAddPeer(deploymentID string, p *networkbaseline.Peer, modifiedDeploymentIDs set.StringSet) {
	if baseline, found := m.baselinesByDeploymentID[deploymentID]; !found || baseline == nil {
		return
	}

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
		if baselineInfo == nil {
			continue
		}

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

type peerInfo struct {
	name      string
	cidrBlock string
}

func (m *manager) lookUpPeerInfo(entity networkgraph.Entity) peerInfo {
	switch entity.Type {
	case storage.NetworkEntityInfo_DEPLOYMENT:
		// If the peer is a deployment, just look it up from the baselines
		peerBaseline, ok := m.baselinesByDeploymentID[entity.ID]
		if !ok || peerBaseline == nil {
			// Unexpected but the chance of this happening should be very slim.
			// - created deployment A and B
			// - created baseline for A
			// - add flow called on dep A <====== only happens in this case
			// - created baseline for B
			// Returning an empty string with a log
			log.Warnf("baseline for deployment peer does not exist: %q", entity.ID)
			return peerInfo{}
		}
		return peerInfo{name: peerBaseline.DeploymentName}
	case storage.NetworkEntityInfo_EXTERNAL_SOURCE:
		// Look it up from datastore since as of now the external source name can change without ID changing.
		networkEntity, found, err := m.networkEntities.GetEntity(managerCtx, entity.ID)
		if err != nil {
			log.Warnf("failed to get network entity for its name: %v", err)
			return peerInfo{}
		}
		if !found {
			// Unexpected. Network entity can only be captured in a flow when it is in the DS
			log.Warnf("network entity peer %q not found", entity.ID)
			return peerInfo{}
		}
		externalSource := networkEntity.GetInfo().GetExternalSource()
		return peerInfo{
			name:      externalSource.GetName(),
			cidrBlock: externalSource.GetCidr(),
		}
	case storage.NetworkEntityInfo_INTERNET:
		return peerInfo{name: networkgraph.InternetExternalSourceName}
	default:
		// Unsupported type.
		log.Warnf("unsupported entity type in network baseline: %v", entity)
		return peerInfo{}
	}
}

func (m *manager) processFlowUpdate(flows map[networkgraph.NetworkConnIndicator]timestamp.MicroTS, initialLoad bool) (set.StringSet, error) {
	modifiedDeploymentIDs := set.NewStringSet()
	for conn, updateTS := range flows {
		if !m.shouldUpdate(&conn, updateTS, initialLoad) {
			continue
		}
		if conn.SrcEntity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
			peer := m.lookUpPeerInfo(conn.DstEntity)
			if peer.name != "" {
				m.maybeAddPeer(conn.SrcEntity.ID, &networkbaseline.Peer{
					IsIngress: false,
					Entity:    conn.DstEntity,
					Name:      peer.name,
					DstPort:   conn.DstPort,
					Protocol:  conn.Protocol,
					CidrBlock: peer.cidrBlock,
				}, modifiedDeploymentIDs)
			}
		}
		if conn.DstEntity.Type == storage.NetworkEntityInfo_DEPLOYMENT {
			peer := m.lookUpPeerInfo(conn.SrcEntity)
			if peer.name != "" {
				m.maybeAddPeer(conn.DstEntity.ID, &networkbaseline.Peer{
					IsIngress: true,
					Entity:    conn.SrcEntity,
					Name:      peer.name,
					DstPort:   conn.DstPort,
					Protocol:  conn.Protocol,
				}, modifiedDeploymentIDs)
			}
		}
	}

	err := m.persistNetworkBaselines(modifiedDeploymentIDs, nil)
	if err != nil {
		return nil, err
	}
	return modifiedDeploymentIDs, nil
}

func (m *manager) processDeploymentCreate(deploymentID, _ string) error {
	// Deployment has already had a baseline created.  Nothing to do in this case.
	if _, exists := m.baselinesByDeploymentID[deploymentID]; exists {
		return nil
	}

	// We don't want to process the deployment until the observation window expires OR the user requests a
	// baseline.  But the cache needs to know the deployment exists.  So map this deployment to nil.
	m.baselinesByDeploymentID[deploymentID] = nil

	// Push the new deployment on to the observation queue.  When Observation ends, flows for this deployment
	// will be pulled and placed into a baseline.
	m.deploymentObservationQueue.Push(
		&queue.DeploymentObservation{
			DeploymentID:   deploymentID,
			InObservation:  true,
			ObservationEnd: getNewObservationPeriodEnd().GogoProtobuf(),
		})

	return nil
}

func (m *manager) ProcessDeploymentCreate(deploymentID, _, clusterID, _ string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.processDeploymentCreate(deploymentID, clusterID)
}

func (m *manager) deleteDeploymentFromBaselines(deploymentID string) error {
	modifiedDeployments := set.NewStringSet()
	for id, baseline := range m.baselinesByDeploymentID {
		if ok, peer := baseline.GetPeer(deploymentID); ok {
			delete(baseline.BaselinePeers, peer)
			modifiedDeployments.Add(id)
		}

		if ok, forbiddenPeer := baseline.GetForbiddenPeer(deploymentID); ok {
			delete(baseline.ForbiddenPeers, forbiddenPeer)
			modifiedDeployments.Add(id)
		}
	}

	if err := m.persistNetworkBaselines(modifiedDeployments, nil); err != nil {
		return errors.Wrapf(err, "deleting baseline of deployment ID %s", deploymentID)
	}

	return nil
}

func (m *manager) processDeploymentDelete(deploymentID string) error {
	deletingBaseline, found := m.baselinesByDeploymentID[deploymentID]
	if !found {
		// Most likely a repeated call to delete. Just return
		return nil
	}

	// If baseline for deleting deploynment exists, then we should look at all entries in its baseline
	// in order to the delete its reference from peer baselines.
	if deletingBaseline != nil {
		modifiedDeployments := set.NewStringSet()
		for peer := range deletingBaseline.BaselinePeers {
			// Delete the edge from that deployment to this deployment
			peerBaseline, peerFound := m.baselinesByDeploymentID[peer.Entity.ID]
			if !peerFound || peerBaseline == nil {
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
			if !found || forbiddenPeerBaseline == nil {
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
	} else {
		// If baseline does not exist yet, it could still be that this deployment is already present in some other
		// deployment's baseline. So we need to manually look through all the baselines
		if err := m.deleteDeploymentFromBaselines(deploymentID); err != nil {
			return err
		}
	}

	err := m.ds.DeleteNetworkBaseline(managerCtx, deploymentID)
	if err != nil {
		return errors.Wrapf(err, "deleting baseline of deployment %q", deploymentID)
	}

	// Clean up cache
	delete(m.baselinesByDeploymentID, deploymentID)

	// Remove the deployment from the observation queue
	m.deploymentObservationQueue.RemoveDeployment(deploymentID)

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

	_, err := m.processFlowUpdate(flows, false)
	return err
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
	if !found || baseline == nil {
		return errors.Wrapf(errox.InvalidArgs, "no baseline found for deployment id %q", deploymentID)
	}
	if err := m.validatePeers(modifyRequest.GetPeers()); err != nil {
		return err
	}

	// It's not ideal to have to duplicate this check, but we do the permission check here upfront so that we know for sure
	// what the end state of the in-memory data structures should be. Otherwise, if there is a permission denied error,
	// we will need to come back and undo the in-memory changes, which is more complex.
	if ok, err := deploymentExtensionSAC.WriteAllowed(ctx, sac.ClusterScopeKey(baseline.ClusterID), sac.NamespaceScopeKey(baseline.Namespace)); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	modifiedDeploymentIDs := set.NewStringSet()
	for _, peerAndStatus := range modifyRequest.GetPeers() {
		v1Peer := peerAndStatus.GetPeer()
		info := m.lookUpPeerInfo(
			networkgraph.Entity{
				Type: v1Peer.GetEntity().GetType(),
				ID:   v1Peer.GetEntity().GetId(),
			})
		peer := networkbaseline.PeerFromV1Peer(v1Peer, info.name, info.cidrBlock)
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
				if otherBaseline != nil {
					otherBaseline.BaselinePeers[reversePeer] = struct{}{}
					delete(otherBaseline.ForbiddenPeers, reversePeer)
					modifiedDeploymentIDs.Add(peer.Entity.ID)
				}
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
				if otherBaseline != nil {
					delete(otherBaseline.BaselinePeers, reversePeer)
					otherBaseline.ForbiddenPeers[reversePeer] = struct{}{}
					modifiedDeploymentIDs.Add(peer.Entity.ID)
				}
			}
		default:
			return utils.ShouldErr(errors.Errorf("unknown status: %v", peerAndStatus.GetStatus()))
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
	deployments, err := m.getDeploymentIDsAffectedByNetworkPolicy(ctx, policy)
	if err != nil {
		return err
	}

	// For each of the affected deployments, reset the corresponding baseline's observation period
	modifiedDeploymentIDs := set.NewStringSet()
	for _, deployment := range deployments {
		baseline, found := m.baselinesByDeploymentID[deployment.GetId()]

		if !found {
			// Maybe somehow the network policy update came first before the deployment create event.
			// In this case do nothing and trust that the deployment create flow should just
			// take care of setting the observation period.  This could also occur if the deployment was
			// still in observation without the user having requested a baseline.  Thus we account for this
			// by updating the observation period in the observation queue above.
			continue
		}

		// Regardless of whether the baseline has already been created or not, we need to update the observation
		// period of the observation queue.  This could entail putting the object back on the list.
		newObservationPeriodEnd := getNewObservationPeriodEnd()

		// If the baseline is nil then we have seen the deployment but haven't yet processed it from the observation
		// queue, so move it to the back of the observation queue.
		if baseline == nil {
			m.deploymentObservationQueue.PutBackInObservation(
				&queue.DeploymentObservation{
					DeploymentID:   deployment.GetId(),
					InObservation:  true,
					ObservationEnd: newObservationPeriodEnd.GogoProtobuf(),
				})
		} else {
			baseline.ObservationPeriodEnd = newObservationPeriodEnd
		}

		modifiedDeploymentIDs.Add(deployment.GetId())
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
	_ context.Context,
	policy *storage.NetworkPolicy,
) ([]*storage.Deployment, error) {
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
	var result []*storage.Deployment
	for _, deployment := range deploymentsInSameClusterAndNamespace {
		if m.isDeploymentAffectedByNetworkPolicy(deployment, policy) {
			result = append(result, deployment)
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
	if !found || baseline == nil {
		return errors.Wrap(errox.InvalidArgs, "no baseline with given deployment ID found")
	}
	// Permission check before modifying in-memory data structures
	if ok, err := deploymentExtensionSAC.WriteAllowed(ctx, sac.ClusterScopeKey(baseline.ClusterID), sac.NamespaceScopeKey(baseline.Namespace)); err != nil {
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

func (m *manager) processPostClusterDelete(deploymentIDs []string) error {
	// NOTE:  This process could have invoked processDeploymentDelete repeatedly,
	// but for doing deletes in bulk it is more efficient to do the work in
	// the map than reach out to the database like that function does.

	// Putting in a set to make it easy to check if the various peers are
	// also being deleted.
	deletingBaselines := set.NewStringSet(deploymentIDs...)

	// Clean up edges in other baselines
	modifiedBaselines := set.NewStringSet()
	for deploymentID, baseline := range m.baselinesByDeploymentID {
		// We need to delete any peers that reference deleted deployments.
		// If we are deleting the baseline we are looking at OR if we are
		// looking at a baseline that has not been processed yet, we do not
		// need to look for peers.
		if deletingBaselines.Contains(deploymentID) || baseline == nil {
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
		// Remove from the observation queue
		m.deploymentObservationQueue.RemoveDeployment(deploymentID)
	}

	return nil
}

func (m *manager) ProcessPostClusterDelete(deploymentIDs []string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.processPostClusterDelete(deploymentIDs)
}

func (m *manager) initFromStore() error {
	walkFn := func() error {
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
	return pgutils.RetryIfPostgres(walkFn)
}

func (m *manager) flushBaselineQueue() {
	for {
		// ObservationEnd is in the future so we have nothing to do at this time
		head := m.deploymentObservationQueue.Peek()
		if head == nil || protoutils.After(head.ObservationEnd, types.TimestampNow()) {
			return
		}

		// Grab the first deployment to baseline.
		// NOTE:  This is the only place from which Pull is being called.
		observedDep := m.deploymentObservationQueue.Pull()

		// Get the details about the deployment.
		deployment, exists, err := m.deploymentDS.GetDeployment(managerCtx, observedDep.DeploymentID)
		if !exists {
			log.Error(errors.Wrapf(errox.NotFound, "deployment with id %q does not exist", observedDep.DeploymentID))
			continue
		}
		if err != nil {
			log.Error(err)
			continue
		}

		err = m.addBaseline(deployment.GetId(), deployment.GetName(), deployment.GetClusterId(), deployment.GetNamespace(), timestamp.FromProtobuf(observedDep.ObservationEnd))
		if err != nil {
			log.Error(err)
		}
	}
}

func (m *manager) flushBaselineQueuePeriodically() {
	defer m.baselineFlushTicker.Stop()
	for range m.baselineFlushTicker.C {
		m.flushBaselineQueue()
	}
}

func (m *manager) getFlowStore(ctx context.Context, clusterID string) (networkFlowDS.FlowDataStore, error) {
	flowStore, err := m.clusterFlows.GetFlowStore(ctx, clusterID)
	if err != nil {
		return nil, errors.Errorf("could not obtain flow store for cluster %s: %v", clusterID, err)
	}
	if flowStore == nil {
		return nil, errors.Wrapf(errox.NotFound, "no flow store found for cluster %s", clusterID)
	}
	return flowStore, nil
}

func (m *manager) addBaseline(deploymentID, deploymentName, clusterID, namespace string, observationEnd timestamp.MicroTS) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// We have already created a baseline for this deployment.  Nothing necessary to do.
	if baseline, found := m.baselinesByDeploymentID[deploymentID]; found && baseline != nil {
		return nil
	}

	writeEmpty := true

	flowStore, _ := m.getFlowStore(managerCtx, clusterID)

	// Create an empty baseline entry in the map.  This will put this deployment in a state where it will be updated until
	// its observation end time.
	m.baselinesByDeploymentID[deploymentID] = &networkbaseline.BaselineInfo{
		ClusterID:            clusterID,
		Namespace:            namespace,
		DeploymentName:       deploymentName,
		ObservationPeriodEnd: observationEnd,
		UserLocked:           false,
		BaselinePeers:        make(map[networkbaseline.Peer]struct{}),
		ForbiddenPeers:       make(map[networkbaseline.Peer]struct{}),
	}

	// Grab flows related to deployment
	flows, err := flowStore.GetFlowsForDeployment(managerCtx, deploymentID, false)
	if err != nil {
		return err
	}

	// If we have flows then process them.  If we don't persist an empty baseline
	if len(flows) > 0 {
		// package them into a map of flows like comes in
		// when packaging the flows up in that map, I think the timestamp has to be now
		flowMap := m.putFlowsInMap(flows)

		// then simply call processFlowUpdate with the map of flows.
		modifiedDeployments, err := m.processFlowUpdate(flowMap, true)
		if err != nil {
			return err
		}
		// If the modified list contains our deployment it was persisted with flows
		// If it does not then that means all peers were locked or not yet written to a baseline so we will
		// need to persist an empty baseline object for this deployment.  As subsequent deployments are processed,
		// flows will be added.
		if modifiedDeployments.Contains(deploymentID) {
			writeEmpty = false
		}
	}

	// If there are no flows OR the peers were all locked, we need to write an empty baseline.
	if writeEmpty {
		// Save the empty baseline because we may not be able to update it if all its connections are locked.
		err := m.persistNetworkBaselines(set.NewStringSet(deploymentID), nil)
		if err != nil {
			return err
		}
	}

	// Since the manager holds a cache of network baselines, we know longer need to hold it in the observation queue after processing it
	m.deploymentObservationQueue.RemoveDeployment(deploymentID)

	return nil
}

func (m *manager) CreateNetworkBaseline(deploymentID string) error {
	deployment, exists, err := m.deploymentDS.GetDeployment(managerCtx, deploymentID)
	if !exists {
		return errors.Wrapf(errox.NotFound, "deployment with id %q does not exist", deploymentID)
	}
	if err != nil {
		return err
	}

	// Need the details from the Observation Queue to grab the proper Observation Time
	depDetails := m.deploymentObservationQueue.GetObservationDetails(deploymentID)
	var t timestamp.MicroTS
	if depDetails == nil {
		t = getNewObservationPeriodEnd()
	} else {
		t = timestamp.FromProtobuf(depDetails.ObservationEnd)
	}

	// Now build the baseline
	err = m.addBaseline(deployment.GetId(), deployment.GetName(), deployment.GetClusterId(), deployment.GetNamespace(), t)
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) putFlowsInMap(newFlows []*storage.NetworkFlow) map[networkgraph.NetworkConnIndicator]timestamp.MicroTS {
	out := make(map[networkgraph.NetworkConnIndicator]timestamp.MicroTS, len(newFlows))
	now := timestamp.Now()
	for _, newFlow := range newFlows {
		t := timestamp.FromProtobuf(newFlow.LastSeenTimestamp)
		if newFlow.LastSeenTimestamp == nil {
			t = now
		}

		out[networkgraph.GetNetworkConnIndicator(newFlow)] = t
	}
	return out
}

// New returns an initialized manager, and starts the manager's processing loop in the background.
func New(
	ds datastore.DataStore,
	networkEntities networkEntityDS.EntityDataStore,
	deploymentDS deploymentDS.DataStore,
	networkPolicyDS networkPolicyDS.DataStore,
	clusterFlows networkFlowDS.ClusterDataStore,
	connectionManager connection.Manager,
) (Manager, error) {
	m := &manager{
		ds:                         ds,
		networkEntities:            networkEntities,
		deploymentDS:               deploymentDS,
		networkPolicyDS:            networkPolicyDS,
		clusterFlows:               clusterFlows,
		connectionManager:          connectionManager,
		seenNetworkPolicies:        set.NewSet[uint64](),
		deploymentObservationQueue: queue.New(),
		baselineFlushTicker:        time.NewTicker(baselineFlushTickerDuration),
		baselinesByDeploymentID:    make(map[string]*networkbaseline.BaselineInfo),
	}
	if err := m.initFromStore(); err != nil {
		return nil, err
	}

	// Start the flush baseline process
	go m.flushBaselineQueuePeriodically()

	return m, nil
}

// GetTestPostgresManager provides a network baseline manager connected to postgres for testing purposes.
func GetTestPostgresManager(t *testing.T, pool postgres.DB) (Manager, error) {
	networkBaselineStore, err := datastore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	networkEntityStore, err := networkEntityDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	deploymentStore, err := deploymentDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	networkPolicyStore, err := networkPolicyDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	networkFlowClusterStore, err := networkFlowDS.GetTestPostgresClusterDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	sensorCnxMgr := connection.ManagerSingleton()
	return New(networkBaselineStore, networkEntityStore, deploymentStore, networkPolicyStore, networkFlowClusterStore, sensorCnxMgr)
}
