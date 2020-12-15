package manager

import (
	"context"
	"sort"

	"github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
)

var (
	managerCtx = sac.WithAllAccess(context.Background())
)

type peer struct {
	isIngress bool
	entity    networkgraph.Entity
	dstPort   uint32
	protocol  storage.L4Protocol
}

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

func (m *manager) shouldUpdate(conn *networkgraph.NetworkConnIndicator, updateTS timestamp.MicroTS) bool {
	var atLeastOneBaselineInObservationPeriod bool
	for _, entity := range []*networkgraph.Entity{&conn.SrcEntity, &conn.DstEntity} {
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

func convertPeersFromProto(protoPeers []*storage.NetworkBaselinePeer) map[peer]struct{} {
	out := make(map[peer]struct{}, len(protoPeers))
	for _, protoPeer := range protoPeers {
		entity := networkgraph.Entity{ID: protoPeer.GetEntity().GetInfo().GetId(), Type: protoPeer.GetEntity().GetInfo().GetType()}
		for _, props := range protoPeer.GetProperties() {
			out[peer{
				isIngress: props.GetIngress(),
				entity:    entity,
				dstPort:   props.GetPort(),
				protocol:  props.GetProtocol(),
			}] = struct{}{}
		}
	}
	return out
}

func convertPeersToProto(peerSet map[peer]struct{}) []*storage.NetworkBaselinePeer {
	if len(peerSet) == 0 {
		return nil
	}
	propertiesByEntity := make(map[networkgraph.Entity][]*storage.NetworkBaselineConnectionProperties)
	for peer := range peerSet {
		propertiesByEntity[peer.entity] = append(propertiesByEntity[peer.entity], &storage.NetworkBaselineConnectionProperties{
			Ingress:  peer.isIngress,
			Port:     peer.dstPort,
			Protocol: peer.protocol,
		})
	}
	out := make([]*storage.NetworkBaselinePeer, 0, len(propertiesByEntity))
	for entity, properties := range propertiesByEntity {
		sort.Slice(properties, func(i, j int) bool {
			if properties[i].Ingress != properties[j].Ingress {
				return properties[i].Ingress
			}
			if properties[i].Protocol != properties[j].Protocol {
				return properties[i].Protocol < properties[j].Protocol
			}
			return properties[i].Port < properties[j].Port
		})
		out = append(out, &storage.NetworkBaselinePeer{
			Entity: &storage.NetworkEntity{
				Info: &storage.NetworkEntityInfo{
					Type: entity.Type,
					Id:   entity.ID,
				},
			},
			Properties: properties,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].GetEntity().GetInfo().GetId() < out[j].GetEntity().GetInfo().GetId()
	})
	return out
}

func (m *manager) persistNetworkBaselines(deploymentIDs set.StringSet) error {
	if deploymentIDs.Cardinality() == 0 {
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
