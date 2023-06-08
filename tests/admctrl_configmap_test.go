package tests

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdmissionControllerConfigMapWithPostgres(t *testing.T) {

	k8sClient := createK8sClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cm, err := k8sClient.CoreV1().ConfigMaps(namespaces.StackRox).Get(ctx, admissioncontrol.ConfigMapName, metav1.GetOptions{})
	require.NoError(t, err, "could not obtain admission controller configmap")

	policiesGZData := cm.BinaryData[admissioncontrol.DeployTimePoliciesGZDataKey]
	configGZData := cm.BinaryData[admissioncontrol.ConfigGZDataKey]

	timestamp := cm.Data[admissioncontrol.LastUpdateTimeDataKey]
	ts, err := time.Parse(time.RFC3339Nano, timestamp)
	assert.NoErrorf(t, err, "unparseable last update timestamp %q", timestamp)

	policiesData, err := gziputil.Decompress(policiesGZData)
	require.NoError(t, err, "missing or corrupted policies data in config map")
	configData, err := gziputil.Decompress(configGZData)
	require.NoError(t, err, "missing or corrupted config data in config map")

	var policyList storage.PolicyList
	require.NoError(t, proto.Unmarshal(policiesData, &policyList), "could not unmarshal policies list")

	var config storage.DynamicClusterConfig
	require.NoError(t, proto.Unmarshal(configData, &config), "could not unmarshal config")

	cc := centralgrpc.GRPCConnectionToCentral(t)

	policyServiceClient := v1.NewPolicyServiceClient(cc)
	newPolicy := &storage.Policy{
		Name:        "testpolicy_" + t.Name() + "_" + uuid.NewV4().String(),
		Description: "test deploy time policy",
		Rationale:   "test deploy time policy",
		Categories:  []string{"Test"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ImageTag,
						Values: []*storage.PolicyValue{
							{
								Value: "admctrl-policy-test-tag",
							},
						},
					},
				},
			},
		},
		PolicyVersion:      "1.1",
		Severity:           storage.Severity_HIGH_SEVERITY,
		LifecycleStages:    []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	newPolicy, err = policyServiceClient.PostPolicy(ctx, &v1.PostPolicyRequest{
		Policy: newPolicy,
	})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = policyServiceClient.DeletePolicy(ctx, &v1.ResourceByID{Id: newPolicy.GetId()})
	}()
	require.NoError(t, err, "failed to create new policy")

	testutils.Retry(t, 10, 3*time.Second, func(t testutils.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		newCM, err := k8sClient.CoreV1().ConfigMaps(namespaces.StackRox).Get(ctx, admissioncontrol.ConfigMapName, metav1.GetOptions{})
		require.NoError(t, err, "could not obtain admission controller configmap")

		newTimestamp := newCM.Data[admissioncontrol.LastUpdateTimeDataKey]
		newTS, err := time.Parse(time.RFC3339Nano, newTimestamp)
		assert.NoErrorf(t, err, "unparseable last update timestamp %q", timestamp)
		assert.True(t, newTS.After(ts), "expected updated timestamp in configmap")

		newPoliciesGZData := newCM.BinaryData[admissioncontrol.DeployTimePoliciesGZDataKey]
		newConfigGZData := newCM.BinaryData[admissioncontrol.ConfigGZDataKey]

		newPoliciesData, err := gziputil.Decompress(newPoliciesGZData)
		require.NoError(t, err, "missing or corrupted policies data in config map")
		newConfigData, err := gziputil.Decompress(newConfigGZData)
		require.NoError(t, err, "missing or corrupted config data in config map")

		var newPolicyList storage.PolicyList
		require.NoError(t, proto.Unmarshal(newPoliciesData, &newPolicyList), "could not unmarshal policies list")
		assert.Len(t, newPolicyList.GetPolicies(), len(policyList.GetPolicies())+1, "expected one additional policy")
		numMatches := 0
		for _, policy := range newPolicyList.GetPolicies() {
			if policy.GetName() == newPolicy.GetName() {
				numMatches++
			}
		}
		assert.Equal(t, 1, numMatches, "expected new policy list to contain new policy exactly once")

		var newConfig storage.DynamicClusterConfig
		require.NoError(t, proto.Unmarshal(newConfigData, &newConfig), "could not unmarshal config")
		assert.True(t, proto.Equal(&newConfig, &config), "new and old config should be equal")
	})
}
