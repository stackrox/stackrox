package trace

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc/metadata"
)

var (
	instance *backgroundEnhancer
	once     sync.Once
)

func withClusterID(ctx context.Context, clusterIDGetter func() string) context.Context {
	return metadata.NewOutgoingContext(ctx,
		metadata.Pairs(logging.ClusterIDContextValue, clusterIDGetter()),
	)
}

type backgroundEnhancer struct {
	clusterIDGetter func() string
	lock            sync.RWMutex
}

// Background creates a context based on context.Background with enriched trace values.
func Background() context.Context {
	once.Do(func() {
		instance = &backgroundEnhancer{
			clusterIDGetter: func() string {
				return ""
			},
		}
	})
	return instance.background()
}

// SetClusterIDGetter injects the ClusterIDGetter function to the backgroundEnhancer
func SetClusterIDGetter(fn func() string) {
	once.Do(func() {
		instance = &backgroundEnhancer{
			clusterIDGetter: fn,
		}
	})
	instance.lock.Lock()
	defer instance.lock.Unlock()
	instance.clusterIDGetter = fn
}

func (b *backgroundEnhancer) background() context.Context {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return withClusterID(context.Background(), b.clusterIDGetter)
}
