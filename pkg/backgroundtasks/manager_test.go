package backgroundtasks

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"gotest.tools/assert"
)

//lint:file-ignore U1000 Unused functions are due to test skip.

func panicTask(args ...interface{}) Task {
	return func(ctx concurrency.ErrorWaitable) (interface{}, error) {
		panic(args)
	}
}

func simpleTask(ds ...time.Duration) Task {
	return func(ctx concurrency.ErrorWaitable) (interface{}, error) {
		if len(ds) == 0 {
			time.Sleep(10 * time.Millisecond)
		}

		for _, d := range ds {
			time.Sleep(d)
		}

		return nil, nil
	}
}

func addWithContext(nums ...int) Task {
	return func(ctx concurrency.ErrorWaitable) (interface{}, error) {
		ch := make(chan int, 1)
		idx := 0
		sum := 0
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()

			case n, ok := <-ch:
				if !ok {
					return sum, nil
				}

				sum += n
			case <-time.After(2 * time.Millisecond):
				if idx == len(nums) {
					close(ch)
				} else {
					ch <- nums[idx]
					idx++
				}
			}
		}
	}
}

func addFunc(nums ...int) (res int) {
	for _, n := range nums {
		res += n
	}

	return
}

func TestPendingTaskQueueSize(t *testing.T) {
	t.Skip("skipping due to race condition risk")

	m := NewManager(WithMaxPendingTaskQueueSize(1), WithMaxTasksInParallel(1))
	m.Start()

	// Should run immediately.
	_, err := m.AddTask(nil, simpleTask())
	assert.NilError(t, err)

	// Should add to pending queue.
	_, err = m.AddTask(nil, simpleTask())
	assert.NilError(t, err)

	// Should not be added to pending queue.
	_, err = m.AddTask(nil, simpleTask())
	assert.ErrorContains(t, err, "Cannot add task: queue full.")
}

func TestTaskExpirationCleanup(t *testing.T) {
	t.Skip("skipping due to race condition risk")

	m := NewManager(WithCleanUpInterval(1*time.Millisecond), WithExpirationCompletedTasks(10*time.Millisecond))
	m.Start()

	id, err := m.AddTask(nil, simpleTask(3*time.Millisecond))
	assert.NilError(t, err)
	metadata, res, completed, err := m.GetTaskStatusAndMetadata(id)
	t.Log("[CHECK] Job should not have completed.")
	assert.NilError(t, err)
	assert.Equal(t, completed, false)
	assert.Equal(t, res, nil)
	assert.Equal(t, len(metadata), 0)
	t.Log("Passed..")

	// Let job to complete.
	t.Log("Allowing job to complete...")
	time.Sleep(5 * time.Millisecond)

	metadata, res, completed, err = m.GetTaskStatusAndMetadata(id)
	t.Log("[CHECK] Job should have completed by now.")
	assert.Equal(t, completed, true)
	assert.NilError(t, err)
	assert.Equal(t, res, nil)
	assert.Equal(t, len(metadata), 0)
	t.Log("Passed..")

	// Let job to expire.
	t.Log("Allowing job to expire and be cleaned up...")
	time.Sleep(12 * time.Millisecond)

	// Let completed job to have expired and cleaned up.
	metadata, res, completed, err = m.GetTaskStatusAndMetadata(id)
	t.Log("[CHECK] Job should have been cleaned up by now.")
	assert.Equal(t, completed, false)
	assert.ErrorContains(t, err, "id does not exist.")
	assert.Equal(t, res, nil)
	assert.Equal(t, len(metadata), 0)
	t.Log("Passed..")
}

