package enhancement

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
)

// Watcher .
type Watcher struct {
	signals map[string]chan *central.DeploymentEnhancementResponse
	lock    sync.Mutex
}

// NewWatcher .
func NewWatcher() *Watcher {
	return &Watcher{}
}

// WaitForEnhancedDeployment .
func (w *Watcher) WaitForEnhancedDeployment(_, messageID string, timeout time.Duration) (*central.DeploymentEnhancementResponse, error) {
	ch := make(chan *central.DeploymentEnhancementResponse, 1)
	w.lock.Lock()
	w.signals[messageID] = ch
	w.lock.Unlock()

	defer func() {
		w.lock.Lock()
		defer w.lock.Unlock()
		delete(w.signals, messageID)
	}()

	select {
	case m, more := <-ch:
		if !more {
			return nil, errors.New("wait channel closed prematurely")
		}
		return m, nil
	case <-time.After(timeout):
		return nil, errors.New("timed out waiting for deployment enhancement")
	}
}

// NotifyEnhancementReceived .
func (w *Watcher) NotifyEnhancementReceived(_ string, msg *central.DeploymentEnhancementResponse) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if s, ok := w.signals[msg.GetId()]; ok {
		select {
		case s <- msg:
			break
		default:
			// In case there is a bug in sensor that makes multiple messages be sent for the same ID this can
			// cause a deadlock. So if writing is blocking we discard the message to avoid deadlocking central.
		}
	}
}
