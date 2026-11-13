package complianceoperator

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/noderole/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite
	client     *fake.FakeDynamicClient
	dispatcher common.PubSubDispatcher
	mcpStore   *store.MachineConfigPoolStore
	manager    *Manager
}

func newTestManagerDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	d, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewBlockingLane(pubsub.ComplianceCentralRequestLane),
		lane.NewBlockingLane(pubsub.ComplianceCentralResultLane),
		lane.NewBlockingLane(pubsub.NodeRoleChangeLane),
	}))
	require.NoError(t, err)
	t.Cleanup(func() { d.Stop() })
	return d
}

func (s *ManagerTestSuite) SetupTest() {
	gvrToListKind := map[schema.GroupVersionResource]string{
		complianceoperator.ScanSetting.GroupVersionResource():        "ScanSettingList",
		complianceoperator.ScanSettingBinding.GroupVersionResource(): "ScanSettingBindingList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
	s.dispatcher = newTestManagerDispatcher(s.T())
	s.mcpStore = store.NewMachineConfigPoolStore()

	ready := concurrency.NewSignal()
	ready.Signal()

	statusInfo := &fakeStatusInfo{namespace: testNS}

	s.manager = NewManager(s.client, statusInfo, s.mcpStore, &ready, s.dispatcher)
	s.manager.mcpDebounceInterval = 10 * time.Millisecond
}

type fakeStatusInfo struct {
	namespace string
}

func (f *fakeStatusInfo) GetNamespace() string {
	return f.namespace
}

func (s *ManagerTestSuite) TestGetRolesDefaultsToMasterWorker() {
	roles := s.manager.getRoles()
	s.ElementsMatch([]string{masterRole, workerRole}, roles)
}

func (s *ManagerTestSuite) TestGetRolesFromMCPStoreWhenFFEnabled() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	if !features.ComplianceCustomNodeRoles.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceCustomNodeRoles.EnvVar())
	}

	s.mcpStore.Add("infra")
	s.mcpStore.Add("gpu")

	roles := s.manager.getRoles()
	s.ElementsMatch([]string{"infra", "gpu"}, roles)
}

func (s *ManagerTestSuite) TestGetRolesFallsBackWhenMCPStoreEmpty() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	if !features.ComplianceCustomNodeRoles.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceCustomNodeRoles.EnvVar())
	}

	roles := s.manager.getRoles()
	s.ElementsMatch([]string{masterRole, workerRole}, roles)
}

func (s *ManagerTestSuite) TestGetRolesIgnoresMCPStoreWhenFFDisabled() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "false")

	s.mcpStore.Add("infra")
	s.mcpStore.Add("gpu")

	roles := s.manager.getRoles()
	s.ElementsMatch([]string{masterRole, workerRole}, roles)
}

func (s *ManagerTestSuite) TestGetRolesNilMCPStore() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	if !features.ComplianceCustomNodeRoles.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceCustomNodeRoles.EnvVar())
	}

	s.manager.mcpStore = nil
	roles := s.manager.getRoles()
	s.ElementsMatch([]string{masterRole, workerRole}, roles)
}

func (s *ManagerTestSuite) registerResultConsumer() *atomic.Pointer[events.ComplianceCentralResultEvent] {
	s.T().Helper()
	var result atomic.Pointer[events.ComplianceCentralResultEvent]
	err := s.dispatcher.RegisterConsumerToLane(
		pubsub.ComplianceHandlerConsumer,
		pubsub.ComplianceCentralResultTopic,
		pubsub.ComplianceCentralResultLane,
		func(ev pubsub.Event) error {
			r := ev.(*events.ComplianceCentralResultEvent)
			result.Store(r)
			return nil
		},
	)
	s.Require().NoError(err)
	return &result
}

func (s *ManagerTestSuite) TestProcessApplyScanConfigScheduledScan() {
	err := s.manager.Start()
	s.Require().NoError(err)
	defer s.manager.Stop()

	result := s.registerResultConsumer()

	reqEvent := &events.ComplianceCentralRequestEvent{
		Context: context.Background(),
		Request: &central.ComplianceRequest{
			Request: &central.ComplianceRequest_ApplyScanConfig{
				ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
					Id: "req-1",
					ScanRequest: &central.ApplyComplianceScanConfigRequest_ScheduledScan_{
						ScheduledScan: &central.ApplyComplianceScanConfigRequest_ScheduledScan{
							ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
								ScanName:       "test-scan",
								StrictNodeScan: true,
								Profiles:       []string{"ocp4-cis"},
							},
							Cron: "0 0 * * *",
						},
					},
				},
			},
		},
	}

	err = s.dispatcher.Publish(reqEvent)
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		return result.Load() != nil
	}, 2*time.Second, 10*time.Millisecond)

	r := result.Load()
	s.Equal("req-1", r.RequestID)
	s.Equal(events.ApplyScanConfigResponse, r.Type)
	s.NoError(r.Error)

	ss, err := s.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "test-scan", v1.GetOptions{})
	s.Require().NoError(err)
	s.Equal("test-scan", ss.GetName())
}