func TestBackgroundTasksManager(t *testing.T) {
	t.Skip("skipping due to race condition risk")

	m := NewManager(WithCleanUpInterval(1 * time.Millisecond))
	m.Start()

	// Task panics.
	t.Log("[CHECK] Validating task panic...")
	testArg := "test"
	id, err := m.AddTask(nil, panicTask(testArg))
	assert.NilError(t, err)

	for {
		_, _, completed, err := m.GetTaskStatusAndMetadata(id)
		if completed {
			assert.ErrorContains(t, err, testArg)
			break
		}
	}
	t.Log("Passed.")

	// Valid input.
	t.Log("[CHECK] Validating valid inputs...")
	testArgs := []int{1, 2, 3}
	testRes := addFunc(testArgs...)
	meta := make(map[string]interface{})
	testKey := "K"
	testVal := "V"
	meta[testKey] = testVal
	c := func(ctx concurrency.ErrorWaitable) (interface{}, error) {
		time.Sleep(4 * time.Millisecond)
		return addFunc(testArgs...), nil
	}

	id, err = m.AddTask(meta, c)
	assert.NilError(t, err)
	for {
		metadata, res, completed, err := m.GetTaskStatusAndMetadata(id)
		assert.NilError(t, err)
		assert.Equal(t, len(metadata), 1)
		assert.Equal(t, metadata[testKey], testVal)
		if completed {
			assert.Equal(t, res.(int), testRes)
			break
		} else {
			assert.Equal(t, res, nil)
		}
	}
	t.Log("Passed.")
}

func TestTaskCancellation(t *testing.T) {
	t.Skip("skipping due to race condition risk")

	m := NewManager(WithCleanUpInterval(1*time.Millisecond), WithMaxTasksInParallel(1))
	m.Start()

	t.Log("[CHECK] Validate cancellation of task while task is running.")
	testArgs := []int{1, 2, 3, 4, 5}
	id, _ := m.AddTask(nil, addWithContext(testArgs...))
	time.Sleep(2 * time.Millisecond)
	// Check task hasnt completed yet.
	_, res, completed, err := m.GetTaskStatusAndMetadata(id)
	assert.NilError(t, err)
	assert.Equal(t, completed, false)
	assert.Equal(t, res, nil)

	err = m.CancelTask(id)
	assert.NilError(t, err)
	time.Sleep(3 * time.Millisecond)
	// Task should be cancelled, with nil results.
	for {
		_, res, completed, err = m.GetTaskStatusAndMetadata(id)
		if completed {
			assert.Equal(t, res, nil)
			assert.ErrorContains(t, err, "context canceled")
			break
		}
	}
	t.Log("Passed.")

	t.Log("[CHECK] Validate cancellation of task after task has already completed fetches results.")
	id, _ = m.AddTask(nil, addWithContext(testArgs...))
	// Check for completion of task.
	for {
		_, res, completed, err = m.GetTaskStatusAndMetadata(id)
		assert.NilError(t, err)
		if completed {
			assert.Equal(t, res, addFunc(testArgs...))
			break
		}

		time.Sleep(3 * time.Millisecond)
	}

	err = m.CancelTask(id)
	assert.NilError(t, err)
	time.Sleep(3 * time.Millisecond)
	// Cancellation shouldnt affect anything.
	_, res, completed, err = m.GetTaskStatusAndMetadata(id)
	assert.Equal(t, completed, true)
	assert.Equal(t, res, addFunc(testArgs...))
	assert.NilError(t, err)
	t.Log("Passed.")

	t.Log("[CHECK] Validate cancellation of task before task has started running.")
	testArgs = []int{1, 2, 3, 4, 5}

	_, err = m.AddTask(nil, simpleTask())
	assert.NilError(t, err)
	id, err = m.AddTask(nil, addWithContext(testArgs...))
	assert.NilError(t, err)
	// Check task hasnt completed yet.
	_, res, completed, err = m.GetTaskStatusAndMetadata(id)
	assert.NilError(t, err)
	assert.Equal(t, completed, false)
	assert.Equal(t, res, nil)

	err = m.CancelTask(id)
	assert.NilError(t, err)
	time.Sleep(3 * time.Millisecond)
	// Task should be cancelled, without results.
	for {
		_, res, completed, err = m.GetTaskStatusAndMetadata(id)
		if completed {
			assert.Equal(t, res, nil)
			assert.ErrorContains(t, err, "context canceled")
			break
		}
	}
	t.Log("Passed.")
}
