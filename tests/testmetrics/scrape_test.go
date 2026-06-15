//go:build test

package testmetrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestScrapePodViaProxy_Validation(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	_, err := scrapePodViaProxy(ctx, cs, "ns", "", 0, "")
	require.Error(t, err)
	_, err = scrapePodViaProxy(ctx, cs, "", "p", 0, "")
	require.Error(t, err)
}

func TestCollectFromPods_Validation(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	_, err := collectFromPods(ctx, cs, podCollectOptions{})
	require.Error(t, err)
	_, err = collectFromPods(ctx, cs, podCollectOptions{
		namespace:     "ns",
		transport:     TransportPortForward,
		labelSelector: "app=x",
	})
	require.Error(t, err)
}

func TestScrapePodViaPortForward_Validation(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	_, err := scrapePodViaPortForward(ctx, cs, nil, "ns", "pod", 9090, "/metrics")
	require.Error(t, err)
	_, err = scrapePodViaPortForward(ctx, cs, &rest.Config{}, "", "pod", 9090, "")
	require.Error(t, err)
	_, err = scrapePodViaPortForward(ctx, cs, &rest.Config{}, "ns", "pod", 0, "")
	require.Error(t, err)
}
