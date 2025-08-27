package queue

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type FileActivityQueueItem struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Node       *storage.Node
	Activity   *storage.FileActivity
}
