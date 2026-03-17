//go:build test_e2e

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/namespaces"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// waitForCollectorReady waits for the collector DaemonSet to have all pods
// updated and ready.
func waitForCollectorReady(t *testing.T, client kubernetes.Interface) {
	t.Helper()

	waitForCondition(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ds, err := client.AppsV1().DaemonSets(namespaces.StackRox).Get(ctx, "collector", metaV1.GetOptions{})
		if err != nil {
			return false
		}
		if ds.Status.DesiredNumberScheduled == 0 || ds.Status.UpdatedNumberScheduled != ds.Status.DesiredNumberScheduled {
			return false
		}
		return ds.Status.NumberReady == ds.Status.DesiredNumberScheduled
	}, "collector DaemonSet ready", 5*time.Minute, 5*time.Second)
}
