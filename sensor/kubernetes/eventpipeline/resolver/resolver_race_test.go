package resolver

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dedupingqueue"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	resolverStore "github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type raceTestEnv struct {
	t        *testing.T
	resolver *resolverImpl

	mockDeploymentStore *mocksStore.MockDeploymentStore
	mockRBACStore       *mocksStore.MockRBACStore
	mockServiceStore    *mocksStore.MockServiceStore
	mockEndpointManager *mocksStore.MockEndpointManager

	sentEvents []*component.ResourceEvent
}

func newRaceTestEnv(t *testing.T) *raceTestEnv {
	t.Helper()
	ctrl := gomock.NewController(t)

	depStore := mocksStore.NewMockDeploymentStore(ctrl)
	rbacStore := mocksStore.NewMockRBACStore(ctrl)
	svcStore := mocksStore.NewMockServiceStore(ctrl)
	endpointMgr := mocksStore.NewMockEndpointManager(ctrl)
	mockOutput := mocks.NewMockOutputQueue(ctrl)

	env := &raceTestEnv{
		t:                   t,
		mockDeploymentStore: depStore,
		mockRBACStore:       rbacStore,
		mockServiceStore:    svcStore,
		mockEndpointManager: endpointMgr,
	}

	mockOutput.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(event *component.ResourceEvent) {
		env.sentEvents = append(env.sentEvents, event)
	})

	var queue *dedupingqueue.DedupingQueue[string]
	if features.SensorAggregateDeploymentReferenceOptimization.Enabled() {
		queue = dedupingqueue.NewDedupingQueue[string]()
	}

	env.resolver = &resolverImpl{
		outputQueue: mockOutput,
		storeProvider: &fakeProvider{
			deploymentStore: depStore,
			serviceStore:    svcStore,
			rbacStore:       rbacStore,
			endpointManager: endpointMgr,
		},
		stopper:               concurrency.NewStopper(),
		pullAndResolveStopped: concurrency.NewSignal(),
		deploymentRefQueue:    queue,
	}

	return env
}

type raceTestCase struct {
	steps          []func(*raceTestEnv)
	testFFDisabled bool
}

