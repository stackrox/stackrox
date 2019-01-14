package accept

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Fragment represents an addressable item that received ScrapeUpdates.
type Fragment interface {
	Match(update *central.ScrapeUpdate) bool
	AcceptUpdate(update *central.ScrapeUpdate)
}

// Accepter holds references to ongoing scrapes to update them.
type Accepter interface {
	AcceptUpdate(update *central.ScrapeUpdate)

	AddFragment(fragment Fragment)
	RemoveFragment(fragment Fragment)
}

// NewAccepter returns a new instance of a Accepter.
func NewAccepter() Accepter {
	accepter := &accepterImpl{
		fragments: make(map[Fragment]struct{}),
	}
	return accepter
}
