package helper

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
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

// GetContainerIdsFromDeployment retrieves the container ids from a given deployment.
func (c *TestContext) GetContainerIdsFromDeployment(obj k8s.Object) ([]string, map[string][]string) {
	containerIDs := make(map[string][]string)
	var podIDs []string
	timeout := time.NewTimer(time.Minute)
	ticker := time.NewTicker(defaultTicker)
	for {
		select {
		case <-timeout.C:
			return podIDs, containerIDs
		case <-ticker.C:
			objs := &v1.PodList{}
			if err := c.r.List(context.Background(), objs,
				resources.WithLabelSelector(labels.SelectorFromSet(obj.GetLabels()).String())); err != nil {
				continue
			}
			for _, pod := range objs.Items {
				if len(pod.Status.ContainerStatuses) == 0 {
					continue
				}
				var podIds []string
				for _, con := range pod.Status.ContainerStatuses {
					if con.Ready {
						_, id := k8sutil.ParseContainerRuntimeString(con.ContainerID)
						podIds = append(podIds, containerid.ShortContainerIDFromInstanceID(id))
					}
				}
				if len(podIds) == len(pod.Status.ContainerStatuses) {
					containerIDs[string(pod.GetUID())] = podIds
				}
			}
			if len(containerIDs) == len(objs.Items) {
				for k := range containerIDs {
					podIDs = append(podIDs, k)
				}
				return podIDs, containerIDs
			}
		}
	}
}

// GetIPFromPod retrieves the IP of a given pod.
func (c *TestContext) GetIPFromPod(obj k8s.Object) string {
	timeout := time.NewTimer(time.Minute)
	ticker := time.NewTicker(defaultTicker)
	for {
		select {
		case <-timeout.C:
			return ""
		case <-ticker.C:
			if err := c.r.Get(context.Background(), obj.GetName(), obj.GetNamespace(), obj); err != nil {
				continue
			}
			pod, ok := obj.(*v1.Pod)
			if !ok {
				return ""
			}
			if len(pod.Status.ContainerStatuses) == 0 {
				continue
			}
			if pod.Status.PodIP == "" {
				continue
			}
			return pod.Status.PodIP
		}
	}
}

// GetIPFromService retrieves the IP from a given service.
func (c *TestContext) GetIPFromService(obj k8s.Object) string {
	timeout := time.NewTimer(time.Minute)
	ticker := time.NewTicker(defaultTicker)
	for {
		select {
		case <-timeout.C:
			return ""
		case <-ticker.C:
			if err := c.r.Get(context.Background(), obj.GetName(), obj.GetNamespace(), obj); err != nil {
				continue
			}
			srv, ok := obj.(*v1.Service)
			if !ok {
				return ""
			}
			if srv.Spec.ClusterIP == "" {
				continue
			}
			return srv.Spec.ClusterIP
		}
	}
}