func TestResolverRaceScenarios(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	cases := map[string]raceTestCase{
		// Central sends UpdatedImage while a K8s deployment update is in-flight.
		// The K8s dispatcher already stored the new wrap (isBuilt=false) but the
		// resolver hasn't processed the deployment's own ref yet. The Central ref
		// reaches the resolver first.
		// FF disabled: resolveDeployment returns false but msg is sent unconditionally
		//   with ReprocessDeployments → PASSES.
		// FF enabled: resolveDeployment returns false and the conditional send in
		//   runPullAndResolve drops the msg → FAILS (bug).
		"central UpdatedImage arrives while deployment is being rebuilt": {
			testFFDisabled: true,
			steps: []func(*raceTestEnv){
				givenDeploymentNotBuilt("deploy-1"),
				dispatchCentralUpdatedImage("deploy-1"),
				resolveNextItem(),
				expectReprocessSent("deploy-1"),
			},
		},

		// Same scenario but the deployment was completely removed from the store
		// between the event being queued and resolved.
		"forceDetection ref for a deleted deployment preserves reprocess data": {
			testFFDisabled: true,
			steps: []func(*raceTestEnv){
				givenDeploymentNotFound("deploy-1"),
				dispatchCentralUpdatedImage("deploy-1"),
				resolveNextItem(),
				expectReprocessSent("deploy-1"),
			},
		},

		// Full resolve path with forceDetection=true but BuildDeploymentWithDependencies
		// fails. The reprocess data was already added to the event but the conditional
		// send drops it.
		"forceDetection ref with build error preserves reprocess data": {
			testFFDisabled: true,
			steps: []func(*raceTestEnv){
				givenDeploymentBuildError("deploy-1"),
				dispatchForceDetectionEvent("deploy-1"),
				resolveNextItem(),
				expectReprocessSent("deploy-1"),
			},
		},

		// Happy path: skipResolving when the deployment is fully built.
		// GetBuiltDeployment returns (d, true), detection succeeds.
		"skipResolving with isBuilt=true succeeds": {
			testFFDisabled: true,
			steps: []func(*raceTestEnv){
				givenDeploymentBuilt("deploy-1"),
				dispatchCentralUpdatedImage("deploy-1"),
				resolveNextItem(),
				expectReprocessSent("deploy-1"),
				expectDetectionSent("deploy-1"),
			},
		},

		// Happy path: full resolve for a new deployment.
		// BuildDeploymentWithDependencies returns newObject=true.
		"full resolve of new deployment triggers detection": {
			testFFDisabled: true,
			steps: []func(*raceTestEnv){
				givenDeploymentResolvable("deploy-1"),
				dispatchK8sDeploymentEvent("deploy-1"),
				resolveNextItem(),
				expectDetectionSent("deploy-1"),
				expectForwardMessageSent("deploy-1"),
			},
		},

		// Full resolve where the deployment hasn't changed (!forceDetection && !newObject).
		// processMessage always sends the original msg (which has no deployment data).
		// resolveNextItem does nothing because resolveDeployment returns false and
		// ReprocessDeployments is empty.
		"full resolve with no changes sends only original msg": {
			testFFDisabled: true,
			steps: []func(*raceTestEnv){
				givenDeploymentUnchanged("deploy-1"),
				dispatchK8sDeploymentEvent("deploy-1"),
				resolveNextItem(),
				expectNoDetectionSent(),
				expectNoForwardMessageSent(),
			},
		},

		// Normal ordering: K8s ref is resolved first (builds deployment, sets isBuilt=true).
		// Then the Central ref arrives and the skipResolving path succeeds.
		"K8s ref resolved before central ref — no race": {
			testFFDisabled: true,
			steps: []func(*raceTestEnv){
				givenDeploymentResolvable("deploy-1"),
				dispatchK8sDeploymentEvent("deploy-1"),
				resolveNextItem(),
				givenDeploymentBuilt("deploy-1"),
				dispatchCentralUpdatedImage("deploy-1"),
				resolveNextItem(),
				expectDetectionSent("deploy-1"),
				expectReprocessSent("deploy-1"),
			},
		},

		// Full resolve with real dependency data flowing through.
		// Verifies that RBAC permissions and service exposure are carried
		// to the output in ForwardMessages and DetectorMessages.
		"full resolve carries dependency data to output": {
			testFFDisabled: true,
			steps: []func(*raceTestEnv){
				givenDeploymentWithDependencies("deploy-1",
					storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
					[]map[service.PortRef][]*storage.PortConfig_ExposureInfo{
						{
							{Port: intstr.IntOrString{IntVal: 8080}, Protocol: "TCP"}: {
								{Level: storage.PortConfig_EXTERNAL, ServiceName: "my.service", ServicePort: 80},
							},
						},
					},
				),
				dispatchK8sDeploymentEvent("deploy-1"),
				resolveNextItem(),
				expectForwardMessageWithPermissionLevel("deploy-1", storage.PermissionLevel_ELEVATED_IN_NAMESPACE),
			},
		},

		// Merge: Central ref (skipResolving=true, forceDetection=true) is pushed first,
		// then K8s ref (skipResolving=false, forceDetection=false) merges into it.
		// Merged result: skipResolving=false, forceDetection=true → full resolve
		// with forced detection. Only one item in the queue.
		"merge: central then K8s ref collapses to full resolve with forceDetection": {
			steps: []func(*raceTestEnv){
				givenDeploymentResolvable("deploy-1"),
				pushSkipResolvingRef("deploy-1"),
				pushFullResolveRef("deploy-1"),
				resolveNextItem(),
				expectDetectionSent("deploy-1"),
				expectReprocessSent("deploy-1"),
				expectForwardMessageSent("deploy-1"),
				expectQueueEmpty(),
			},
		},

		// Merge: same as above but K8s ref is pushed first.
		// Verifies that push order doesn't affect the merged result.
		"merge: K8s then central ref collapses to full resolve with forceDetection": {
			steps: []func(*raceTestEnv){
				givenDeploymentResolvable("deploy-1"),
				pushFullResolveRef("deploy-1"),
				pushSkipResolvingRef("deploy-1"),
				resolveNextItem(),
				expectDetectionSent("deploy-1"),
				expectReprocessSent("deploy-1"),
				expectForwardMessageSent("deploy-1"),
				expectQueueEmpty(),
			},
		},

		// Merge: two K8s refs for the same deployment. Flags don't change
		// (both skipResolving=false, forceDetection=false). Only one resolve happens.
		"merge: two K8s refs dedup to single resolve": {
			steps: []func(*raceTestEnv){
				givenDeploymentResolvable("deploy-1"),
				pushFullResolveRef("deploy-1"),
				pushFullResolveRef("deploy-1"),
				resolveNextItem(),
				expectDetectionSent("deploy-1"),
				expectForwardMessageSent("deploy-1"),
				expectQueueEmpty(),
			},
		},

		// Merge eliminates the isBuilt race: K8s ref is already in the queue when
		// the Central ref arrives. The merge produces skipResolving=false,
		// forceDetection=true. The full resolve path bypasses the isBuilt guard
		// entirely.
		"merge: both refs in queue bypass isBuilt guard": {
			steps: []func(*raceTestEnv){
				givenDeploymentResolvable("deploy-1"),
				pushFullResolveRef("deploy-1"),
				pushSkipResolvingRef("deploy-1"),
				resolveNextItem(),
				expectDetectionSent("deploy-1"),
				expectReprocessSent("deploy-1"),
				expectQueueEmpty(),
			},
		},
	}

	for name, tc := range cases {
		ffStates := []bool{true}
		if tc.testFFDisabled {
			ffStates = append(ffStates, false)
		}
		for _, ffEnabled := range ffStates {
			t.Run(fmt.Sprintf("%s/ff=%t", name, ffEnabled), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					t.Setenv(features.SensorAggregateDeploymentReferenceOptimization.EnvVar(),
						fmt.Sprintf("%t", ffEnabled))
					env := newRaceTestEnv(t)
					for _, step := range tc.steps {
						step(env)
					}
				})
			})
		}
	}
}