func (s *ManagerTestSuite) TestProcessDeleteScanConfig() {
	err := s.manager.Start()
	s.Require().NoError(err)
	defer s.manager.Stop()

	request := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName:       "del-scan",
		StrictNodeScan: true,
		Profiles:       []string{"ocp4-cis"},
	}
	err = s.manager.writer.createScanResources(context.Background(), testNS, request, "0 0 * * *")
	s.Require().NoError(err)

	result := s.registerResultConsumer()

	reqEvent := &events.ComplianceCentralRequestEvent{
		Context: context.Background(),
		Request: &central.ComplianceRequest{
			Request: &central.ComplianceRequest_DeleteScanConfig{
				DeleteScanConfig: &central.DeleteComplianceScanConfigRequest{
					Id:   "req-del",
					Name: "del-scan",
				},
			},
		},
	}

	err = s.dispatcher.Publish(reqEvent)
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		return result.Load() != nil
	}, 2*time.Second, 10*time.Millisecond)

	r := result.Load()
	s.Equal("req-del", r.RequestID)
	s.Equal(events.DeleteScanConfigResponse, r.Type)
	s.NoError(r.Error)

	_, err = s.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "del-scan", v1.GetOptions{})
	s.Error(err)
}

func (s *ManagerTestSuite) TestProcessSyncScanConfigs() {
	err := s.manager.Start()
	s.Require().NoError(err)
	defer s.manager.Stop()

	reqEvent := &events.ComplianceCentralRequestEvent{
		Context: context.Background(),
		Request: &central.ComplianceRequest{
			Request: &central.ComplianceRequest_SyncScanConfigs{
				SyncScanConfigs: &central.SyncComplianceScanConfigRequest{
					ScanConfigs: []*central.ApplyComplianceScanConfigRequest{
						{
							ScanRequest: &central.ApplyComplianceScanConfigRequest_UpdateScan{
								UpdateScan: &central.ApplyComplianceScanConfigRequest_UpdateScheduledScan{
									ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
										ScanName:       "sync-scan",
										StrictNodeScan: true,
										Profiles:       []string{"ocp4-cis"},
									},
									Cron: "0 0 * * *",
								},
							},
						},
					},
				},
			},
		},
	}

	err = s.dispatcher.Publish(reqEvent)
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		ss, err := s.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
			Namespace(testNS).Get(context.Background(), "sync-scan", v1.GetOptions{})
		return err == nil && ss != nil
	}, 2*time.Second, 10*time.Millisecond)
}

func (s *ManagerTestSuite) TestProcessApplyWithEmptyNamespace() {
	statusInfo := &fakeStatusInfo{namespace: ""}
	ready := concurrency.NewSignal()
	ready.Signal()
	m := NewManager(s.client, statusInfo, s.mcpStore, &ready, s.dispatcher)
	m.mcpDebounceInterval = 10 * time.Millisecond

	err := m.Start()
	s.Require().NoError(err)
	defer m.Stop()

	result := s.registerResultConsumer()

	reqEvent := &events.ComplianceCentralRequestEvent{
		Context: context.Background(),
		Request: &central.ComplianceRequest{
			Request: &central.ComplianceRequest_ApplyScanConfig{
				ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
					Id: "req-empty-ns",
					ScanRequest: &central.ApplyComplianceScanConfigRequest_ScheduledScan_{
						ScheduledScan: &central.ApplyComplianceScanConfigRequest_ScheduledScan{
							ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
								ScanName: "test-scan",
								Profiles: []string{"ocp4-cis"},
							},
							Cron: "0 0 * * *",
						},
					},
				},
			},
		},
	}

	err = s.dispatcher.Publish(reqEvent)
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		return result.Load() != nil
	}, 2*time.Second, 10*time.Millisecond)

	r := result.Load()
	s.Equal("req-empty-ns", r.RequestID)
	s.Error(r.Error)
	s.Contains(r.Error.Error(), "compliance operator namespace not known")
}

