package policyrefresh

import (
	"time"

	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/lifecycle"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	policyPkg "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/throttle"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(policyDataStore.Singleton(), lifecycle.SingletonManager(), buildtime.SingletonPolicySet())
}

// NewPipeline returns a new instance of Pipeline for k8s role bindings
func NewPipeline(policies policyDataStore.DataStore, deployAndRuntimeManager lifecycle.Manager, buildTimePolicies detection.PolicySet) pipeline.Fragment {
	return &pipelineImpl{
		policies:                policies,
		deployAndRuntimeManager: deployAndRuntimeManager,
		buildTimePolicies:       buildTimePolicies,
		throttler:               throttle.NewDropThrottle(time.Second),
	}
}

type pipelineImpl struct {
	policies                policyDataStore.DataStore
	deployAndRuntimeManager lifecycle.Manager
	buildTimePolicies       detection.PolicySet
	throttler               throttle.DropThrottle
}

// Reconcile runs after all updates for a cluster have be run through their respective pipelines.
func (s *pipelineImpl) Reconcile(clusterID string) error {
	// Recompile all policies that might need it, but throttle.
	s.throttler.Run(func() {
		err := s.updatePolicies(msgAndPolicyPredicates)
		if err != nil {
			log.Error(err)
		}
	})
	return nil
}

// Match returns whether or not the input message should be run in this pipeline.
func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return len(predicatesForMessage(msg)) > 0
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	// Recompile all policies that might need it, but throttle.
	// Later we can change how we throttle to be per-policy, so we only refresh policies that need it based on what the
	// input message might have changed. Basically, instead of passing in all the predicates here, only pass those
	// returned by predicatesForMessage(msg).
	s.throttler.Run(func() {
		err := s.updatePolicies(msgAndPolicyPredicates)
		if err != nil {
			log.Error(err)
		}
	})
	return nil
}

func (s *pipelineImpl) OnFinish(clusterID string) {}

// Non-static helper functions.
///////////////////////////////

// Update all of the policies that pass the input list of predicates.
func (s *pipelineImpl) updatePolicies(predicates []*predicate) error {
	policies, err := s.policies.GetPolicies()
	if err != nil {
		return err
	}

	policiesToUpdate := filterPoliciesThatNeedRecompile(policies, predicates)
	if len(policiesToUpdate) == 0 {
		return nil
	}
	return s.recompilePolicies(policiesToUpdate)
}

// Recompile all of the input policies.
func (s *pipelineImpl) recompilePolicies(policiesToUpdate []*storage.Policy) error {
	errorList := errorhelpers.NewErrorList("unable to refresh policies")
	for _, policy := range policiesToUpdate {
		if policyPkg.AppliesAtBuildTime(policy) {
			errorList.AddError(s.buildTimePolicies.Recompile(policy.GetId()))
		}
		if policyPkg.AppliesAtDeployTime(policy) || policyPkg.AppliesAtRunTime(policy) {
			errorList.AddError(s.deployAndRuntimeManager.RecompilePolicy(policy))
		}
	}
	return errorList.ToError()
}

// Static helper functions.
///////////////////////////

// Filter out policies from the input list that do not match any of the input predicates.
func filterPoliciesThatNeedRecompile(in []*storage.Policy, predicates []*predicate) []*storage.Policy {
	out := make([]*storage.Policy, 0, len(in))
	for _, policy := range in {
		for _, predicate := range predicates {
			if predicate.policyPred(policy) {
				out = append(out, policy)
				break
			}
		}
	}
	return out
}