// --- Step functions: mock setup ---

func givenDeploymentNotBuilt(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.mockDeploymentStore.EXPECT().Get(gomock.Eq(id)).
			Return(&storage.Deployment{Id: id}).Times(1)
		env.mockDeploymentStore.EXPECT().GetBuiltDeployment(gomock.Eq(id)).
			Return(&storage.Deployment{Id: id}, false).Times(1)
	}
}

func givenDeploymentNotFound(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.mockDeploymentStore.EXPECT().Get(gomock.Eq(id)).
			Return(nil).Times(1)
		// GetBuiltDeployment is NOT called because Get returns nil → early return.
	}
}

func givenDeploymentBuilt(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.mockDeploymentStore.EXPECT().Get(gomock.Eq(id)).
			Return(&storage.Deployment{Id: id}).Times(1)
		env.mockDeploymentStore.EXPECT().GetBuiltDeployment(gomock.Eq(id)).
			Return(&storage.Deployment{Id: id}, true).Times(1)
	}
}

func givenDeploymentResolvable(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		givenFullResolveMocks(env, id,
			storage.PermissionLevel_NONE, nil,
			&storage.Deployment{Id: id}, true, nil)
	}
}

func givenDeploymentUnchanged(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		givenFullResolveMocks(env, id,
			storage.PermissionLevel_NONE, nil,
			&storage.Deployment{Id: id}, false, nil)
	}
}

func givenDeploymentBuildError(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		givenFullResolveMocks(env, id,
			storage.PermissionLevel_NONE, nil,
			nil, false, errors.New("dependency error"))
	}
}

func givenDeploymentWithDependencies(
	id string,
	permLevel storage.PermissionLevel,
	exposures []map[service.PortRef][]*storage.PortConfig_ExposureInfo,
) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		var flatExposures []*storage.PortConfig_ExposureInfo
		for _, e := range exposures {
			for _, list := range e {
				flatExposures = append(flatExposures, list...)
			}
		}
		builtDeployment := &storage.Deployment{
			Id:                            id,
			ServiceAccountPermissionLevel: permLevel,
			Ports: []*storage.PortConfig{
				{ExposureInfos: flatExposures},
			},
		}
		givenFullResolveMocks(env, id, permLevel, exposures, builtDeployment, true, nil)
	}
}

