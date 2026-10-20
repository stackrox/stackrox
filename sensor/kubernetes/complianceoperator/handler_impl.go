package complianceoperator

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/k8sapi"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

const (
	defaultMaxRetries     = 5
	defaultAPICallTimeout = 5 * time.Second
	defaultRetryTimeout   = 30 * time.Second
)

type handlerImpl struct {
	client                 dynamic.Interface
	complianceOperatorInfo StatusInfo

	response chan *message.ExpiringMessage
	request  chan *central.ComplianceRequest

	disabled              concurrency.Signal
	stopSignal            concurrency.Signal
	started               *atomic.Bool
	complianceIsReady     *concurrency.Signal
	handlerMaxRetries     int
	handlerAPICallTimeout time.Duration
	handlerRetryTimeout   time.Duration

	// Node-role auto-discovery (ROX_COMPLIANCE_AUTODISCOVER_NODE_ROLES). mcpLister and mcpInformerSynced are set
	// by startMCPRoleInformer, which runs in its own goroutine (see Start()) so a slow or unreachable API server
	// can never delay the rest of Sensor's startup. They stay nil when MachineConfigPools aren't listable
	// (non-OpenShift) or discovery failed to start, and discoverNodeRoles reports an error so every caller falls
	// back to master+worker. mcpMu guards both fields: they are written from the startMCPRoleInformer goroutine
	// and read from run() (triggered independently by the informer's own event handlers), so plain field access
	// would be a data race.
	mcpMu                       sync.RWMutex
	mcpLister                   cache.GenericLister
	mcpInformerSynced           cache.InformerSynced
	roleReconcileTrigger        chan struct{}
	nodeRoleReconcilePassBudget time.Duration
	// nodeRoleReconcileInterval is the period of the safety-net reconcile tick in run(); overridable in tests.
	nodeRoleReconcileInterval time.Duration

	// backgroundWG tracks the goroutines Start() spawns (run, startMCPRoleInformer, reconcileOnceReady) so Stop()
	// can wait for them to finish. Without the wait, a goroutine can still read the package-level log (e.g.
	// startMCPRoleInformer's sync-timeout warning) after Stop() returns, which races tests that swap log.
	backgroundWG sync.WaitGroup
}

func (m *handlerImpl) Name() string {
	return "complianceoperator.handlerImpl"
}

type scanScheduleConfiguration struct {
	Suspend        *bool
	Schedule       *string
	ScanName       string
	Request        interface{}
	ValidationFunc func(interface{}) error
}

// NewRequestHandler returns instance of common.SensorComponent interface which can handle compliance requests from Central.
func NewRequestHandler(client dynamic.Interface, complianceOperatorInfo StatusInfo, coIsReady *concurrency.Signal) common.SensorComponent {
	return &handlerImpl{
		client:                 client,
		complianceOperatorInfo: complianceOperatorInfo,

		request:  make(chan *central.ComplianceRequest),
		response: make(chan *message.ExpiringMessage),

		started:               &atomic.Bool{},
		disabled:              concurrency.NewSignal(),
		stopSignal:            concurrency.NewSignal(),
		complianceIsReady:     coIsReady,
		handlerMaxRetries:     defaultMaxRetries,
		handlerAPICallTimeout: defaultAPICallTimeout,
		handlerRetryTimeout:   defaultRetryTimeout,

		roleReconcileTrigger:        make(chan struct{}, 1),
		nodeRoleReconcilePassBudget: defaultNodeRoleReconcilePassBudget,
		nodeRoleReconcileInterval:   resolveNodeRoleReconcileInterval(),
	}
}

func (m *handlerImpl) Start() error {
	// Guards against a double-Start leaking a second run() goroutine and starting a second MachineConfigPool
	// informer.
	if !m.started.CompareAndSwap(false, true) {
		return nil
	}

	// Add/Done are at the call site, not inside the functions, so tests can still call them directly without
	// unbalancing backgroundWG.
	m.backgroundWG.Add(3)

	// TODO: create default scan setting for ad-hoc scan
	go func() { defer m.backgroundWG.Done(); m.run() }()

	// Runs in the background to avoid delaying Sensor's sequential component startup.
	go func() { defer m.backgroundWG.Done(); m.startMCPRoleInformer() }()

	// One-shot reconcile in case informer triggers arrive before complianceIsReady
	go func() { defer m.backgroundWG.Done(); m.reconcileOnceReady() }()

	return nil
}

// reconcileOnceReady requests one reconcile when compliance becomes ready, or returns if the handler stops first.
func (m *handlerImpl) reconcileOnceReady() {
	select {
	case <-m.complianceIsReady.Done():
		m.requestReconcile()
	case <-m.stopSignal.Done():
	}
}

func (m *handlerImpl) Stop() {
	m.stopSignal.Signal()
	// The background goroutines all derive their contexts from stopSignal, so this wait returns promptly once
	// signalled. It makes Stop() a true join point: no goroutine from this handler outlives the Stop() call.
	m.backgroundWG.Wait()
}

func (m *handlerImpl) Notify(_ common.SensorComponentEvent) {}

