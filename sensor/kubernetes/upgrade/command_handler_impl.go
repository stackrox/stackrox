package upgrade

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/upgrade"
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
}

// NewCommandHandler returns a new upgrade command handler for Kubernetes.
func NewCommandHandler() (upgrade.CommandHandler, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining Kubernetes REST config")
	}

	return &commandHandler{
		baseK8sRESTConfig: config,
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
