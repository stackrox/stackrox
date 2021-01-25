package concurrency

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobProcessorGracefulStop(t *testing.T) {
	sentJobs, completedJobs := set.NewIntSet(), set.NewIntSet()
	var sentJobsLock, completedJobsLock sync.Mutex

	const numWorkers = 5
	processor := NewJobProcessor(numWorkers)

	const numVals = 200

	quarterOfTheJobsSent := NewSignal()

	go func() {
		for i := 0; i < numVals; i++ {
			func(val int) {
				err := processor.AddJob(val, nil, func() {
					WithLock(&completedJobsLock, func() {
						completedJobs.Add(val)
					})
				})
				if err != nil {
					require.Equal(t, ErrJobProcessorStopped, err)
				} else {
					WithLock(&sentJobsLock, func() {
						sentJobs.Add(val)
						if len(sentJobs) > numVals/4 {
							quarterOfTheJobsSent.Signal()
						}
					})
				}
			}(i)
		}
	}()

	// Somewhere in between the other job submissions, do a graceful stop.
	assert.True(t, WaitWithTimeout(&quarterOfTheJobsSent, time.Second))
	processor.GracefulStop()
	assert.True(t, PollWithTimeout(processor.Stopped, 10*time.Millisecond, 2*time.Second))
	assert.ElementsMatch(t, sentJobs.AsSlice(), completedJobs.AsSlice())
}

func TestJobProcessorPerfectlyParallel(t *testing.T) {
	const numWorkers = 10
	processor := NewJobProcessor(numWorkers)
	defer processor.Stop()

	const sleepTime = 10 * time.Millisecond

	outChan := make(chan int)

	const numVals = 200

	start := time.Now()
	for i := 0; i < numVals; i++ {
		val := i
		require.NoError(t, processor.AddJob(val, nil, func() {
			time.Sleep(sleepTime)
			outChan <- val
		}))
	}

	receivedVals := set.NewIntSet()
	for i := 0; i < numVals; i++ {
		val := <-outChan
		assert.True(t, receivedVals.Add(val), "value %d duplicated", val)
	}

	totalTimeTaken := time.Since(start)
	fastestPossible := numVals * sleepTime / numWorkers
	slowestAllowed := 3 * fastestPossible
	// Ensure that there is at least _some_ parallelization happening, and that the implementation is not doing it
	// all in sequence. In practice, it will be a lot faster than this, but we want to ensure there are no unit test flakes.
	assert.True(t, fastestPossible < totalTimeTaken && totalTimeTaken < slowestAllowed, "Expected it to take between "+
		"%v and %v but took %v", fastestPossible, slowestAllowed, totalTimeTaken)
}

func TestJobProcessorComplexCase(t *testing.T) {
	const numWorkers = 10
	processor := NewJobProcessor(numWorkers)
	defer processor.Stop()

	const sleepTime = 20 * time.Millisecond

	outChan := make(chan int)

	const numVals = 200

	start := time.Now()
	for i := 0; i < numVals; i++ {
		val := i
		require.NoError(t, processor.AddJob(val, func(otherJobMetadata interface{}) bool {
			// All jobs where the id is a multiple of 10 are not allowed
			// to execute until all the jobs preceding them have been.
			if val%10 == 0 {
				return otherJobMetadata.(int) < val
			}
			// All jobs where the id is a multiple of 6 are not allowed to
			// execute until the job with the id = half this job's id has executed.
			if val%6 == 0 {
				return otherJobMetadata.(int) == val/2
			}
			return false
		}, func() {
			time.Sleep(sleepTime)
			outChan <- val
		}))
	}

	receivedVals := make([]int, 0, numVals)
	for i := 0; i < numVals; i++ {
		receivedVals = append(receivedVals, <-outChan)
	}

	totalTimeTaken := time.Since(start)
	seenSoFar := set.NewIntSet()
	for _, val := range receivedVals {
		assert.True(t, seenSoFar.Add(val), "val %d seen twice", val)
		if val%10 == 0 {
			for i := 0; i < val; i++ {
				assert.Contains(t, seenSoFar, i, "should contain %d since %d is a multiple of 10", i, val)
			}
		}
		if val%6 == 0 {
			assert.Contains(t, seenSoFar, val/2, "should contain %d since %d is a multiple of 6", val/2, val)
		}
	}
	// Ensure that there is at least _some_ parallelization happening, and that the implementation is not doing it
	// all in sequence. In practice, it will be a lot faster than this, but we want to ensure there are no unit test flakes.
	assert.True(t, totalTimeTaken < (numVals*sleepTime)/2, "Expected it to take "+
		"less than %v but took %v", (numVals*sleepTime)/2, totalTimeTaken)
}

func TestJobProcessorEverythingConflicts(t *testing.T) {
	const numWorkers = 10
	processor := NewJobProcessor(numWorkers)
	defer processor.Stop()

	const sleepTime = 20 * time.Millisecond

	outChan := make(chan int)

	const numVals = 10

	for i := 0; i < numVals; i++ {
		val := i
		require.NoError(t, processor.AddJob(val, func(_ interface{}) bool {
			return true
		}, func() {
			time.Sleep(sleepTime)
			outChan <- val
		}))
	}

	// Since everything conflicts, the output should be in order.
	for i := 0; i < numVals; i++ {
		val := <-outChan
		assert.Equal(t, i, val)
	}
}
