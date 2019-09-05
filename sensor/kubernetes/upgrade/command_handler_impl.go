package upgrade

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	kubernetes2 "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/upgrade"
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
}

// NewCommandHandler returns a new upgrade command handler for Kubernetes.
func NewCommandHandler() (upgrade.CommandHandler, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining Kubernetes REST config")
	}
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating Kubernetes client set")
	}

	return &commandHandler{
		baseK8sRESTConfig: config,
		k8sClient:         k8sClient,
	}, nil
}

func (h *commandHandler) Start() {
	h.runSig.Reset()
}

func (h *commandHandler) Stop() {
	h.runSig.Signal()
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

func (h *commandHandler) SendCommand(trigger *central.SensorUpgradeTrigger) bool {
	h.currentProcessMutex.Lock()
	defer h.currentProcessMutex.Unlock()

	if h.runSig.IsDone() {
		return false
	}

	oldProcess := h.currentProcess
	if oldProcess != nil {
		if oldProcess.GetID() == trigger.GetUpgradeProcessId() {
			return true // idempotent
		}
	}

	if trigger.GetUpgradeProcessId() == "" {
		// No upgrade should be in progress. Delete any deployment that might be lingering around.
		go h.deleteUpgraderDeployments()
		h.currentProcess = nil
		return true
	}

	newProc, err := newProcess(trigger, h.baseK8sRESTConfig)
	if err != nil {
		return false
	}

	h.currentProcess = newProc
	if oldProcess != nil {
		oldProcess.Terminate(errors.Errorf("superseded by new upgrade process %s", trigger.GetUpgradeProcessId()))
	}

	go newProc.Run()
	go h.waitForTermination(newProc)

	return true
}

func (h *commandHandler) deleteUpgraderDeployments() {
	// Only try deleting once. There's no big issue if these linger around as the upgrader doesn't do anything without
	// being told to by central, so we don't go out of our way to make sure they are gone.
	err := h.k8sClient.ExtensionsV1beta1().Deployments(namespaces.StackRox).DeleteCollection(
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