func (m *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	if syncScanConfigsOnStartup.BooleanSetting() {
		return []centralsensor.SensorCapability{centralsensor.ComplianceV2ScanConfigSync}
	}
	return nil
}

func (m *handlerImpl) Accepts(msg *central.MsgToSensor) bool {
	return msg.GetComplianceRequest() != nil
}

func (m *handlerImpl) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
	req := msg.GetComplianceRequest()
	if req == nil {
		return nil
	}
	// Sync Scan Configs is done during the syncing process between Sensor and Central
	if _, ok := req.GetRequest().(*central.ComplianceRequest_SyncScanConfigs); ok {
		log.Info("received scan config sync from central")
		return m.handleSyncScanCfgRequest(req.GetSyncScanConfigs())
	}

	if !m.started.Load() {
		return errors.Errorf("the compliance operator handler was not started ignoring message %v to avoid blocking", msg.GetComplianceRequest())
	}

	select {
	case <-ctx.Done():
		// TODO(ROX-30333): Pass this context together with `req` to `m.request`
		return errors.Wrapf(ctx.Err(), "message processing in component %s", m.Name())
	case m.request <- req:
		return nil
	case <-m.stopSignal.Done():
		return errors.Errorf("Could not process compliance request: %s", protoutils.NewWrapper(msg))
	}
}

func (m *handlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return m.response
}

func (m *handlerImpl) run() {
	// Safety-net reconcile tick: catches passes skipped or cut short between MachineConfigPool events.
	// reconcileNodeRoles is idempotent and a no-op once converged.
	reconcileTicker := time.NewTicker(m.nodeRoleReconcileInterval)
	defer reconcileTicker.Stop()

	for !m.stopSignal.IsDone() {
		select {
		case req := <-m.request:
			var requestProcessed bool
			operationName := fmt.Sprintf("%T", req.GetRequest())
			switch r := req.GetRequest().(type) {
			case *central.ComplianceRequest_EnableCompliance:
				requestProcessed = m.enableCompliance(r.EnableCompliance)
			case *central.ComplianceRequest_DisableCompliance:
				requestProcessed = m.disableCompliance(r.DisableCompliance)
			case *central.ComplianceRequest_ApplyScanConfig:
				requestProcessed = m.processApplyScanCfgRequest(r.ApplyScanConfig)
			case *central.ComplianceRequest_DeleteScanConfig:
				requestProcessed = m.processDeleteScanCfgRequest(r.DeleteScanConfig)
			}
			commandsFromCentral.With(prometheus.Labels{
				"operation": operationName,
				"processed": strconv.FormatBool(requestProcessed)},
			).Inc()

			if !requestProcessed {
				log.Errorf("Could not send response for compliance request: %s", protoutils.NewWrapper(req))
			}
		case <-m.roleReconcileTrigger:
			log.Debugf("Node-role reconcile triggered by MachineConfigPool informer")
			m.reconcileNodeRoles()
		case <-reconcileTicker.C:
			log.Debugf("Node-role reconcile triggered by the periodic safety-net tick")
			m.reconcileNodeRoles()
		case <-m.stopSignal.Done():
			return
		}
	}
}

func (m *handlerImpl) enableCompliance(request *central.EnableComplianceRequest) bool {
	m.disabled.Reset()
	// Any MachineConfigPool events that arrived while compliance was disabled were dropped by
	// reconcileNodeRoles' gate, so request one reconcile now that it's active again. This mirrors
	// reconcileOnceReady, which covers the other gate: compliance not yet ready at startup.
	m.requestReconcile()
	return m.composeAndSendEnableComplianceResponse(request.GetId(), nil)
}

func (m *handlerImpl) disableCompliance(request *central.DisableComplianceRequest) bool {
	// Disabling compliance should not disable compliance operator status monitoring. Users must be informed about
	// the compliance operator health.

	// TODO: Pause all scans. Currently, the only way a compliance scan can be disabled is by deleting the ScanSettingBinding.
	// TODO: Drop all custom resource listener events.
	m.disabled.Signal()
	return m.composeAndSendDisableComplianceResponse(request.GetId(), nil)
}

func (m *handlerImpl) processApplyScanCfgRequest(request *central.ApplyComplianceScanConfigRequest) bool {
	select {
	case <-m.disabled.Done():
		err := errors.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(request))
		return m.composeAndSendApplyScanConfigResponse(request.GetId(), err)
	case <-m.stopSignal.Done():
		return true
	default:
		if request.GetScanRequest() == nil {
			applyScanConfigCommands.WithLabelValues("nil").Inc()
			return m.composeAndSendApplyScanConfigResponse(request.GetId(), errors.New("Compliance scan request is empty"))
		}

		applyScanConfigCommands.WithLabelValues(fmt.Sprintf("%T", request.GetScanRequest())).Inc()
		switch r := request.GetScanRequest().(type) {
		case *central.ApplyComplianceScanConfigRequest_ScheduledScan_:
			return m.processScheduledScanRequest(request.GetId(), r.ScheduledScan)
		case *central.ApplyComplianceScanConfigRequest_SuspendScan:
			return m.processSuspendScheduledScanRequest(request.GetId(), r.SuspendScan)
		case *central.ApplyComplianceScanConfigRequest_ResumeScan:
			return m.processResumeScheduledScanRequest(request.GetId(), r.ResumeScan)
		case *central.ApplyComplianceScanConfigRequest_RerunScan:
			return m.processRerunScheduledScanRequest(request.GetId(), r.RerunScan)
		case *central.ApplyComplianceScanConfigRequest_UpdateScan:
			return m.processUpdateScanRequest(request.GetId(), r.UpdateScan)
		default:
			return m.composeAndSendApplyScanConfigResponse(request.GetId(), errors.New("Cannot handle compliance scan request"))
		}
	}
}

