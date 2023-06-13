package trace

import (
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once           sync.Once
	authzTraceSink observe.AuthzTraceSink
)

func initialize() {
	authzTraceSink = observe.NewAuthzTraceSink()
}

// AuthzTraceSinkSingleton returns the authz trace sink instance.
func AuthzTraceSinkSingleton() observe.AuthzTraceSink {
	once.Do(initialize)
	return authzTraceSink
}
