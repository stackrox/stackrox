package kocache

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

var errCentralUnreachable = errors.New("central is currently unreachable")

// offlineCtrl exists to ensure consistency between variables being related to offline state
type offlineCtrl struct {
	parentCtx context.Context
	// onlineCtx will be canceled when connectivity to central is lost
	onlineCtx       context.Context
	onlineCtxCancel context.CancelCauseFunc
	ctxMutex        *sync.Mutex
}

func newOfflineCtrl(parentCtx context.Context, startOnline bool) *offlineCtrl {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	onlineCtx, onlineCtxCancel := context.WithCancelCause(parentCtx)
	oc := &offlineCtrl{
		parentCtx:       parentCtx,
		onlineCtx:       onlineCtx,
		onlineCtxCancel: onlineCtxCancel,
		ctxMutex:        &sync.Mutex{},
	}
	if !startOnline {
		oc.GoOffline()
	}
	return oc
}

func (o *offlineCtrl) IsOnline() bool {
	return o.onlineCtx.Err() == nil
}

func (o *offlineCtrl) Context() context.Context {
	o.ctxMutex.Lock()
	defer o.ctxMutex.Unlock()
	return o.onlineCtx
}

func (o *offlineCtrl) GoOnline() {
	o.ctxMutex.Lock()
	defer o.ctxMutex.Unlock()
	o.onlineCtx, o.onlineCtxCancel = context.WithCancelCause(o.parentCtx)
}

func (o *offlineCtrl) GoOffline() {
	o.onlineCtxCancel(errCentralUnreachable)
}
