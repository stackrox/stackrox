package queue

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// DeploymentQueueItem defines a deployment item for the Queue
type DeploymentQueueItem struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Action     central.ResourceAction
}
