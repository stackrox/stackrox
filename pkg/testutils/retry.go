package testutils

import "time"

// T generalizes testing.T
type T interface {
	Errorf(format string, args ...interface{})
	FailNow()
}

type failure struct{}

type retryT struct{}

func (retryT) Errorf(string, ...interface{}) {
	panic(failure{})
}

func (retryT) FailNow() {
	panic(failure{})
}

func runRetry(testFn func(t T)) (success bool) {
	defer func() {
		if success {
			return
		}

		r := recover()
		if _, ok := r.(failure); !ok {
			panic(r)
		}
	}()

	testFn(retryT{})
	success = true

	return
}

// Retry retries a test function up to the given number of times.
func Retry(t T, times int, sleepInterval time.Duration, testFn func(t T)) {
	for i := 0; i < times-1; i++ {
		if runRetry(testFn) {
			return
		}
		time.Sleep(sleepInterval)
	}
	testFn(t)
}
