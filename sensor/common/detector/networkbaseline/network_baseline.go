package networkbaseline

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/networkbaseline"
	"github.com/stackrox/rox/pkg/sync"
)

// Evaluator encapsulates the interface to the network baseline evaluator
type Evaluator interface {
	RemoveBaselineByDeploymentID(id string)
	AddBaseline(baseline *storage.NetworkBaseline) error
	IsOutsideLockedBaseline(flow *storage.NetworkFlow, srcName, dstName string) bool
}

type networkBaselineEvaluator struct {
	// deployment ID -> baselines
	baselines    map[string]*networkbaseline.BaselineInfo
	baselineLock sync.RWMutex
}

// NewNetworkBaselineEvaluator creates a new network baseline evaluator
func NewNetworkBaselineEvaluator() Evaluator {
	return &networkBaselineEvaluator{
		baselines: make(map[string]*networkbaseline.BaselineInfo),
	}
}

// RemoveBaselineByDeploymentID removes the baselines for this specific deployment
func (e *networkBaselineEvaluator) RemoveBaselineByDeploymentID(id string) {
	e.baselineLock.Lock()
	defer e.baselineLock.Unlock()

	delete(e.baselines, id)
}

// AddBaseline adds a baseline to the evaluator. We only store the locked baselines, as those
// are the ones that matter in the context of evaluator
func (e *networkBaselineEvaluator) AddBaseline(baseline *storage.NetworkBaseline) error {
	if !baseline.GetLocked() {
		// In case we were passed in an unlocked baseline, remember to delete it from evaluator as well
		// as this function can be called in the path of any baseline update, including unlocking
		e.baselineLock.Lock()
		defer e.baselineLock.Unlock()
		delete(e.baselines, baseline.GetDeploymentId())
		return nil
	}

	baselineInfo, err := networkbaseline.ConvertBaselineInfoFromProto(baseline)
	if err != nil {
		return err
	}

	e.baselineLock.Lock()
	defer e.baselineLock.Unlock()
	e.baselines[baseline.GetDeploymentId()] = baselineInfo
	return nil
}

func (e *networkBaselineEvaluator) checkPeerInBaselineForEntity(
	baselineEntity *storage.NetworkEntityInfo,
	peerEntity *storage.NetworkEntityInfo,
	peerEntityName string,
	dstPort uint32,
	protocol storage.L4Protocol,
	isIngressToBaselineEntity bool,
) bool {
	baselineInfo, ok := e.baselines[baselineEntity.GetId()]
	if !ok {
		// If no baseline exists then we do not mark it as anomalous
		return true
	}
	peer := networkbaseline.PeerFromNetworkEntityInfo(peerEntity, peerEntityName, dstPort, protocol, isIngressToBaselineEntity)
	_, peerInBaseline := baselineInfo.BaselinePeers[peer]
	return peerInBaseline
}

// IsOutsideLockedBaseline checks if the network flow is within a locked baseline
// If both entities within this flow are deployments, then we check on both sides
// If the baseline does not exist, then we return true
func (e *networkBaselineEvaluator) IsOutsideLockedBaseline(flow *storage.NetworkFlow, srcName, dstName string) bool {
	e.baselineLock.RLock()
	defer e.baselineLock.RUnlock()

	// Check for src
	if flow.GetProps().GetSrcEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT {
		if !e.checkPeerInBaselineForEntity(
			flow.GetProps().GetSrcEntity(),
			flow.GetProps().GetDstEntity(),
			dstName,
			flow.GetProps().GetDstPort(),
			flow.GetProps().GetL4Protocol(),
			false) {
			return true
		}
	}
	if flow.GetProps().GetDstEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT {
		if !e.checkPeerInBaselineForEntity(
			flow.GetProps().GetDstEntity(),
			flow.GetProps().GetSrcEntity(),
			srcName,
			flow.GetProps().GetDstPort(),
			flow.GetProps().GetL4Protocol(),
			true) {
			return true
		}
	}
	// Passed on both sides. Flow is not anomalous
	return false
}