func (m *handlerImpl) processScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_ScheduledScan) bool {
	if err := validateApplyScheduledScanConfigRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}

	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}

	return m.createScanResources(requestID, ns, request.GetScanSettings(), request.GetCron(), m.discoverDefaultNodeRoles().rolesForCreate())
}

func (m *handlerImpl) createScanResources(requestID string, ns string, request *central.ApplyComplianceScanConfigRequest_BaseScanSettings, cron string, roles []string) bool {
	scanSetting, err := runtimeObjToUnstructured(convertCentralRequestToScanSetting(ns, request, cron, roles))
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	scanSettingBinding, err := runtimeObjToUnstructured(convertCentralRequestToScanSettingBinding(ns, request, ""))
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	err = m.callWithRetry(func(ctx context.Context) error {
		_, err = m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns).Create(ctx, scanSetting, v1.CreateOptions{})
		return errors.Wrapf(err, "Could not create namespaces/%s/scansettings/%s", ns, scanSetting.GetName())
	})
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	err = m.callWithRetry(func(ctx context.Context) error {
		_, err = m.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns).Create(ctx, scanSettingBinding, v1.CreateOptions{})
		return errors.Wrapf(err, "Could not create namespaces/%s/scansettingbindings/%s", ns, scanSettingBinding.GetName())
	})
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) processUpdateScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) bool {
	if err := validateUpdateScheduledScanConfigRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}

	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}

	// Retrieve the ScanSetting and ScanSettingBinding objects for update
	resSS := m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
	var ssObj *unstructured.Unstructured
	err := m.callWithRetry(func(ctx context.Context) error {
		var err error
		ssObj, err = resSS.Get(ctx, request.GetScanSettings().GetScanName(), v1.GetOptions{})
		return errors.Wrapf(err, "namespaces/%s/scansettings/%s not found.  Treating as a create", ns, request.GetScanSettings().GetScanName())
	})
	if err != nil {
		log.Warn(err)
	}

	resSSB := m.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns)
	var ssbObj *unstructured.Unstructured
	err = m.callWithRetry(func(ctx context.Context) error {
		var err error
		ssbObj, err = resSSB.Get(ctx, request.GetScanSettings().GetScanName(), v1.GetOptions{})
		return errors.Wrapf(err, "namespaces/%s/scansettingsbindings/%s not found.  Treating as a create", ns, request.GetScanSettings().GetScanName())
	})
	if err != nil {
		log.Warn(err)
	}

	// Compute the node roles once for this request and reuse them below. If discovery is not authoritative and ssObj
	// already exists, its current roles are preserved instead of being replaced by the static fallback.
	roles := m.discoverDefaultNodeRoles().rolesForUpdate(ssObj)

	if ssObj == nil && ssbObj == nil {
		// This is an add instead
		return m.createScanResources(requestID, ns, request.GetScanSettings(), request.GetCron(), roles)
	}

	// Invalid case because scan setting is created first, so we should not have a situation where
	// we have a scan setting binding and not a scan setting.  This probably isn't even possible.
	if ssObj == nil {
		err = errors.Wrap(err, "Could not convert unstructured to scan setting")
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	updatedScanSetting, err := updateScanSettingFromUpdateRequest(ssObj, request, roles)
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	var updatedScanSettingBinding *unstructured.Unstructured
	// It is possible that we successfully created the scan setting but had issues on the binding.
	if ssbObj != nil {
		updatedScanSettingBinding, err = updateScanSettingBindingFromUpdateRequest(ssbObj, request)
		if err != nil {
			return m.composeAndSendApplyScanConfigResponse(requestID, err)
		}

	} else {
		updatedScanSettingBinding, err = runtimeObjToUnstructured(convertCentralRequestToScanSettingBinding(ns, request.GetScanSettings(), ""))
		if err != nil {
			return m.composeAndSendApplyScanConfigResponse(requestID, err)
		}
	}

	err = m.callWithRetryWithOnConflictCallback(
		func(ctx context.Context) error {
			if updatedScanSetting == nil {
				return retry.MakeRetryable(errors.Errorf("updated ScanSetting %q is 'nil'", request.GetScanSettings().GetScanName()))
			}
			_, err := resSS.Update(ctx, updatedScanSetting, v1.UpdateOptions{})
			return errors.Wrapf(err, "Could not update namespaces/%s/scansettings/%s", ns, updatedScanSetting.GetName())
		},
		func(ctx context.Context) error {
			ssObj, err = resSS.Get(ctx, request.GetScanSettings().GetScanName(), v1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, "unable to get namespaces/%s/scansettings/%s", ns, request.GetScanSettings().GetScanName())
			}
			updatedScanSetting, err = updateScanSettingFromUpdateRequest(ssObj, request, roles)
			if err != nil {
				return err
			}
			return nil
		})

	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	// Process SSB as an update
	if ssbObj != nil {
		err = m.callWithRetryWithOnConflictCallback(
			func(ctx context.Context) error {
				if updatedScanSettingBinding == nil {
					return retry.MakeRetryable(errors.Errorf("updated ScanSettingBinding %q is 'nil'", request.GetScanSettings().GetScanName()))
				}
				_, err = resSSB.Update(ctx, updatedScanSettingBinding, v1.UpdateOptions{})
				return errors.Wrapf(err, "Could not update namespaces/%s/scansettingbindings/%s", ns, updatedScanSettingBinding.GetName())
			},
			func(ctx context.Context) error {
				ssbObj, err = resSSB.Get(ctx, request.GetScanSettings().GetScanName(), v1.GetOptions{})
				if err != nil {
					return errors.Wrapf(err, "unable to get namespaces/%s/scansettingbindings/%s", ns, request.GetScanSettings().GetScanName())
				}
				updatedScanSettingBinding, err = updateScanSettingBindingFromUpdateRequest(ssbObj, request)
				if err != nil {
					return err
				}
				return nil
			})

		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	err = m.callWithRetry(func(ctx context.Context) error {
		_, err = m.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns).Create(ctx, updatedScanSettingBinding, v1.CreateOptions{})
		return errors.Wrapf(err, "Could not create namespaces/%s/scansettingbindings/%s", ns, updatedScanSettingBinding.GetName())
	})

	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) processRerunScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_RerunScheduledScan) bool {
	if err := validateApplyRerunScheduledScanRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}

	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}

	// Conventionally, compliance scan can be rerun by applying an
	// annotation to ComplianceScan CR. The ComplianceScan CRs can be
	// found from the ComplianceSuite CR. Note that the ComplianceSuite
	// created from a scan configuration have the same name as scan
	// configuration.
	resI := m.client.Resource(complianceoperator.ComplianceSuite.GroupVersionResource()).Namespace(ns)
	var obj *unstructured.Unstructured
	err := m.callWithRetry(func(ctx context.Context) error {
		var err error
		obj, err = resI.Get(ctx, request.GetScanName(), v1.GetOptions{})
		return errors.Wrapf(err, "namespaces/%s/compliancesuites/%s not found", ns, request.GetScanName())
	})
	if err != nil || obj == nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	var complianceSuite v1alpha1.ComplianceSuite
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &complianceSuite); err != nil {
		err = errors.Wrap(err, "Could not convert unstructured to compliance suite")
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	// Apply annotation to indicate compliance scan be rerun.
	for _, scan := range complianceSuite.Spec.Scans {
		if err := m.reRunScan(scan, ns); err != nil {
			return m.composeAndSendApplyScanConfigResponse(requestID, err)
		}

	}
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) reRunScan(scan v1alpha1.ComplianceScanSpecWrapper, ns string) error {
	var obj *unstructured.Unstructured
	resI := m.client.Resource(complianceoperator.ComplianceScan.GroupVersionResource()).Namespace(ns)
	// Get the scan and apply annotation to indicate compliance scan be rerun.
	complianceScan, scanObj, err := m.getScanForReRun(func() (*unstructured.Unstructured, error) {
		err := m.callWithRetry(func(ctx context.Context) error {
			var err error
			obj, err = resI.Get(ctx, scan.Name, v1.GetOptions{})
			return errors.Wrapf(err, "namespaces/%s/compliancescans/%s not found", ns, scan.Name)
		})
		return obj, err
	}, scan.Name)
	if err != nil {
		return err
	}
	return m.callWithRetryWithOnConflictCallback(
		func(ctx context.Context) error {
			log.Infof("Rerunning compliance scan %s", complianceScan)
			if scanObj == nil {
				return retry.MakeRetryable(errors.Errorf("Scan %q is 'nil'", complianceScan))
			}
			_, err = resI.Update(ctx, scanObj, v1.UpdateOptions{})
			return errors.Wrapf(err, "Could not update namespaces/%s/compliancescans/%s", ns, complianceScan)
		},
		func(ctx context.Context) error {
			complianceScan, scanObj, err = m.getScanForReRun(func() (*unstructured.Unstructured, error) {
				return resI.Get(ctx, complianceScan, v1.GetOptions{})
			}, scan.Name)
			if err != nil {
				return err
			}
			return nil
		})
}