func givenFullResolveMocks(
	env *raceTestEnv,
	id string,
	permLevel storage.PermissionLevel,
	exposures []map[service.PortRef][]*storage.PortConfig_ExposureInfo,
	builtDeployment *storage.Deployment,
	newObject bool,
	buildErr error,
) {
	env.mockDeploymentStore.EXPECT().Get(gomock.Eq(id)).
		Return(&storage.Deployment{Id: id}).Times(1)
	env.mockEndpointManager.EXPECT().OnDeploymentCreateOrUpdateByID(gomock.Eq(id)).Times(1)
	env.mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).
		Return(permLevel).Times(1)
	env.mockServiceStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).
		Return(exposures).Times(1)
	env.mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(
		gomock.Eq(id), gomock.Eq(store.Dependencies{
			PermissionLevel: permLevel,
			Exposures:       exposures,
			LocalImages:     set.NewStringSet(),
		})).
		Return(builtDeployment, newObject, buildErr).Times(1)
}

// --- Step functions: dispatch events ---

// dispatchCentralUpdatedImage creates a ResourceEvent with skipResolving+forceDetection
// and feeds it through processMessage.
// Represents: Central sent UpdatedImage, ReprocessDeployment, or InvalidateImageCache.
func dispatchCentralUpdatedImage(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		event := &component.ResourceEvent{
			DeploymentReferences: []component.DeploymentReference{
				{
					Reference:            resolverStore.ResolveDeploymentIds(id),
					ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
					ForceDetection:       true,
					SkipResolving:        true,
				},
			},
			Context: context.Background(),
		}
		env.resolver.processMessage(event)
	}
}

// dispatchK8sDeploymentEvent creates a ResourceEvent with a full-resolve ref
// and feeds it through processMessage.
// Represents: K8s dispatcher event (deployment create/update, service, RBAC, secret).
func dispatchK8sDeploymentEvent(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		event := &component.ResourceEvent{
			DeploymentReferences: []component.DeploymentReference{
				{
					Reference:            resolverStore.ResolveDeploymentIds(id),
					ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
				},
			},
			Context: context.Background(),
		}
		env.resolver.processMessage(event)
	}
}

// dispatchForceDetectionEvent creates a ResourceEvent with forceDetection=true
// (but skipResolving=false) and feeds it through processMessage.
// Represents: network policy or service account event.
func dispatchForceDetectionEvent(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		event := &component.ResourceEvent{
			DeploymentReferences: []component.DeploymentReference{
				{
					Reference:            resolverStore.ResolveDeploymentIds(id),
					ParentResourceAction: central.ResourceAction_UPDATE_RESOURCE,
					ForceDetection:       true,
				},
			},
			Context: context.Background(),
		}
		env.resolver.processMessage(event)
	}
}

// --- Step functions: direct queue operations ---

// pushSkipResolvingRef pushes a skipResolving+forceDetection ref directly to the queue.
// Represents: Central sent UpdatedImage, ReprocessDeployment, or InvalidateImageCache.
// Used in merge tests where we need two refs in the queue before resolving.
func pushSkipResolvingRef(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.resolver.deploymentRefQueue.Push(&deploymentRef{
			context:        context.Background(),
			id:             id,
			action:         central.ResourceAction_UPDATE_RESOURCE,
			forceDetection: true,
			skipResolving:  true,
		})
	}
}

// pushFullResolveRef pushes a full-resolve ref directly to the queue.
// Represents: K8s dispatcher event (deployment create/update, service, RBAC, secret).
// Used in merge tests where we need two refs in the queue before resolving.
func pushFullResolveRef(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.resolver.deploymentRefQueue.Push(&deploymentRef{
			context:        context.Background(),
			id:             id,
			action:         central.ResourceAction_UPDATE_RESOURCE,
			forceDetection: false,
			skipResolving:  false,
		})
	}
}

