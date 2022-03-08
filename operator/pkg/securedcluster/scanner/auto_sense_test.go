package scanner

import (
	"context"
	"testing"

	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var securedCluster = platform.SecuredCluster{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "secured-cluster",
		Namespace: testutils.TestNamespace,
	},
}

func TestAutoSenseLocalScannerSupportShouldBeEnabled(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t).Build()

	enabled, err := AutoSenseLocalScannerSupport(context.Background(), client, securedCluster)
	require.NoError(t, err)
	assert.True(t, enabled, "Expected Scanner to be enabled if Central is not present")
}

func TestAutoSenseIsDisabledWithCentralPresentShouldBeDisabled(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testutils.TestNamespace,
			Name:      "central",
		},
		Spec: platform.CentralSpec{},
	}).Build()

	enabled, err := AutoSenseLocalScannerSupport(context.Background(), client, securedCluster)
	require.NoError(t, err)
	require.False(t, enabled, "Expected Scanner to be disabled if Central is present")
}

func TestAutoSenseIsEnabledWithCentralInADifferentNamespace(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "another-namespace",
			Name:      "central",
		},
		Spec: platform.CentralSpec{},
	}).Build()

	enabled, err := AutoSenseLocalScannerSupport(context.Background(), client, securedCluster)
	require.NoError(t, err)
	require.True(t, enabled, "Expected Scanner to be enabled if Central is deployed in a different namespace")
}