func (m *handlerImpl) getScanForReRun(getFunc func() (*unstructured.Unstructured, error), scanName string) (string, *unstructured.Unstructured, error) {
	var obj *unstructured.Unstructured
	obj, err := getFunc()
	if err != nil || obj == nil {
		return scanName, nil, err
	}
	var complianceScan v1alpha1.ComplianceScan
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &complianceScan); err != nil {
		err = errors.Wrap(err, "Could not convert unstructured to compliance scan")
		return scanName, nil, err
	}

	if complianceScan.GetAnnotations() == nil {
		complianceScan.Annotations = make(map[string]string)
	}
	complianceScan.Annotations[rescanAnnotation] = ""

	obj, err = runtimeObjToUnstructured(&complianceScan)
	if err != nil {
		return complianceScan.Name, nil, err
	}
	return complianceScan.Name, obj, nil
}

func (m *handlerImpl) processScanConfigScheduleChangeRequest(requestID string, config scanScheduleConfiguration) bool {
	if err := config.ValidationFunc(config.Request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}

	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}

	obj, err := m.getScanSettingForUpdate(func() (*unstructured.Unstructured, error) {
		resI := m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
		var obj *unstructured.Unstructured
		err := m.callWithRetry(func(ctx context.Context) error {
			var err error
			obj, err = resI.Get(ctx, config.ScanName, v1.GetOptions{})
			return errors.Wrapf(err, "namespaces/%s/scansettings/%s not found", ns, config.ScanName)
		})
		return obj, err
	}, config)
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}
	resI := m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
	err = m.callWithRetryWithOnConflictCallback(
		func(ctx context.Context) error {
			if obj == nil {
				return retry.MakeRetryable(errors.Errorf("ScanSetting %q is 'nil'", config.ScanName))
			}
			_, err := resI.Update(ctx, obj, v1.UpdateOptions{})
			return errors.Wrapf(err, "Could not update namespaces/%s/scansettings/%s", ns, config.ScanName)
		},
		func(ctx context.Context) error {
			obj, err = m.getScanSettingForUpdate(func() (*unstructured.Unstructured, error) {
				return resI.Get(ctx, config.ScanName, v1.GetOptions{})
			}, config)
			if err != nil {
				return errors.Wrapf(err, "unable to retrieve ScanSetting %s", config.ScanName)
			}
			return nil
		})

	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) getScanSettingForUpdate(getFunc func() (*unstructured.Unstructured, error), config scanScheduleConfiguration) (*unstructured.Unstructured, error) {
	var obj *unstructured.Unstructured
	obj, err := getFunc()
	if err != nil || obj == nil {
		return nil, err
	}

	var scanSetting v1alpha1.ScanSetting
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &scanSetting); err != nil {
		err = errors.Wrap(err, "Could not convert unstructured to scan setting")
		return nil, err
	}

	// Check if scanSetting has the "Suspend" field
	if _, ok := reflect.TypeFor[v1alpha1.ScanSetting]().FieldByName("Suspend"); ok {
		scanSetting.Suspend = *config.Suspend
	} else {
		// Handle the case where the field doesn't exist (older CRD)
		return nil, errors.New("suspending a scan is not supported on this version of the compliance operator")
	}

	obj, err = runtimeObjToUnstructured(&scanSetting)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func validateInterface(i interface{}) error {
	switch req := i.(type) {
	case *central.ApplyComplianceScanConfigRequest_SuspendScheduledScan:
		return validateApplySuspendScheduledScanRequest(req)
	case *central.ApplyComplianceScanConfigRequest_ResumeScheduledScan:
		return validateApplyResumeScheduledScanRequest(req)
	// Add other cases as needed for other request types.
	// ex. we might need be adding suppport for changing schedule for scheduled scans.
	default:
		return errors.Errorf("Cannot validate request of type %T", i)
	}
}

