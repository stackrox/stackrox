package queue

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type FileAccessQueueItem struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Node       *storage.Node
	Access     *storage.FileAccess
}
