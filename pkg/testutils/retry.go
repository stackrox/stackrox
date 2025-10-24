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
	Logf(format string, args ...interface{})
}

type failure struct{}

type retryT struct {
	t T
}

func (retryT) Errorf(string, ...interface{}) {
	panic(failure{})
}

func (retryT) Fatalf(string, ...interface{}) {
	panic(failure{})
}

func (retryT) FailNow() {
	panic(failure{})
}

func (r retryT) Logf(format string, args ...interface{}) {
	r.t.Logf(format, args...)
}

func runRetry(t T, testFn func(t T)) (success bool) {
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

	testFn(retryT{t: t})
	success = true

	return
}

// Retry retries a test function up to the given number of times.
func Retry(t T, times int, sleepInterval time.Duration, testFn func(t T)) {
	for i := 0; i < times-1; i++ {
		log.Infof("Test attempt: %d", i)
		if runRetry(t, testFn) {
			return
		}
		time.Sleep(sleepInterval)
	}
	log.Info("Final test attempt")
	testFn(t)
}
