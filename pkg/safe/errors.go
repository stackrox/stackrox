package safe

import "github.com/pkg/errors"

var (
	// ErrWaitableTriggered is returned when a waitable signal is triggered before or during a channel write operation.
	ErrWaitableTriggered = errors.New("waitable was triggered")

	// ErrChannelFull is returned when attempting a non-blocking write to a full channel.
	ErrChannelFull = errors.New("channel is full")
)
