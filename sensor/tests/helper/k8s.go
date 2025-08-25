package helper

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

func getGVR(api apiMetaV1.APIResource) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    api.Group,
		Version:  api.Version,
		Resource: api.Name,
	}
}

// AssertResourceDoesExist asserts whether the given resource exits in the cluster
func (c *TestContext) AssertResourceDoesExist(ctx context.Context, t *testing.T, resourceName string, namespace string, api apiMetaV1.APIResource) *unstructured.Unstructured {
	t.Helper()
	client, err := dynamic.NewForConfig(c.r.GetConfig())
	require.NoError(t, err)

	var obj *unstructured.Unstructured
	cli := clientForResource(client, api, namespace)
	require.Eventuallyf(t, func() bool {
		obj, err = cli.Get(ctx, resourceName, apiMetaV1.GetOptions{})
		return err == nil
	}, 30*time.Second, 10*time.Millisecond, "Resource %s/%s of kind %s should exist", namespace, resourceName, api.Kind)
	return obj
}

// AssertResourceWasUpdated asserts whether the given resource was updated in the cluster
func (c *TestContext) AssertResourceWasUpdated(ctx context.Context, t *testing.T, resourceName string, namespace string, api apiMetaV1.APIResource, oldResourceVersion string) *unstructured.Unstructured {
	t.Helper()
	client, err := dynamic.NewForConfig(c.r.GetConfig())
	require.NoError(t, err)

	var obj *unstructured.Unstructured
	cli := clientForResource(client, api, namespace)
	require.Eventuallyf(t, func() bool {
		obj, err = cli.Get(ctx, resourceName, apiMetaV1.GetOptions{})
		return err == nil && obj.GetResourceVersion() != oldResourceVersion
	}, 30*time.Second, 10*time.Millisecond, "Resource %s/%s of kind %s should be updated (old version: %s)", namespace, resourceName, api.Kind, oldResourceVersion)
	return obj
}

// AssertResourceDoesNotExist asserts whether the given resource does not exit in the cluster
func (c *TestContext) AssertResourceDoesNotExist(ctx context.Context, t *testing.T, resourceName string, namespace string, api apiMetaV1.APIResource) {
	t.Helper()
	client, err := dynamic.NewForConfig(c.r.GetConfig())
	require.NoError(t, err)

	cli := clientForResource(client, api, namespace)
	require.Eventuallyf(t, func() bool {
		_, err = cli.Get(ctx, resourceName, apiMetaV1.GetOptions{})
		return err != nil && kubeAPIErr.IsNotFound(err)
	}, 30*time.Second, 10*time.Millisecond, "Resource %s/%s of kind %s should not exist", namespace, resourceName, api.Kind)
}

// WaitForResourceDelete wait for a resource to be deleted
func (c *TestContext) WaitForResourceDelete(ctx context.Context, t *testing.T, resourceName string, namespace string, api apiMetaV1.APIResource) {
	t.Helper()
	client, err := dynamic.NewForConfig(c.r.GetConfig())
	require.NoError(t, err)

	var obj *unstructured.Unstructured
	cli := clientForResource(client, api, namespace)
	require.Eventuallyf(t, func() bool {
		obj, err = cli.Get(ctx, resourceName, apiMetaV1.GetOptions{})
		return err == nil || kubeAPIErr.IsNotFound(err)
	}, 30*time.Second, 10*time.Millisecond, "Resource %s/%s of kind %s should be deleted or not found", namespace, resourceName, api.Kind)

	if obj != nil {
		if err := wait.For(conditions.New(c.r).ResourceDeleted(obj)); err != nil {
			t.Logf("failed to wait for resource %s deletion", resourceName)
		}
	}
}

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

func clientForResource(client *dynamic.DynamicClient, api apiMetaV1.APIResource, namespace string) dynamic.ResourceInterface {
	if namespace != "" {
		return client.Resource(getGVR(api)).Namespace(namespace)
	}
	return client.Resource(getGVR(api))
}
