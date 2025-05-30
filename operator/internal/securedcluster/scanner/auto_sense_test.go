package scanner

import (
	"context"
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
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
	client := testutils.NewFakeClientBuilder(t, testutils.ValidClusterVersion).Build()

	config, err := AutoSenseLocalScannerConfig(context.Background(), client, securedCluster)
	require.NoError(t, err)
	assert.True(t, config.EnableLocalImageScanning)
	assert.True(t, config.DeployScannerResources)
}

func TestAutoSenseIsDisabledWithCentralPresentShouldBeDisabled(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, testutils.ValidClusterVersion, &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testutils.TestNamespace,
			Name:      "central",
		},
		Spec: platform.CentralSpec{},
	}).Build()

	config, err := AutoSenseLocalScannerConfig(context.Background(), client, securedCluster)
	require.NoError(t, err)
	assert.False(t, config.DeployScannerResources, "Expected Scanner resource deployment to be disabled if Central is present")
	assert.True(t, config.EnableLocalImageScanning, "Expected Local Image Scanning feature to be enabled.")
}

func TestAutoSenseIsEnabledWithCentralInADifferentNamespace(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, testutils.ValidClusterVersion, &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "another-namespace",
			Name:      "central",
		},
		Spec: platform.CentralSpec{},
	}).Build()

	config, err := AutoSenseLocalScannerConfig(context.Background(), client, securedCluster)
	require.NoError(t, err)
	require.True(t, config.DeployScannerResources)
	require.True(t, config.EnableLocalImageScanning)
}
