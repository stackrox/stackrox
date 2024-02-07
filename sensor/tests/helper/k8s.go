package helper

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

// GetContainerIdsFromPod retrieves the container ids from a given pod.
func (c *TestContext) GetContainerIdsFromPod(ctx context.Context, obj k8s.Object) []string {
	var ret []string
	timeout := time.NewTimer(time.Minute)
	ticker := time.NewTicker(defaultTicker)
	for {
		select {
		case <-timeout.C:
			return ret
		case <-ticker.C:
			if err := c.r.Get(ctx, obj.GetName(), obj.GetNamespace(), obj); err != nil {
				continue
			}
			pod, ok := obj.(*v1.Pod)
			if !ok {
				return ret
			}
			if len(pod.Status.ContainerStatuses) == 0 {
				continue
			}
			for _, con := range pod.Status.ContainerStatuses {
				if con.Ready {
					_, id := k8sutil.ParseContainerRuntimeString(con.ContainerID)
					ret = append(ret, containerid.ShortContainerIDFromInstanceID(id))
				}
			}
			if len(ret) == len(pod.Status.ContainerStatuses) {
				return ret
			}
		}
	}
}