func (m *handlerImpl) processSuspendScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_SuspendScheduledScan) bool {
	config := scanScheduleConfiguration{
		Suspend:        pointer.Bool(true),
		ScanName:       request.GetScanName(),
		Request:        request,
		ValidationFunc: validateInterface,
	}
	return m.processScanConfigScheduleChangeRequest(requestID, config)
}

func (m *handlerImpl) processResumeScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_ResumeScheduledScan) bool {
	config := scanScheduleConfiguration{
		Suspend:        pointer.Bool(false),
		ScanName:       request.GetScanName(),
		Request:        request,
		ValidationFunc: validateInterface,
	}
	return m.processScanConfigScheduleChangeRequest(requestID, config)
}

func (m *handlerImpl) processDeleteScanCfgRequest(request *central.DeleteComplianceScanConfigRequest) bool {
	select {
	case <-m.disabled.Done():
		err := errors.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(request))
		return m.composeAndSendDeleteResponse(request.GetId(), "", err)
	case <-m.stopSignal.Done():
		return true
	default:
		if request.GetName() == defaultScanSettingName {
			return m.composeAndSendDeleteResponse(request.GetId(), "", errors.Errorf("Default compliance scan configuration %q cannot be deleted", defaultScanSettingName))
		}

		// Delete ScanSetting and ScanSettingBinding custom resource. Deleting ScanSettingBindings deletes all owned resources.
		// Each ad-hoc scan creates a unique ScanSettingBinding that reuses default ACS ScanSetting CR.
		// Each scheduled scan configuration has a uniquely identifiable ScanSetting and ScanSettingBinding.
		ns := m.complianceOperatorInfo.GetNamespace()
		if ns == "" {
			return m.composeAndSendDeleteResponse(request.GetId(), "", errors.New("Compliance operator not known"))
		}
		deletePolicy := v1.DeletePropagationForeground
		scanSettingBindingResourceI := m.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns)
		err := m.callWithRetry(func(ctx context.Context) error {
			return scanSettingBindingResourceI.Delete(ctx, request.GetName(), v1.DeleteOptions{PropagationPolicy: &deletePolicy})
		})
		if err != nil && !kubeAPIErr.IsNotFound(err) {
			return m.composeAndSendDeleteResponse(request.GetId(), fmt.Sprintf("scansettingbindings/%s", request.GetName()), err)
		}

		scanSettingResourceI := m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
		err = m.callWithRetry(func(ctx context.Context) error {
			return scanSettingResourceI.Delete(ctx, request.GetName(), v1.DeleteOptions{PropagationPolicy: &deletePolicy})
		})
		if err != nil && !kubeAPIErr.IsNotFound(err) {
			return m.composeAndSendDeleteResponse(request.GetId(), fmt.Sprintf("scansettings/%s", request.GetName()), err)
		}
		return m.composeAndSendDeleteResponse(request.GetId(), "", nil)
	}
}

