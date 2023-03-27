package testutils

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Interceptor is a client that intercepts calls to the underlying client and
// calls the provided functions instead. If the function is nil, the call is
// forwarded to the underlying client.
func Interceptor(client ctrlClient.WithWatch, fns InterceptorFns) ctrlClient.WithWatch {
	return interceptor{client: client, fns: fns}
}

// InterceptorFns contains functions that are called instead of the underlying
// client's methods.
type InterceptorFns struct {
	Get         func(ctx context.Context, client ctrlClient.WithWatch, key ctrlClient.ObjectKey, obj ctrlClient.Object, opts ...ctrlClient.GetOption) error
	List        func(ctx context.Context, client ctrlClient.WithWatch, list ctrlClient.ObjectList, opts ...ctrlClient.ListOption) error
	Create      func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.CreateOption) error
	Delete      func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.DeleteOption) error
	DeleteAllOf func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.DeleteAllOfOption) error
	Update      func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.UpdateOption) error
	Patch       func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, patch ctrlClient.Patch, opts ...ctrlClient.PatchOption) error
	Watch       func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.ObjectList, opts ...ctrlClient.ListOption) (watch.Interface, error)
}

type interceptor struct {
	client ctrlClient.WithWatch
	fns    InterceptorFns
}

var _ ctrlClient.WithWatch = &interceptor{}

func (c interceptor) Get(ctx context.Context, key ctrlClient.ObjectKey, obj ctrlClient.Object, opts ...ctrlClient.GetOption) error {
	if c.fns.Get != nil {
		return c.fns.Get(ctx, c.client, key, obj, opts...)
	}
	return c.client.Get(ctx, key, obj, opts...)
}

func (c interceptor) List(ctx context.Context, list ctrlClient.ObjectList, opts ...ctrlClient.ListOption) error {
	if c.fns.List != nil {
		return c.fns.List(ctx, c.client, list, opts...)
	}
	return c.client.List(ctx, list, opts...)
}

func (c interceptor) Create(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.CreateOption) error {
	if c.fns.Create != nil {
		return c.fns.Create(ctx, c.client, obj, opts...)
	}
	return c.client.Create(ctx, obj, opts...)
}

func (c interceptor) Delete(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.DeleteOption) error {
	if c.fns.Delete != nil {
		return c.fns.Delete(ctx, c.client, obj, opts...)
	}
	return c.client.Delete(ctx, obj, opts...)
}

func (c interceptor) Update(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.UpdateOption) error {
	if c.fns.Update != nil {
		return c.fns.Update(ctx, c.client, obj, opts...)
	}
	return c.client.Update(ctx, obj, opts...)
}

func (c interceptor) Patch(ctx context.Context, obj ctrlClient.Object, patch ctrlClient.Patch, opts ...ctrlClient.PatchOption) error {
	if c.fns.Patch != nil {
		return c.fns.Patch(ctx, c.client, obj, patch, opts...)
	}
	return c.client.Patch(ctx, obj, patch, opts...)
}

func (c interceptor) DeleteAllOf(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.DeleteAllOfOption) error {
	if c.fns.DeleteAllOf != nil {
		return c.fns.DeleteAllOf(ctx, c.client, obj, opts...)
	}
	return c.client.DeleteAllOf(ctx, obj, opts...)
}

func (c interceptor) Status() ctrlClient.SubResourceWriter {
	return c.client.Status()
}

func (c interceptor) SubResource(subResource string) ctrlClient.SubResourceClient {
	return c.client.SubResource(subResource)
}

func (c interceptor) Scheme() *runtime.Scheme {
	return c.client.Scheme()
}

func (c interceptor) RESTMapper() meta.RESTMapper {
	return c.client.RESTMapper()
}

func (c interceptor) Watch(ctx context.Context, obj ctrlClient.ObjectList, opts ...ctrlClient.ListOption) (watch.Interface, error) {
	if c.fns.Watch != nil {
		return c.fns.Watch(ctx, c.client, obj, opts...)
	}
	return c.client.Watch(ctx, obj, opts...)
}
