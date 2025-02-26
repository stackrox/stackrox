package manager

import v2 "github.com/stackrox/rox/generated/api/v2"

// Manager manages the registered actions. It is responsible for registering new actions, executing the registered actions,
// deleting actions and sending proceed signals to routines waiting due to wait actions
//
//go:generate mockgen-wrapper
type Manager interface {
	// RegisterAction registers new action
	RegisterAction(action *v2.DebugAction) error
	// GetActionStatus returns the current status of the action registered for given identifier
	GetActionStatus(identifier string) (*v2.ActionStatus, error)
	// DeleteAction deletes the action registered for the given identifier.
	// If any routines are waiting due to this action, they are all signalled to proceed
	DeleteAction(identifier string) error
	// ProceedOldest sends proceed signal the oldest routine waiting on the given action identifier.
	// This is only relevant when registered action is of WaitAction type
	ProceedOldest(identifier string) error
	// ProceedAll Proceeds all routines waiting on the given action identifier.
	// This is only relevant when registered action is of WaitAction type
	ProceedAll(identifier string) error

	// Start action manager
	Start()
	// Stop action manager. All registered actions are deleted and all waiting go routines are signalled to proceed.
	Stop()
}

// New returns a new instance of the Manager
func New() Manager {
	return &managerImpl{}
}