// handleSyncScanCfgRequest spawns its own goroutine (pre-existing, kept as-is here) that calls processSyncScanCfg
// directly instead of going through the single-writer run() loop, so it can race reconcileNodeRoles (also writing
// ScanSettings) on the same objects. This is an accepted trade-off, not a bug: both paths go through
// callWithRetryWithOnConflictCallback, so a lost race surfaces as a Conflict that gets retried with the latest
// object, and the two converge on a consistent final state instead of silently clobbering each other.
func (m *handlerImpl) handleSyncScanCfgRequest(request *central.SyncComplianceScanConfigRequest) error {
	go func() {
		select {
		case <-m.disabled.Done():
			log.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(request))
			return
		case <-m.stopSignal.Done():
			return
		case <-m.complianceIsReady.Done():
			log.Debugf("compliance is ready. Starting the reconciliation of %d scan configs", len(request.GetScanConfigs()))
			if err := m.processSyncScanCfg(request); err != nil {
				log.Error(err)
				return
			}
		}
	}()
	return nil
}

func generateScanIndex(namespace string, scanName string) string {
	return fmt.Sprintf("%s-%s", namespace, scanName)
}

func (m *handlerImpl) getResourcesInCluster(api k8sapi.APIResource) (map[string]unstructured.Unstructured, error) {
	return m.getResourcesInClusterCtx(m.ctx(), api)
}

// getResourcesInClusterCtx is getResourcesInCluster with an explicit parent context, so a caller can bound the
// retry window tighter than handlerRetryTimeout (e.g. a reconcile pass with its own overall deadline, see
// nodeRoleReconcilePassBudget), instead of always retrying against the handler's full lifetime context.
func (m *handlerImpl) getResourcesInClusterCtx(ctx context.Context, api k8sapi.APIResource) (map[string]unstructured.Unstructured, error) {
	resourceInterface := m.client.Resource(api.GroupVersionResource())
	var resourcesInCluster *unstructured.UnstructuredList
	err := m.callWithRetryCtx(ctx, func(ctx context.Context) error {
		var err error
		resourcesInCluster, err = resourceInterface.List(ctx, v1.ListOptions{LabelSelector: labels.SelectorFromSet(stackroxLabels).String()})
		return errors.Wrap(err, "listing resources in cluster")
	})
	if err != nil {
		return nil, err
	}
	resourcesInClusterMap := make(map[string]unstructured.Unstructured)
	for _, resource := range resourcesInCluster.Items {
		log.Debugf("%s in the cluster: %s", api.Kind, resource.GetName())
		resourcesInClusterMap[generateScanIndex(resource.GetNamespace(), resource.GetName())] = resource
	}
	return resourcesInClusterMap, nil
}

