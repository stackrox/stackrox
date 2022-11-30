package phonehome

import (
	"context"

	"github.com/stackrox/rox/pkg/set"
)

// Properties collected by data gatherers.
type Properties = map[string]any

// Telemeter defines a common interface for telemetry gatherers.
//go:generate mockgen-wrapper
type Telemeter interface {
	Start()
	Stop()
	GetID() string
	Identify(props Properties)
	Track(event, userID string, props Properties)
	Group(groupID, userID string, props Properties)
}

// Config represents the central instance telemetry configuration.
type Config struct {
	CentralID  string
	TenantID   string
	APIPaths   set.FrozenSet[string]
	Properties Properties
}

// GatherFunc returns properties gathered by a data source.
type GatherFunc func(context.Context) (Properties, error)
