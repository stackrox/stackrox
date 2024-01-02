package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var cancelExecutionMarker = func() {}

type taskWrapper struct {
	// Task id.
	id string

	// Function to execute.
	exec Task

	// Metadata related to task.
	metadata map[string]interface{}

	// Results from execution of task.
	result interface{}

	// Errors from execution of task.
	err error

	// Cancels the execution of task.
	cancel context.CancelFunc
}

func (t *taskWrapper) execute(ctx context.Context) (res interface{}, err error) {
	if err = ctx.Err(); err != nil {
		return
	}

	panicked := true
	defer func() {
		if p := recover(); p != nil || panicked {
			err = fmt.Errorf("Panic during execution of task: %v", p)
		}
	}()

	res, err = t.exec(ctx)
	panicked = false
	return
}

type managerImpl struct {
	existingTasks  map[string]*taskWrapper
	completedTasks map[string]time.Time
	pendingTasks   chan *taskWrapper

	existingTasksMutex   sync.Mutex
	completedTasksMutex  sync.Mutex
	cancelFuncTasksMutex *concurrency.KeyedMutex
	parallel             chan struct{}

	completedTaskExpiryTime time.Duration
	cleanUpInterval         time.Duration
}

// WithExpirationCompletedTasks sets the duration after which completed tasks are cleaned up.
func WithExpirationCompletedTasks(t time.Duration) ManagerOption {
	return func(m *managerImpl) {
		m.completedTaskExpiryTime = t
	}
}

// WithMaxTasksInParallel sets the max tasks that are allowed to run in parallel.
func WithMaxTasksInParallel(p int) ManagerOption {
	return func(m *managerImpl) {
		m.parallel = make(chan struct{}, p)
		m.cancelFuncTasksMutex = concurrency.NewKeyedMutex(uint32(cap(m.parallel) + cap(m.pendingTasks)))
	}
}

// WithMaxPendingTaskQueueSize sets the max pending job queue size.
func WithMaxPendingTaskQueueSize(sz int) ManagerOption {
	return func(m *managerImpl) {
		m.pendingTasks = make(chan *taskWrapper, sz)
		m.cancelFuncTasksMutex = concurrency.NewKeyedMutex(uint32(cap(m.parallel) + cap(m.pendingTasks)))
	}
}

// WithCleanUpInterval sets the duration after which the manager wakes up to perform it's clean up jobs.
func WithCleanUpInterval(t time.Duration) ManagerOption {
	return func(m *managerImpl) {
		m.cleanUpInterval = t
	}
}

func (m *managerImpl) applyDefaults() {
	m.parallel = make(chan struct{}, 8)
	m.completedTaskExpiryTime = 20 * time.Minute
	m.pendingTasks = make(chan *taskWrapper, 256)
	m.cleanUpInterval = 5 * time.Minute
	m.existingTasks = make(map[string]*taskWrapper)
	m.completedTasks = make(map[string]time.Time)
	m.cancelFuncTasksMutex = concurrency.NewKeyedMutex(uint32(cap(m.parallel) + cap(m.pendingTasks)))
}

// AddTask adds a task to its pending task list to be run as a background process and returns a jobid.
func (m *managerImpl) AddTask(metadata map[string]interface{}, task Task) (string, error) {
	t := &taskWrapper{
		exec:     task,
		metadata: metadata,
	}

	id := uuid.NewV4().String()
	t.id = id
	select {
	case m.pendingTasks <- t:
	default:
		return "", errors.New("Cannot add task: queue full.")
	}

	m.existingTasksMutex.Lock()
	defer m.existingTasksMutex.Unlock()
	m.existingTasks[id] = t
	return id, nil
}

// Start initiates the manager to accept tasks and monitor the execution.
func (m *managerImpl) Start() {
	parentCtx := context.Background()
	go func() {
		for {
			m.parallel <- struct{}{}
			t := <-m.pendingTasks
			ctx, cancel := context.WithCancel(parentCtx)

			m.cancelFuncTasksMutex.DoWithLock(t.id, func() {
				if t.cancel != nil {
					cancel()
				}
				t.cancel = cancel
			})

			m.startTaskExec(ctx, t)
		}
	}()

	go func() {
		for {
			select {
			case <-parentCtx.Done():
				return

			case <-time.After(m.cleanUpInterval):
				m.cleanupExpiredCompletedTasks()
			}
		}
	}()
}

func (m *managerImpl) startTaskExec(ctx context.Context, t *taskWrapper) {
	go func(t *taskWrapper) {
		res, err := t.execute(ctx)
		<-m.parallel

		m.completedTasksMutex.Lock()
		defer m.completedTasksMutex.Unlock()
		m.completedTasks[t.id] = time.Now()
		t.result = res
		t.err = err
	}(t)
}

func (m *managerImpl) cleanupExpiredCompletedTasks() {
	m.completedTasksMutex.Lock()
	defer m.completedTasksMutex.Unlock()

	// check expired completed tasks.
	currentTime := time.Now()
	var removeIds []string

	for id, t := range m.completedTasks {
		if t.Add(m.completedTaskExpiryTime).Before(currentTime) {
			removeIds = append(removeIds, id)
		}
	}

	m.existingTasksMutex.Lock()
	defer m.existingTasksMutex.Unlock()
	for _, id := range removeIds {
		delete(m.completedTasks, id)
		delete(m.existingTasks, id)
	}
}

// GetTaskStatus returns the status of the jobid.
func (m *managerImpl) GetTaskStatusAndMetadata(taskID string) (map[string]interface{}, interface{}, bool, error) {
	m.completedTasksMutex.Lock()
	defer m.completedTasksMutex.Unlock()
	m.existingTasksMutex.Lock()
	defer m.existingTasksMutex.Unlock()

	t, ok := m.existingTasks[taskID]
	if !ok {
		return nil, nil, false, errors.New("id does not exist.")
	}

	_, completed := m.completedTasks[taskID]
	return t.metadata, t.result, completed, t.err
}

func (m *managerImpl) CancelTask(taskID string) error {
	m.existingTasksMutex.Lock()
	defer m.existingTasksMutex.Unlock()
	t, ok := m.existingTasks[taskID]
	if !ok {
		return errors.New("id does not exist.")
	}

	// We defer unlock of existingTasksMutex so that it cant be cleaned up, and we know that it exists throughout the
	// completion of this fn.
	m.cancelFuncTasksMutex.Lock(taskID)
	defer m.cancelFuncTasksMutex.Unlock(taskID)
	if t.cancel == nil {
		t.cancel = cancelExecutionMarker
		return nil
	}

	t.cancel()
	return nil
}