func (s *ManagerTestSuite) TestReconcileMCPRolesUpdatesExistingScanSettings() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	if !features.ComplianceCustomNodeRoles.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceCustomNodeRoles.EnvVar())
	}

	request := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName:       "role-scan",
		StrictNodeScan: true,
		Profiles:       []string{"ocp4-cis"},
	}
	err := s.manager.writer.createScanResources(context.Background(), testNS, request, "0 0 * * *")
	s.Require().NoError(err)

	s.mcpStore.Add("infra")
	s.mcpStore.Add("gpu")

	s.manager.reconcileMCPRoles()

	ss, err := s.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "role-scan", v1.GetOptions{})
	s.Require().NoError(err)

	var scanSetting v1alpha1.ScanSetting
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(ss.Object, &scanSetting)
	s.Require().NoError(err)
	s.ElementsMatch([]string{"infra", "gpu"}, scanSetting.Roles)
}

func (s *ManagerTestSuite) TestReconcileMCPRolesNoOpWhenFFDisabled() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "false")

	request := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName:       "noop-scan",
		StrictNodeScan: true,
		Profiles:       []string{"ocp4-cis"},
	}
	err := s.manager.writer.createScanResources(context.Background(), testNS, request, "0 0 * * *")
	s.Require().NoError(err)

	s.mcpStore.Add("custom-role")

	s.manager.reconcileMCPRoles()

	ss, err := s.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "noop-scan", v1.GetOptions{})
	s.Require().NoError(err)

	var scanSetting v1alpha1.ScanSetting
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(ss.Object, &scanSetting)
	s.Require().NoError(err)
	s.ElementsMatch([]string{masterRole, workerRole}, scanSetting.Roles)
}

func (s *ManagerTestSuite) TestReconcileMCPRolesNoOpWithEmptyNamespace() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	if !features.ComplianceCustomNodeRoles.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceCustomNodeRoles.EnvVar())
	}

	s.manager.complianceOperatorInfo = &fakeStatusInfo{namespace: ""}
	s.mcpStore.Add("infra")

	s.manager.reconcileMCPRoles()
}

func (s *ManagerTestSuite) TestReconcileMCPRolesNilMCPStore() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	if !features.ComplianceCustomNodeRoles.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceCustomNodeRoles.EnvVar())
	}

	s.manager.mcpStore = nil

	s.manager.reconcileMCPRoles()
}

func getScanSettingRoles(client *fake.FakeDynamicClient, name string) ([]string, error) {
	ss, err := client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var scanSetting v1alpha1.ScanSetting
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(ss.Object, &scanSetting); err != nil {
		return nil, err
	}
	return scanSetting.Roles, nil
}

func (s *ManagerTestSuite) TestMCPEventTriggersReconciliation() {
	s.T().Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	if !features.ComplianceCustomNodeRoles.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceCustomNodeRoles.EnvVar())
	}

	request := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName:       "mcp-event-scan",
		StrictNodeScan: true,
		Profiles:       []string{"ocp4-cis"},
	}
	err := s.manager.writer.createScanResources(context.Background(), testNS, request, "0 0 * * *")
	s.Require().NoError(err)

	// Verify initial roles are the defaults.
	roles, err := getScanSettingRoles(s.client, "mcp-event-scan")
	s.Require().NoError(err)
	s.ElementsMatch([]string{masterRole, workerRole}, roles)

	err = s.manager.Start()
	s.Require().NoError(err)
	defer s.manager.Stop()

	s.mcpStore.Add("infra")
	s.mcpStore.Add("gpu")

	err = s.dispatcher.Publish(&events.NodeRoleChangeEvent{})
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		roles, err := getScanSettingRoles(s.client, "mcp-event-scan")
		if err != nil {
			return false
		}
		return slices.Contains(roles, "infra")
	}, 2*time.Second, 10*time.Millisecond, "expected MCP debounce to trigger role update")

	roles, err = getScanSettingRoles(s.client, "mcp-event-scan")
	s.Require().NoError(err)
	s.ElementsMatch([]string{"infra", "gpu"}, roles)
}

func TestOnMCPEventNonBlocking(t *testing.T) {
	ready := concurrency.NewSignal()
	ready.Signal()
	statusInfo := &fakeStatusInfo{namespace: testNS}
	d := newTestManagerDispatcher(t)

	gvrToListKind := map[schema.GroupVersionResource]string{
		complianceoperator.ScanSetting.GroupVersionResource():        "ScanSettingList",
		complianceoperator.ScanSettingBinding.GroupVersionResource(): "ScanSettingBindingList",
	}
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)

	m := NewManager(client, statusInfo, nil, &ready, d)

	for range 10 {
		err := m.onMCPEvent(nil)
		assert.NoError(t, err)
	}
}
