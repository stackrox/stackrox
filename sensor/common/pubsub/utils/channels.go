package utils

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
)

// SafeBlockingWriteToChannel pushes an item to a channel making sure there are
// no races between the Waitable and the writing to the channel.
func SafeBlockingWriteToChannel[T any](mu *sync.Mutex, waitable concurrency.Waitable, ch chan<- T, item T) error {
	// We need two selects to make sure we do not race
	// The function that triggers and closes the channel should have the
	// following structure:
	//   - trigger the waitable
	//   - wait for the waitable to report is done
	//   - close the channel while holding the mutex
	mu.Lock()
	defer mu.Unlock()
	// First select will exit early if waitable is already triggered
	select {
	case <-waitable.Done():
		return pubsubErrors.WaitableTriggeredErr
	default:
	}
	// Second select will exit if we are blocked waiting to write in the channel
	select {
	case <-waitable.Done():
		return pubsubErrors.WaitableTriggeredErr
	case ch <- item:
		return nil
	}
}

// SafeWriteToChannel pushes an item to a channel making sure there are
// no races between the Waitable and the writing to the channel.
// This will drop the item if the channel is full.
func SafeWriteToChannel[T any](mu *sync.Mutex, waitable concurrency.Waitable, ch chan<- T, item T) error {
	// We need two selects to make sure we do not race
	// The function that triggers and closes the channel should have the
	// following structure:
	//   - trigger the waitable
	//   - wait for the waitable to report is done
	//   - close the channel while holding the mutex
	mu.Lock()
	defer mu.Unlock()
	// First select will exit early if waitable is already triggered
	select {
	case <-waitable.Done():
		return pubsubErrors.WaitableTriggeredErr
	default:
	}
	// Second select will exit if we are blocked waiting to write in the channel
	select {
	case <-waitable.Done():
		return pubsubErrors.WaitableTriggeredErr
	case ch <- item:
		return nil
	default:
		return pubsubErrors.ChannelFullErr
	}
}
