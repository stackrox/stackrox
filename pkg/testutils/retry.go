package testutils

import (
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// T generalizes testing.T
type T interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	FailNow()
}

type failure struct{}

type retryT struct{}

func (retryT) Errorf(string, ...interface{}) {
	panic(failure{})
}

func (retryT) Fatalf(string, ...interface{}) {
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
		log.Infof("Retry defer handler got: %v", r)
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
		log.Infof("Test attempt: %d", i)
		if runRetry(testFn) {
			return
		}
		time.Sleep(sleepInterval)
	}
	log.Info("Final test attempt")
	testFn(t)
}
