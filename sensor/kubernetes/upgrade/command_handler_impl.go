package upgrade

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/message"
	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	log                             = logging.LoggerForModule()
	_   common.CentralGRPCConnAware = (*commandHandler)(nil)
	_   common.SensorComponent      = (*commandHandler)(nil)
)

type commandHandler struct {
	stopSig concurrency.Signal

	currentProcess      *process
	currentProcessMutex sync.Mutex
	baseK8sRESTConfig   *rest.Config
	k8sClient           kubernetes.Interface
	checkInClient       central.SensorUpgradeControlServiceClient

	configHandler config.Handler
}

// NewCommandHandler returns a new upgrade command handler for Kubernetes.
func NewCommandHandler(configHandler config.Handler) (common.SensorComponent, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining in-cluster Kubernetes config")
	}

	k8sClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "create Kubernetes clientset")
	}

	return &commandHandler{
		baseK8sRESTConfig: config,
		k8sClient:         k8sClientSet,
		configHandler:     configHandler,
	}, nil
}

// SetCentralGRPCClient sets the central gRPC connection
func (h *commandHandler) SetCentralGRPCClient(cc grpc.ClientConnInterface) {
	h.checkInClient = central.NewSensorUpgradeControlServiceClient(cc)
}

func (h *commandHandler) Start() error {
	h.stopSig.Reset()
	return nil
}

func (h *commandHandler) Stop(_ error) {
	h.stopSig.Signal()
}

func (h *commandHandler) Notify(common.SensorComponentEvent) {}

func (h *commandHandler) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *commandHandler) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

func (h *commandHandler) waitForTermination(proc *process) {
	select {
	case <-h.stopSig.Done():
		return
	case <-proc.doneSig.Done():
	}

	procErr := proc.doneSig.Err()

	h.currentProcessMutex.Lock()
	defer h.currentProcessMutex.Unlock()

	if h.currentProcess != proc {
		return // not interesting - probably just replaced
	}

	h.currentProcess = nil
	if procErr != nil {
		log.Errorf("Active upgrade process terminated with error: %v", procErr)
	}
}

func (h *commandHandler) ProcessMessage(msg *central.MsgToSensor) error {
	trigger := msg.GetSensorUpgradeTrigger()
	if trigger == nil {
		return nil
	}

	h.currentProcessMutex.Lock()
	defer h.currentProcessMutex.Unlock()

	if h.stopSig.IsDone() {
		return errors.Errorf("unable to send command: %s", proto.MarshalTextString(trigger))
	}

	oldProcess := h.currentProcess
	if oldProcess != nil {
		if oldProcess.GetID() == trigger.GetUpgradeProcessId() {
			return nil // idempotent
		}

		// If we receive a trigger with a different ID (or no ID), we should always terminate the current process,
		// regardless of whether or not we can successfully launch a new one.
		oldProcess.Terminate(errors.New("upgrade process is no longer current"))
	}

	if trigger.GetUpgradeProcessId() == "" {
		// No upgrade should be in progress. Delete any deployment that might be lingering around.
		go h.deleteUpgraderDeployments()
		h.currentProcess = nil
		return nil
	}

	if h.configHandler.GetHelmManagedConfig() != nil && !h.configHandler.GetHelmManagedConfig().GetNotHelmManaged() {
		upgradesNotSupportedErr := errors.New("Cluster is Helm-managed and does not support auto-upgrades")
		go h.rejectUpgradeRequest(trigger, upgradesNotSupportedErr)
		go h.deleteUpgraderDeployments()
		h.currentProcess = nil
		return upgradesNotSupportedErr
	}

	newProc, err := newProcess(trigger, h.checkInClient, h.baseK8sRESTConfig)
	if err != nil {
		return errors.Wrap(err, "error creating new upgrade process")
	}

	h.currentProcess = newProc

	go newProc.Run()
	go h.waitForTermination(newProc)

	return nil
}

func (h *commandHandler) deleteUpgraderDeployments() {
	// Only try deleting once. There's no big issue if these linger around as the upgrader doesn't do anything without
	// being told to by central, so we don't go out of our way to make sure they are gone.
	err := h.k8sClient.AppsV1().Deployments(namespaces.StackRox).DeleteCollection(
		h.ctx(), pkgKubernetes.DeleteBackgroundOption, v1.ListOptions{
			LabelSelector: v1.FormatLabelSelector(&v1.LabelSelector{
				MatchExpressions: []v1.LabelSelectorRequirement{
					{Key: "app", Operator: v1.LabelSelectorOpIn, Values: []string{upgraderDeploymentName}},
					{Key: processIDLabelKey, Operator: v1.LabelSelectorOpExists},
				},
			}),
		})
	if err != nil {
		log.Errorf("Could not delete upgrader deployment: %v", err)
	}
}

func (h *commandHandler) ctx() context.Context {
	return concurrency.AsContext(&h.stopSig)
}

func (h *commandHandler) rejectUpgradeRequest(trigger *central.SensorUpgradeTrigger, errReason error) {
	checkInReq := &central.UpgradeCheckInFromSensorRequest{
		UpgradeProcessId: trigger.GetUpgradeProcessId(),
		ClusterId:        clusterid.Get(), // will definitely be available at this point
		State: &central.UpgradeCheckInFromSensorRequest_LaunchError{
			LaunchError: errReason.Error(),
		},
	}
	// We don't care about the error, if any.
	_, _ = h.checkInClient.UpgradeCheckInFromSensor(h.ctx(), checkInReq)
}
