package queue

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

type FileAccessQueueItem struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Node       *storage.Node
	Access     *storage.FileAccess
	Netpols    *augmentedobjs.NetworkPoliciesApplied
}
