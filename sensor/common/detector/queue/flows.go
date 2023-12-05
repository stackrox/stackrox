package queue

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/queue"
)

// FlowQueueItem defines a item for the NetworkFlowsQueue
type FlowQueueItem struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Flow       *augmentedobjs.NetworkFlowDetails
	Netpols    *augmentedobjs.NetworkPoliciesApplied
}

// NetworkFlowsQueue wraps a PausableQueue to make it pullable with a channel.
type NetworkFlowsQueue struct {
	queue   queue.PausableQueue[*FlowQueueItem]
	outputC chan *FlowQueueItem
	stopper concurrency.Stopper
}

// NewNetworkFlowQueue creates a new NetworkFlowQueue.
func NewNetworkFlowQueue(stopper concurrency.Stopper, queue queue.PausableQueue[*FlowQueueItem]) *NetworkFlowsQueue {
	return &NetworkFlowsQueue{
		queue:   queue,
		outputC: make(chan *FlowQueueItem),
		stopper: stopper,
	}
}

// Start the queue.
func (n *NetworkFlowsQueue) Start() {
	// TODO(ROX-21052): Resuming, pausing, and stopping the internal queue should be done in the QueueManager
	n.queue.Resume()
	go n.run()
}

// Push an item to the queue
func (n *NetworkFlowsQueue) Push(item *FlowQueueItem) {
	n.queue.Push(item)
}

func (n *NetworkFlowsQueue) run() {
	defer close(n.outputC)
	// TODO(ROX-21052): Resuming, pausing, and stopping the internal queue should be done in the QueueManager
	defer n.queue.Stop()
	for {
		select {
		case <-n.stopper.Flow().StopRequested():
			return
		default:
			n.outputC <- n.queue.PullBlocking()
		}
	}
}

// Pull returns the channel where run writes the front of the queue.
func (n *NetworkFlowsQueue) Pull() <-chan *FlowQueueItem {
	return n.outputC
}
