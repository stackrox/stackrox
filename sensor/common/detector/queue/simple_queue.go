package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/queue"
)

// NewSimpleQueue create a new pkg/queue
func NewSimpleQueue[T comparable](name string, size int, counter *prometheus.CounterVec, dropped prometheus.Counter) *queue.Queue[T] {
	var opts []queue.OptionFunc[T]
	if size > 0 {
		opts = append(opts, queue.WithMaxSize[T](size))
	}
	if counter != nil {
		opts = append(opts, queue.WithCounterVec[T](counter))
	}
	if dropped != nil {
		opts = append(opts, queue.WithDroppedMetric[T](dropped))
	}
	if name != "" {
		opts = append(opts, queue.WithQueueName[T](name))
	}
	return queue.NewQueue[T](opts...)
}
