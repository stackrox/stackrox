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

func TestAutoSenseLocalScannerAlwaysReturnsDisabled(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, testutils.ValidClusterVersion).Build()

	config, err := AutoSenseLocalScannerConfig(context.Background(), client, securedCluster)
	require.NoError(t, err)
	assert.False(t, config.EnableLocalImageScanning)
	assert.False(t, config.DeployScannerResources)
}

func TestAutoSenseLocalScannerIgnoresCentralPresence(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, testutils.ValidClusterVersion, &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testutils.TestNamespace,
			Name:      "central",
		},
		Spec: platform.CentralSpec{},
	}).Build()

	config, err := AutoSenseLocalScannerConfig(context.Background(), client, securedCluster)
	require.NoError(t, err)
	assert.False(t, config.DeployScannerResources)
	assert.False(t, config.EnableLocalImageScanning)
}

func TestAutoSenseLocalScannerIgnoresCentralInDifferentNamespace(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, testutils.ValidClusterVersion, &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "another-namespace",
			Name:      "central",
		},
		Spec: platform.CentralSpec{},
	}).Build()

	config, err := AutoSenseLocalScannerConfig(context.Background(), client, securedCluster)
	require.NoError(t, err)
	assert.False(t, config.DeployScannerResources)
	assert.False(t, config.EnableLocalImageScanning)
}
