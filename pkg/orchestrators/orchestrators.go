package orchestrators

import (
	v1 "k8s.io/api/core/v1"
)

// Creator is a function stub that defined how to create a Orchestrator
type Creator func() (Orchestrator, error)

// Orchestrator is the interface that allows for actions against an orchestrator
//go:generate mockgen-wrapper
type Orchestrator interface {
	GetNode(nodeName string) (*v1.Node, error)
}
