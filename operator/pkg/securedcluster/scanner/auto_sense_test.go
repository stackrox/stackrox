package scanner

import (
	"context"
	"testing"

	osconfigv1 "github.com/openshift/api/config/v1"
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

var validClusterVersion = &osconfigv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testutils.TestNamespace,
		Name:      "version",
	},
	Spec: osconfigv1.ClusterVersionSpec{
		ClusterID: "test-cluster-id",
	},
}

func TestAutoSenseLocalScannerSupportShouldBeEnabled(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, validClusterVersion).Build()

	enabled, err := AutoSenseLocalScannerSupport(context.Background(), client, securedCluster)
	require.NoError(t, err)
	assert.True(t, enabled, "Expected Scanner to be enabled for OpenShift cluster if Central is not present")
}

func TestAutoSenseIsDisabledWithCentralPresentShouldBeDisabled(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, validClusterVersion, &platform.Central{
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
	client := testutils.NewFakeClientBuilder(t, validClusterVersion, &platform.Central{
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

func TestAutoSenseIsDisabledIfClusterVersionNotFound(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, &osconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testutils.TestNamespace,
			Name:      "not-version-name",
		},
		Spec: osconfigv1.ClusterVersionSpec{
			ClusterID: "test-cluster-id",
		},
	}).Build()

	enabled, err := AutoSenseLocalScannerSupport(context.Background(), client, securedCluster)
	require.Error(t, err)
	require.False(t, enabled, `Expected an error if clusterversions.config.openshift.io %q not found`, ClusterVersionDefaultName)
}

func TestAutoSenseIsDisabledIfClusterIdIsEmpty(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t, &osconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testutils.TestNamespace,
			Name:      "version",
		},
		Spec: osconfigv1.ClusterVersionSpec{
			ClusterID: "",
		},
	}).Build()

	enabled, err := AutoSenseLocalScannerSupport(context.Background(), client, securedCluster)
	require.NoError(t, err)
	require.False(t, enabled, "Expected Scanner to be disabled if clusterversions.ClusterID is empty")
}
