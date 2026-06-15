//go:build test

package testmetrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCollectFromPods_Validation(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	_, err := collectFromPods(ctx, cs, ScrapeTarget{})
	require.Error(t, err)
}
