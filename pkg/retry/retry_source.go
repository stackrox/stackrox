package retry

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// RetryableSource is a proxy with an object that is able to compute a result, but
// that might forget our request, or return an error result, and that returns the
// result asynchronously.
// AskForResult() can be called to request a result, that should be make available in the
// returned channel. Each time AskForResult() is called the previously returned channel is abandoned.
// Retry() can be called several times to retry the result computation, the
// RetryableSource is in charge of handling the cancellation of the computation if needed.
type RetryableSource interface {
	AskForResult(ctx context.Context) chan *Result
	Retry()
}

// Result wraps a pair (result, err) produced by a source. By convention
// either err or v has the zero value of its type.
type Result struct {
	v   interface{}
	err error
}

// RetryableSourceRetriever be used to retrieve the result in a RetryableSource.
type RetryableSourceRetriever struct {
	// time to consider failed a call to AskForResult() that didn't return a result yet.
	RequestTimeout time.Duration
	// optionally specify a function to invoke on each error. waitDuration is the time until
	// the next retry.
	OnError func(err error, timeToNextRetry time.Duration)
	// optionally specify a validation function for each result.
	ValidateResult func(interface{}) bool
	// should be reset between calls to Run.
	Backoff      wait.Backoff
	timeoutC     chan struct{}
	timeoutTimer *time.Timer
}

// NewRetryableSourceRetriever create a new NewRetryableSourceRetriever
func NewRetryableSourceRetriever(backoff wait.Backoff, requestTimeout time.Duration) *RetryableSourceRetriever {
	return &RetryableSourceRetriever{
		RequestTimeout: requestTimeout,
		Backoff:        backoff,
	}
}

// Run gets the result from the specified source.
// Any timeout in ctx is respected.
func (r *RetryableSourceRetriever) Run(ctx context.Context, source RetryableSource) (interface{}, error) {
	r.timeoutC = make(chan struct{})

	resultC := source.AskForResult(ctx)
	r.setTimeoutTimer(r.RequestTimeout)
	defer r.setTimeoutTimer(-1)
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("request cancelled")
		case <-r.timeoutC:
			// assume result will never come.
			r.handleError(errors.New("timeout"), source)
		case result := <-resultC:
			err := result.err
			if err != nil {
				r.handleError(err, source)
			} else {
				if r.ValidateResult != nil && !r.ValidateResult(result.v) {
					err := errors.Errorf("validation failed for value %v", result.v)
					r.handleError(err, source)
				} else {
					return result.v, nil
				}
			}
		}
	}
}

func (r *RetryableSourceRetriever) handleError(err error, source RetryableSource) {
	waitDuration := r.Backoff.Step()
	if r.OnError != nil {
		r.OnError(err, waitDuration)
	}
	r.setTimeoutTimer(-1)
	time.AfterFunc(waitDuration, func() {
		source.Retry()
		r.setTimeoutTimer(r.RequestTimeout)
	})
}

// use negative timeout to just stop the timer.
func (r *RetryableSourceRetriever) setTimeoutTimer(timeout time.Duration) {
	if r.timeoutTimer != nil {
		r.timeoutTimer.Stop()
	}
	if timeout >= 0 {
		r.timeoutTimer = time.AfterFunc(timeout, func() {
			r.timeoutC <- struct{}{}
		})
	} else {
		r.timeoutTimer = nil
	}
}
