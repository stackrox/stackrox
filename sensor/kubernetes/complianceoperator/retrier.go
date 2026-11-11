package complianceoperator

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/retry"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
)

type retriableCall func(context.Context) error

type onRetriableCallError func(context.Context, error) error

type onConflictCall func(context.Context) error

func onConflictErrorWrapper(fn onConflictCall) onRetriableCallError {
	return func(ctx context.Context, parentErr error) error {
		if !kubeAPIErr.IsConflict(parentErr) && !retry.IsRetryable(parentErr) {
			return nil
		}
		if fn == nil {
			return errors.New("onConflictCall is 'nil'")
		}
		return fn(ctx)
	}
}

type retrier struct {
	maxRetries     int
	apiCallTimeout time.Duration
	retryTimeout   time.Duration
}

func (r *retrier) callWithRetry(ctx context.Context, fn retriableCall) error {
	retryCtx, cancel := context.WithTimeout(ctx, r.retryTimeout)
	defer cancel()
	return r.callWithRetryWithOnErrorCallback(retryCtx, fn, nil)
}

func (r *retrier) callWithRetryWithOnConflictCallback(ctx context.Context, fn retriableCall, onConflict onConflictCall) error {
	retryCtx, cancel := context.WithTimeout(ctx, r.retryTimeout)
	defer cancel()
	return r.callWithRetryWithOnErrorCallback(retryCtx, fn, onConflictErrorWrapper(onConflict))
}

func (r *retrier) callWithRetryWithOnErrorCallback(retryCtx context.Context, fn retriableCall, onErrFn onRetriableCallError) error {
	if fn == nil {
		return errors.New("retriableCall is 'nil'")
	}
	return retry.WithRetry(func() error {
		callCtx, callCancel := context.WithTimeout(retryCtx, r.apiCallTimeout)
		defer callCancel()
		err := fn(callCtx)
		if err != nil && onErrFn != nil {
			errCallCtx, errCallCancel := context.WithTimeout(retryCtx, r.apiCallTimeout)
			defer errCallCancel()
			if onErr := onErrFn(errCallCtx, err); onErr != nil {
				return errors.Wrapf(onErr, "callback error: %v", err)
			}
		}
		return err
	},
		retry.WithContext(retryCtx),
		retry.Tries(r.maxRetries),
		retry.WithExponentialBackoff())
}
