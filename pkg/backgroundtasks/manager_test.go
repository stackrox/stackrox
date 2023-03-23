package backgroundtasks

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
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

func add(nums ...int) (res int) {
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
	assert.NoError(t, err)

	// Should add to pending queue.
	_, err = m.AddTask(nil, simpleTask())
	assert.NoError(t, err)

	// Should not be added to pending queue.
	_, err = m.AddTask(nil, simpleTask())
	assert.ErrorContains(t, err, "Cannot add task: queue full.")
}

func TestTaskExpirationCleanup(t *testing.T) {
	t.Skip("skipping due to race condition risk")

	m := NewManager(WithCleanUpInterval(1*time.Millisecond), WithExpirationCompletedTasks(10*time.Millisecond))
	m.Start()

	id, err := m.AddTask(nil, simpleTask(3*time.Millisecond))
	assert.NoError(t, err)
	metadata, res, completed, err := m.GetTaskStatusAndMetadata(id)
	t.Log("[CHECK] Job should not have completed.")
	assert.NoError(t, err)
	assert.False(t, completed)
	assert.Nil(t, res)
	assert.Empty(t, metadata)
	t.Log("Passed..")

	// Let job to complete.
	t.Log("Allowing job to complete...")
	time.Sleep(5 * time.Millisecond)

	metadata, res, completed, err = m.GetTaskStatusAndMetadata(id)
	t.Log("[CHECK] Job should have completed by now.")
	assert.True(t, completed)
	assert.NoError(t, err)
	assert.Nil(t, res)
	assert.Empty(t, metadata)
	t.Log("Passed..")

	// Let job to expire.
	t.Log("Allowing job to expire and be cleaned up...")
	time.Sleep(12 * time.Millisecond)

	// Let completed job to have expired and cleaned up.
	metadata, res, completed, err = m.GetTaskStatusAndMetadata(id)
	t.Log("[CHECK] Job should have been cleaned up by now.")
	assert.False(t, completed)
	assert.ErrorContains(t, err, "id does not exist.")
	assert.Nil(t, res)
	assert.Empty(t, metadata)
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
	assert.NoError(t, err)

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
	testRes := add(testArgs...)
	meta := make(map[string]interface{})
	testKey := "K"
	testVal := "V"
	meta[testKey] = testVal
	c := func(ctx concurrency.ErrorWaitable) (interface{}, error) {
		time.Sleep(4 * time.Millisecond)
		return add(testArgs...), nil
	}

	id, err = m.AddTask(meta, c)
	assert.NoError(t, err)
	for {
		metadata, res, completed, err := m.GetTaskStatusAndMetadata(id)
		assert.NoError(t, err)
		assert.Len(t, metadata, 1)
		assert.Equal(t, testVal, metadata[testKey])
		if completed {
			assert.Equal(t, testRes, res.(int))
			break
		}

		assert.Nil(t, res)
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
	assert.NoError(t, err)
	assert.False(t, completed)
	assert.Nil(t, res)

	err = m.CancelTask(id)
	assert.NoError(t, err)
	time.Sleep(3 * time.Millisecond)
	// Task should be cancelled, with nil results.
	for {
		_, res, completed, err = m.GetTaskStatusAndMetadata(id)
		if completed {
			assert.Nil(t, res)
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
		assert.NoError(t, err)
		if completed {
			assert.Equal(t, add(testArgs...), res)
			break
		}

		time.Sleep(3 * time.Millisecond)
	}

	err = m.CancelTask(id)
	assert.NoError(t, err)
	time.Sleep(3 * time.Millisecond)
	// Cancellation shouldn't affect anything.
	_, res, completed, err = m.GetTaskStatusAndMetadata(id)
	assert.True(t, completed)
	assert.Equal(t, add(testArgs...), res)
	assert.NoError(t, err)
	t.Log("Passed.")

	t.Log("[CHECK] Validate cancellation of task before task has started running.")
	testArgs = []int{1, 2, 3, 4, 5}

	_, err = m.AddTask(nil, simpleTask())
	assert.NoError(t, err)
	id, err = m.AddTask(nil, addWithContext(testArgs...))
	assert.NoError(t, err)
	// Check task hasnt completed yet.
	_, res, completed, err = m.GetTaskStatusAndMetadata(id)
	assert.NoError(t, err)
	assert.False(t, completed)
	assert.Nil(t, res)

	err = m.CancelTask(id)
	assert.NoError(t, err)
	time.Sleep(3 * time.Millisecond)
	// Task should be cancelled, without results.
	for {
		_, res, completed, err = m.GetTaskStatusAndMetadata(id)
		if completed {
			assert.Nil(t, res)
			assert.ErrorContains(t, err, "context canceled")
			break
		}
	}
	t.Log("Passed.")
}
