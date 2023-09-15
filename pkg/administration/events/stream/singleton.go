package stream

import (
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	stream events.Stream
)

// Singleton returns an instance of the administration events stream.
// Note that the Stream interface only holds events within memory and does not
// require any database related interfaces, nor does it initialize them.
//
// Currently, the Stream is separated from `central` to allow packages under `pkg` to rely on it, instead
// of requiring a dependency towards packages within `central`.
func Singleton() events.Stream {
	once.Do(func() {
		stream = newStream()
	})
	return stream
}
