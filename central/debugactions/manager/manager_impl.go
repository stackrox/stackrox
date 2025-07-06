package manager

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
)

const maxWeight = int64(math.MaxInt64)

var log = logging.LoggerForModule()

type actionParts struct {
	actionStatus *v2.ActionStatus
	sema         *semaphore.Weighted
	ctx          context.Context
	cancelFunc   context.CancelFunc
}

type managerImpl struct {
	actionIDToParts map[string]*actionParts
	mapLock         sync.Mutex
}

func (m *managerImpl) RegisterAction(debugAction *v2.DebugAction) error {
	if buildinfo.ReleaseBuild {
		return errors.New("Debug actions are not supported for release builds")
	}
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	_, ok := m.actionIDToParts[debugAction.GetIdentifier()]
	if ok {
		return errors.Wrapf(errox.AlreadyExists, "An action with identifier '%s' is already registered", debugAction.GetIdentifier())
	}
	if debugAction.GetAction() == nil {
		return errors.Wrap(errox.InvalidArgs, "Action should define one of 'SleepAction' or 'WaitAction'")
	}

	var sema *semaphore.Weighted
	ctx, cancelFunc := context.WithCancel(context.Background())
	if debugAction.GetWaitAction() != nil {
		weight := maxWeight
		if debugAction.GetNumTimes() > 0 {
			weight = debugAction.GetNumTimes()
		}
		sema = semaphore.NewWeighted(weight)
		err := sema.Acquire(ctx, weight)
		if err != nil {
			cancelFunc()
			return err
		}
	}
	m.actionIDToParts[debugAction.GetIdentifier()] = &actionParts{
		actionStatus: &v2.ActionStatus{
			DebugAction:      debugAction,
			TimesEncountered: 0,
			TimesExecuted:    0,
			TimesSignaled:    0,
		},
		sema:       sema,
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
	return nil
}

func (m *managerImpl) ExecRegisteredAction(identifier string) {
	if buildinfo.ReleaseBuild {
		log.Error("Debug actions are not supported for release builds")
		return
	}
	parts := m.retrieveAndUpdateActionParts(identifier)
	if parts == nil {
		return
	}

	switch parts.actionStatus.GetDebugAction().GetAction().(type) {
	case *v2.DebugAction_SleepAction:
		execSleepAction(parts)
	case *v2.DebugAction_WaitAction:
		execWaitAction(parts)
	}
}

func execSleepAction(parts *actionParts) {
	done := parts.ctx.Done()
	select {
	case <-done:
		return
	default:
		sleepDuration := time.Duration(parts.actionStatus.GetDebugAction().GetSleepAction().GetSeconds())
		time.Sleep(sleepDuration * time.Second)
	}
}

func execWaitAction(parts *actionParts) {
	err := parts.sema.Acquire(parts.ctx, 1)
	if err != nil {
		log.Errorf("Error ")
		return
	}
}

func (m *managerImpl) retrieveAndUpdateActionParts(identifier string) *actionParts {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	parts, ok := m.actionIDToParts[identifier]
	if !ok {
		log.Debugf("No action registered for identifier '%s'", identifier)
		return nil
	}
	parts.actionStatus.TimesEncountered += 1
	if parts.actionStatus.GetTimesEncountered() > parts.actionStatus.GetDebugAction().GetNumTimes() {
		log.Debugf("Action registered for identifier '%s' already executed max number of times (%v), "+
			"skipping further executions", parts.actionStatus.GetDebugAction().GetNumTimes())
		return nil
	}
	parts.actionStatus.TimesExecuted += 1
	return parts
}

func (m *managerImpl) GetActionStatus(identifier string) (*v2.ActionStatus, error) {
	if buildinfo.ReleaseBuild {
		return nil, errors.New("Debug actions are not supported for release builds")
	}
	parts, ok := m.actionIDToParts[identifier]
	if !ok {
		return nil, errors.Wrapf(errox.NotFound, "No action registered for identifier '%s'", identifier)
	}
	return parts.actionStatus, nil
}

func (m *managerImpl) DeleteAction(identifier string) error {
	if buildinfo.ReleaseBuild {
		return errors.New("Debug actions are not supported for release builds")
	}
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	parts, ok := m.actionIDToParts[identifier]
	if !ok {
		return errors.Wrapf(errox.NotFound, "No action registered for identifier '%s'", identifier)
	}
	m.deleteActionNoLock(identifier, parts)
	return nil
}

func (m *managerImpl) deleteActionNoLock(identifier string, parts *actionParts) {
	// This should make all waiting routines proceed immediately
	parts.cancelFunc()
	delete(m.actionIDToParts, identifier)
}

func (m *managerImpl) ProceedOldest(identifier string) error {
	if buildinfo.ReleaseBuild {
		return errors.New("Debug actions are not supported for release builds")
	}
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	parts, ok := m.actionIDToParts[identifier]
	if !ok {
		return errors.Wrapf(errox.NotFound, "No action registered for identifier '%s'", identifier)
	}
	if parts.actionStatus.GetDebugAction().GetWaitAction() == nil {
		return errors.Wrapf(errox.InvalidArgs, "Proceed signals are only supported for wait actions")
	}

	if parts.actionStatus.GetTimesExecuted() <= parts.actionStatus.GetTimesSignaled() {
		return errors.Wrapf(errox.InvalidArgs, "Identifier '%s' has no routines waiting for a proceed signal", identifier)
	}
	// there are waiting routines. So we can release the semaphore
	parts.sema.Release(1)
	parts.actionStatus.TimesSignaled += 1
	return nil
}

func (m *managerImpl) ProceedAll(identifier string) error {
	if buildinfo.ReleaseBuild {
		return errors.New("Debug actions are not supported for release builds")
	}
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	parts, ok := m.actionIDToParts[identifier]
	if !ok {
		return errors.Wrapf(errox.NotFound, "No action registered for identifier '%s'", identifier)
	}
	if parts.actionStatus.GetDebugAction().GetWaitAction() == nil {
		return errors.Wrapf(errox.InvalidArgs, "Proceed signals are only supported for wait actions")
	}

	if parts.actionStatus.GetTimesExecuted() <= parts.actionStatus.GetTimesSignaled() {
		return errors.Wrapf(errox.InvalidArgs, "Identifier '%s' has no routines waiting for a proceed signal", identifier)
	}
	// there are waiting routines. So we can release the semaphore
	parts.sema.Release(parts.actionStatus.GetTimesExecuted() - parts.actionStatus.GetTimesSignaled())
	parts.actionStatus.TimesSignaled += 1
	return nil
}

func (m *managerImpl) Start() {
	if buildinfo.ReleaseBuild {
		log.Error("Debug actions are not supported for release builds")
		return
	}
}

func (m *managerImpl) Stop() {
	if buildinfo.ReleaseBuild {
		log.Error("Debug actions are not supported for release builds")
		return
	}
	m.mapLock.Lock()
	defer m.mapLock.Lock()
	for identifier, parts := range m.actionIDToParts {
		m.deleteActionNoLock(identifier, parts)
	}
}
