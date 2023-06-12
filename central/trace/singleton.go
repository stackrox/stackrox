package trace

import (
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton observe.AuthzTraceSink
)

func initialize() {
	singleton = observe.NewAuthzTraceSink()
}

// AuthzTraceSinkSingleton returns the authz trace sink instance.
func AuthzTraceSinkSingleton() observe.AuthzTraceSink {
	once.Do(initialize)
	return singleton
}
