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
	_, err := ScrapePodViaProxy(ctx, cs, PodScrapeOptions{Namespace: "ns"})
	require.Error(t, err)
	_, err = ScrapePodViaProxy(ctx, cs, PodScrapeOptions{PodName: "p"})
	require.Error(t, err)
}

func TestCollectFromPods_Validation(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	_, err := CollectFromPods(ctx, cs, PodCollectOptions{})
	require.Error(t, err)
	_, err = CollectFromPods(ctx, cs, PodCollectOptions{
		Namespace:     "ns",
		Transport:     TransportPortForward,
		LabelSelector: "app=x",
	})
	require.Error(t, err)
}

func TestScrapePodViaPortForward_Validation(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	_, err := ScrapePodViaPortForward(ctx, cs, nil, "ns", "pod", 9090, "/metrics")
	require.Error(t, err)
	_, err = ScrapePodViaPortForward(ctx, cs, &rest.Config{}, "", "pod", 9090, "")
	require.Error(t, err)
	_, err = ScrapePodViaPortForward(ctx, cs, &rest.Config{}, "ns", "pod", 0, "")
	require.Error(t, err)
}
