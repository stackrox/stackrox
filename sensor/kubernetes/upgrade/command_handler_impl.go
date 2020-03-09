package upgrade

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	kubernetes2 "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	log = logging.LoggerForModule()
)

type commandHandler struct {
	runSig concurrency.Signal

	currentProcess      *process
	currentProcessMutex sync.Mutex
	baseK8sRESTConfig   *rest.Config
	k8sClient           kubernetes.Interface
	checkInClient       central.SensorUpgradeControlServiceClient
}

// NewCommandHandler returns a new upgrade command handler for Kubernetes.
func NewCommandHandler() (common.SensorComponent, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining in-cluster Kubernetes config")
	}

	k8sClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "create Kubernetes clientset")
	}
	conn, err := clientconn.AuthenticatedGRPCConnection(env.CentralEndpoint.Setting(), mtls.CentralSubject, clientconn.UseServiceCertToken(true))
	if err != nil {
		return nil, errors.Wrap(err, "establishing central gRPC connection")
	}

	return &commandHandler{
		baseK8sRESTConfig: config,
		k8sClient:         k8sClientSet,
		checkInClient:     central.NewSensorUpgradeControlServiceClient(conn),
	}, nil
}

func (h *commandHandler) Start() error {
	h.runSig.Reset()
	return nil
}

func (h *commandHandler) Stop(err error) {
	h.runSig.Signal()
}

func (h *commandHandler) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *commandHandler) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func (h *commandHandler) waitForTermination(proc *process) {
	select {
	case <-h.runSig.Done():
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

	if h.runSig.IsDone() {
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
		kubernetes2.DeleteBackgroundOption, v1.ListOptions{
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