// --- Step functions: resolve ---

// resolveNextItem pulls one item from the DedupingQueue and resolves it
// via resolveAndSend — the same method used by runPullAndResolve.
// When the FF is disabled (no queue), this is a no-op because processMessage
// already resolved the ref inline.
func resolveNextItem() func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		if env.resolver.deploymentRefQueue == nil {
			return
		}

		stop := concurrency.NewSignal()
		item := env.resolver.deploymentRefQueue.PullBlocking(&stop)
		require.NotNil(env.t, item, "expected an item in the queue")

		ref, ok := item.(*deploymentRef)
		require.True(env.t, ok, "pulled item is not a *deploymentRef")

		env.resolver.resolveAndSend(ref)
	}
}

// --- Step functions: assertions ---

func expectReprocessSent(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.t.Helper()
		for _, event := range env.sentEvents {
			if slices.Contains(event.ReprocessDeployments, id) {
				return
			}
		}
		assert.Failf(env.t, "missing reprocess",
			"expected ReprocessDeployments containing %q, got events: %s",
			id, formatSentEvents(env.sentEvents))
	}
}

func expectDetectionSent(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.t.Helper()
		for _, event := range env.sentEvents {
			for _, det := range event.DetectorMessages {
				if det.Object.GetId() == id {
					return
				}
			}
		}
		assert.Failf(env.t, "missing detection",
			"expected DetectorMessages for %q, got events: %s",
			id, formatSentEvents(env.sentEvents))
	}
}

func expectForwardMessageSent(id string) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.t.Helper()
		for _, event := range env.sentEvents {
			for _, msg := range event.ForwardMessages {
				if msg.GetDeployment().GetId() == id {
					return
				}
			}
		}
		assert.Failf(env.t, "missing forward message",
			"expected ForwardMessages with deployment %q, got events: %s",
			id, formatSentEvents(env.sentEvents))
	}
}

func expectForwardMessageWithPermissionLevel(id string, level storage.PermissionLevel) func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.t.Helper()
		for _, event := range env.sentEvents {
			for _, msg := range event.ForwardMessages {
				d := msg.GetDeployment()
				if d.GetId() == id && d.GetServiceAccountPermissionLevel() == level {
					return
				}
			}
		}
		assert.Failf(env.t, "missing forward message with permission level",
			"expected ForwardMessages with deployment %q and permission level %s, got events: %s",
			id, level, formatSentEvents(env.sentEvents))
	}
}

func expectNoDetectionSent() func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.t.Helper()
		for _, event := range env.sentEvents {
			assert.Empty(env.t, event.DetectorMessages,
				"expected no DetectorMessages, got events: %s", formatSentEvents(env.sentEvents))
		}
	}
}

func expectNoForwardMessageSent() func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.t.Helper()
		for _, event := range env.sentEvents {
			assert.Empty(env.t, event.ForwardMessages,
				"expected no ForwardMessages, got events: %s", formatSentEvents(env.sentEvents))
		}
	}
}

// expectQueueEmpty asserts that the DedupingQueue has no remaining items.
// Verifies that the merge collapsed multiple refs into one.
// Uses synctest.Wait to confirm PullBlocking is durably blocked (queue empty).
func expectQueueEmpty() func(*raceTestEnv) {
	return func(env *raceTestEnv) {
		env.t.Helper()
		if env.resolver.deploymentRefQueue == nil {
			return
		}
		stop := concurrency.NewSignal()
		var pulled bool
		go func() {
			env.resolver.deploymentRefQueue.PullBlocking(&stop)
			pulled = true
		}()
		synctest.Wait()
		assert.False(env.t, pulled, "expected queue to be empty after merge, but an item was pulled")
		stop.Signal()
	}
}

func formatSentEvents(events []*component.ResourceEvent) string {
	if len(events) == 0 {
		return "[]"
	}
	result := "["
	for i, e := range events {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("{Forward:%d, Detector:%d, Reprocess:%v}",
			len(e.ForwardMessages), len(e.DetectorMessages), e.ReprocessDeployments)
	}
	return result + "]"
}