func (m *handlerImpl) reconcileCreateOrUpdateResource(
	namespace string,
	req *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan,
	inClusterResources map[string]unstructured.Unstructured,
	updateFn updateFunction,
	convertFn convertFunction,
	api k8sapi.APIResource,
) error {
	namespaceNameIndex := generateScanIndex(namespace, req.GetScanSettings().GetScanName())
	if resource, isInCluster := inClusterResources[namespaceNameIndex]; isInCluster {
		// Update Resource
		log.Debugf("Update %s %s", api.Kind, req.GetScanSettings().GetScanName())
		delete(inClusterResources, namespaceNameIndex)
		updatedResource, err := updateFn(&resource, req)
		if err != nil {
			return err
		}
		err = m.callWithRetryWithOnConflictCallback(
			func(ctx context.Context) error {
				if updatedResource == nil {
					return retry.MakeRetryable(errors.Errorf("updated %s %q is 'nil'", api.GroupVersionResource(), resource.GetName()))
				}
				_, err := m.client.Resource(api.GroupVersionResource()).Namespace(namespace).Update(ctx, updatedResource, v1.UpdateOptions{})
				return errors.Wrapf(err, "updating namespace %q", namespace)
			},
			func(ctx context.Context) error {
				currentResource, err := m.client.Resource(api.GroupVersionResource()).Namespace(namespace).Get(ctx, resource.GetName(), v1.GetOptions{})
				if err != nil {
					return errors.Wrapf(err, "unable to get namespaces/%s/scans/%s", namespace, resource.GetName())
				}
				updatedResource, err = updateFn(currentResource, req)
				if err != nil {
					return err
				}
				return nil
			})
		if err != nil {
			return err
		}
	} else {
		// The Resource is in Central but not in the cluster
		log.Debugf("Create %s %s", api.Kind, req.GetScanSettings().GetScanName())
		resource, err := runtimeObjToUnstructured(convertFn(namespace, req.GetScanSettings(), req.GetCron()))
		if err != nil {
			return err
		}
		err = m.callWithRetry(func(ctx context.Context) error {
			_, err := m.client.Resource(api.GroupVersionResource()).Namespace(namespace).Create(ctx, resource, v1.CreateOptions{})
			return errors.Wrapf(err, "Could not create namespaces/%s/%s/%s", namespace, api.Name, resource.GetName())
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *handlerImpl) reconcileDeleteResource(inCentral set.StringSet, inClusterResources map[string]unstructured.Unstructured, api k8sapi.APIResource) error {
	var errList errorhelpers.ErrorList
	deletePolicy := v1.DeletePropagationForeground
	// Delete Resources that are no longer in Central
	for index, resource := range inClusterResources {
		if !inCentral.Contains(index) {
			// The Resource is in the cluster but not in Central
			log.Debugf("Delete %s %s", api.Kind, resource.GetName())
			cli := m.client.Resource(api.GroupVersionResource()).Namespace(resource.GetNamespace())
			err := m.callWithRetry(func(ctx context.Context) error {
				return cli.Delete(ctx, resource.GetName(), v1.DeleteOptions{PropagationPolicy: &deletePolicy})
			})
			if err != nil && !kubeAPIErr.IsNotFound(err) {
				errList.AddError(err)
			}
		}
	}
	return errList.ToError()
}

func (m *handlerImpl) processSyncScanCfg(request *central.SyncComplianceScanConfigRequest) error {
	// List all ScanSettings in the cluster with the `stackrox` label.
	scanSettingsInCluster, err := m.getResourcesInCluster(complianceoperator.ScanSetting)
	if err != nil {
		return err
	}
	// List all ScanSettingBindings in the cluster with the `stackrox` label.
	scanSettingBindingsInCluster, err := m.getResourcesInCluster(complianceoperator.ScanSettingBinding)
	if err != nil {
		return err
	}

	complianceNamespace := m.complianceOperatorInfo.GetNamespace()
	if complianceNamespace == "" {
		return errors.New("Compliance operator namespace not known")
	}

	// Run discovery once for this sync batch and reuse the outcome for every ScanSetting below. When discovery is
	// not authoritative, an existing ScanSetting keeps its current roles instead of losing them to the static
	// fallback; a new one still gets the fallback.
	disc := m.discoverDefaultNodeRoles()
	convertScanSetting := func(namespace string, req *central.ApplyComplianceScanConfigRequest_BaseScanSettings, cron string) runtime.Object {
		return convertCentralRequestToScanSetting(namespace, req, cron, disc.rolesForUpdate(nil))
	}
	updateScanSetting := func(obj *unstructured.Unstructured, req *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) (*unstructured.Unstructured, error) {
		return updateScanSettingFromUpdateRequest(obj, req, disc.rolesForUpdate(obj))
	}

	// Compare with the ScanConfig in the request.
	var errList errorhelpers.ErrorList
	inCentralSet := set.NewStringSet()
	for _, scanCfg := range request.GetScanConfigs() {
		if err := validateUpdateScheduledScanConfigRequest(scanCfg.GetUpdateScan()); err != nil {
			errList.AddError(err)
			continue
		}
		inCentralSet.Add(generateScanIndex(complianceNamespace, scanCfg.GetUpdateScan().GetScanSettings().GetScanName()))
		// Reconcile ScanSetting
		if err := m.reconcileCreateOrUpdateResource(
			complianceNamespace,
			scanCfg.GetUpdateScan(),
			scanSettingsInCluster,
			updateScanSetting,
			convertScanSetting,
			complianceoperator.ScanSetting,
		); err != nil {
			errList.AddError(err)
		}
		// Reconcile ScanSettingBinding
		if err := m.reconcileCreateOrUpdateResource(
			complianceNamespace,
			scanCfg.GetUpdateScan(),
			scanSettingBindingsInCluster,
			updateScanSettingBindingFromUpdateRequest,
			convertCentralRequestToScanSettingBinding,
			complianceoperator.ScanSettingBinding,
		); err != nil {
			errList.AddError(err)
		}
	}
	// Delete ScanSettingBindings that are no longer in Central
	if err := m.reconcileDeleteResource(inCentralSet, scanSettingBindingsInCluster, complianceoperator.ScanSettingBinding); err != nil {
		errList.AddError(err)
	}
	// Delete ScanSettings that are no longer in Central
	if err := m.reconcileDeleteResource(inCentralSet, scanSettingsInCluster, complianceoperator.ScanSetting); err != nil {
		errList.AddError(err)
	}
	return errList.ToError()
}

func (m *handlerImpl) composeAndSendEnableComplianceResponse(requestID string, err error) bool {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_EnableComplianceResponse_{
			EnableComplianceResponse: &central.ComplianceResponse_EnableComplianceResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		err = errors.Wrap(err, "Could not enable compliance")
		log.Error(err)

		msg.GetEnableComplianceResponse().Payload = &central.ComplianceResponse_EnableComplianceResponse_Error{
			Error: err.Error(),
		}
	}
	return m.sendResponse(msg)
}

func (m *handlerImpl) composeAndSendDisableComplianceResponse(requestID string, err error) bool {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_DisableComplianceResponse_{
			DisableComplianceResponse: &central.ComplianceResponse_DisableComplianceResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		err = errors.Wrap(err, "Could not enable compliance")
		log.Error(err)

		msg.GetDisableComplianceResponse().Payload = &central.ComplianceResponse_DisableComplianceResponse_Error{
			Error: err.Error(),
		}
	}
	return m.sendResponse(msg)
}

func (m *handlerImpl) composeAndSendApplyScanConfigResponse(requestID string, err error) bool {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_ApplyComplianceScanConfigResponse_{
			ApplyComplianceScanConfigResponse: &central.ComplianceResponse_ApplyComplianceScanConfigResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		err = errors.Wrap(err, "Could not apply compliance scan configuration")
		log.Error(err)

		msg.GetApplyComplianceScanConfigResponse().Payload = &central.ComplianceResponse_ApplyComplianceScanConfigResponse_Error{
			Error: err.Error(),
		}
	}
	return m.sendResponse(msg)
}

func (m *handlerImpl) composeAndSendDeleteResponse(requestID string, resource string, err error) bool {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_DeleteComplianceScanConfigResponse_{
			DeleteComplianceScanConfigResponse: &central.ComplianceResponse_DeleteComplianceScanConfigResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		if resource != "" {
			err = errors.Wrapf(err, "Could not delete namespaces/%s/%s", m.complianceOperatorInfo.GetNamespace(), resource)
		}
		log.Error(err)
		msg.GetDeleteComplianceScanConfigResponse().Payload = &central.ComplianceResponse_DeleteComplianceScanConfigResponse_Error{
			Error: err.Error(),
		}
	}
	return m.sendResponse(msg)
}

func (m *handlerImpl) sendResponse(response *central.ComplianceResponse) bool {
	select {
	case m.response <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ComplianceResponse{
			ComplianceResponse: response,
		},
	}):
		return true
	case <-m.stopSignal.Done():
		return false
	}
}

func (m *handlerImpl) ctx() context.Context {
	return concurrency.AsContext(&m.stopSignal)
}

type retriableCall func(context.Context) error

type onRetriableCallError func(context.Context, error) error

type onConflictCall func(context.Context) error

func onConflictErrorWrapper(fn onConflictCall) onRetriableCallError {
	return func(ctx context.Context, parentErr error) error {
		if !kubeAPIErr.IsConflict(parentErr) && !retry.IsRetryable(parentErr) {
			return nil
		}
		if fn == nil {
			return errors.New("onConflictCall is 'nil'")
		}
		return fn(ctx)
	}
}

func (m *handlerImpl) callWithRetry(fn retriableCall) error {
	return m.callWithRetryCtx(m.ctx(), fn)
}

// callWithRetryCtx is callWithRetry with an explicit parent context.
func (m *handlerImpl) callWithRetryCtx(ctx context.Context, fn retriableCall) error {
	retryCtx, cancel := context.WithTimeout(ctx, m.handlerRetryTimeout)
	defer cancel()
	return m.callWithRetryWithOnErrorCallback(retryCtx, fn, nil)
}

func (m *handlerImpl) callWithRetryWithOnConflictCallback(fn retriableCall, onConflict onConflictCall) error {
	return m.callWithRetryWithOnConflictCallbackCtx(m.ctx(), fn, onConflict)
}

// callWithRetryWithOnConflictCallbackCtx is callWithRetryWithOnConflictCallback with an explicit parent context.
func (m *handlerImpl) callWithRetryWithOnConflictCallbackCtx(ctx context.Context, fn retriableCall, onConflict onConflictCall) error {
	retryCtx, cancel := context.WithTimeout(ctx, m.handlerRetryTimeout)
	defer cancel()
	return m.callWithRetryWithOnErrorCallback(retryCtx, fn, onConflictErrorWrapper(onConflict))
}

func (m *handlerImpl) callWithRetryWithOnErrorCallback(retryCtx context.Context, fn retriableCall, onErrFn onRetriableCallError) error {
	if fn == nil {
		return errors.New("retriableCall is 'nil'")
	}
	return retry.WithRetry(func() error {
		callCtx, callCancel := context.WithTimeout(retryCtx, m.handlerAPICallTimeout)
		defer callCancel()
		err := fn(callCtx)
		if err != nil && onErrFn != nil {
			errCallCtx, errCallCancel := context.WithTimeout(retryCtx, m.handlerAPICallTimeout)
			defer errCallCancel()
			if onErr := onErrFn(errCallCtx, err); onErr != nil {
				// propagate the onErrFn callback failure and keep the conflict as context
				return errors.Wrapf(onErr, "callback error: %v", err)
			}
		}
		return err
	},
		retry.WithContext(retryCtx),
		retry.Tries(m.handlerMaxRetries),
		retry.WithExponentialBackoff())
}
