package accept

import (
	"sync"

	"github.com/stackrox/rox/generated/internalapi/central"
)

type accepterImpl struct {
	lock      sync.Mutex
	fragments map[Fragment]struct{}
}

// AcceptUpdate forwards the update to a matching registered scrape.
func (s *accepterImpl) AcceptUpdate(update *central.ScrapeUpdate) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for fragment := range s.fragments {
		if fragment.Match(update) {
			fragment.AcceptUpdate(update)
		}
	}
}

func (s *accepterImpl) OnFinish() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for fragment := range s.fragments {
		fragment.OnFinish()
	}
}

func (s *accepterImpl) AddFragment(fragment Fragment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.fragments[fragment] = struct{}{}
}

func (s *accepterImpl) RemoveFragment(fragment Fragment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.fragments, fragment)
}
