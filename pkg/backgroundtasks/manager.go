package backgroundtasks

import (
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// Task is the functional type through which this manager accepts tasks.
type Task func(ctx concurrency.ErrorWaitable) (interface{}, error)

// Manager takes care of accepting and running background processes.
type Manager interface {
	// AddTask adds a task to run as a background process and returns a task id. It also stores any metadata related to
	// the task.
	AddTask(metadata map[string]interface{}, task Task) (string, error)

	// GetTaskStatusAndMetadata returns the status of the task id, along with results and associated metadata.
	GetTaskStatusAndMetadata(taskID string) (metadata map[string]interface{}, result interface{}, completed bool, err error)

	// CancelTask cancels the execution for the particular taskID. It is the responsibility of the task to clean up any
	// intermediate results/resource acquisitions that occur while it is executing and is asked to stop.
	CancelTask(taskID string) error

	// Start initiates the manager to accept tasks and monitor the execution.
	Start()
}

// ManagerOption returns configurable options for the manager.
type ManagerOption func(impl *managerImpl)

// NewManager returns an implementation of a background task manager.
func NewManager(opts ...ManagerOption) Manager {
	m := &managerImpl{}
	m.applyDefaults()
	applyOptions(m, opts)
	return m
}

func applyOptions(manager *managerImpl, opts []ManagerOption) {
	for _, opt := range opts {
		opt(manager)
	}
}
