package utils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/operator/api/central/v1alpha1"
	common "github.com/stackrox/rox/operator/api/common/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestUpdateStatusCondition_AddCondition(t *testing.T) {
	var status v1alpha1.CentralStatus

	var uSt unstructured.Unstructured
	var err error
	uSt.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
	require.NoError(t, err)

	preUpdateTS := time.Unix(time.Now().Unix(), 0) // status condition time only exists at second granularity
	updated := updateStatusCondition(&uSt, string(common.ConditionDeployed), metav1.ConditionFalse, string(common.ReasonReconcileError), "reconcile error")

	assert.True(t, updated)
	var postStatus v1alpha1.CentralStatus
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &postStatus))

	require.Len(t, postStatus.Conditions, 1)
	assert.Equal(t, common.ConditionDeployed, postStatus.Conditions[0].Type)
	assert.EqualValues(t, common.StatusFalse, postStatus.Conditions[0].Status)
	assert.EqualValues(t, common.ReasonReconcileError, postStatus.Conditions[0].Reason)
	assert.Equal(t, "reconcile error", postStatus.Conditions[0].Message)
	assert.False(t, preUpdateTS.After(postStatus.Conditions[0].LastTransitionTime.Time))
}

func TestUpdateStatusCondition_UpdateCondition_NoStatusChange(t *testing.T) {
	tenSecAgo := metav1.NewTime(time.Unix(time.Now().Add(-10*time.Second).Unix(), 0))
	status := &v1alpha1.CentralStatus{
		Conditions: []common.StackRoxCondition{
			{
				Type:               common.ConditionDeployed,
				Status:             common.StatusFalse,
				Reason:             common.ReasonReconcileError,
				Message:            "no reconcile",
				LastTransitionTime: tenSecAgo,
			},
		},
	}

	var uSt unstructured.Unstructured
	var err error
	uSt.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
	require.NoError(t, err)

	updated := updateStatusCondition(&uSt, string(common.ConditionDeployed), metav1.ConditionFalse, string(common.ReasonInstallError), "install error")

	assert.True(t, updated)
	var postStatus v1alpha1.CentralStatus
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &postStatus))

	require.Len(t, postStatus.Conditions, 1)
	assert.Equal(t, common.ConditionDeployed, postStatus.Conditions[0].Type)
	assert.EqualValues(t, common.StatusFalse, postStatus.Conditions[0].Status)
	assert.EqualValues(t, common.ReasonInstallError, postStatus.Conditions[0].Reason)
	assert.Equal(t, "install error", postStatus.Conditions[0].Message)
	assert.Equal(t, tenSecAgo, postStatus.Conditions[0].LastTransitionTime)
}

func TestUpdateStatusCondition_UpdateCondition_WithStatusChange(t *testing.T) {
	tenSecAgo := metav1.NewTime(time.Unix(time.Now().Add(-10*time.Second).Unix(), 0))
	status := &v1alpha1.CentralStatus{
		Conditions: []common.StackRoxCondition{
			{
				Type:               common.ConditionDeployed,
				Status:             common.StatusFalse,
				Reason:             common.ReasonReconcileError,
				Message:            "no reconcile",
				LastTransitionTime: tenSecAgo,
			},
		},
	}

	var uSt unstructured.Unstructured
	var err error
	uSt.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
	require.NoError(t, err)

	preUpdateTS := time.Unix(time.Now().Unix(), 0)
	updated := updateStatusCondition(&uSt, string(common.ConditionDeployed), metav1.ConditionTrue, "AllGood", "everything okay")

	assert.True(t, updated)
	var postStatus v1alpha1.CentralStatus
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &postStatus))

	require.Len(t, postStatus.Conditions, 1)
	assert.Equal(t, common.ConditionDeployed, postStatus.Conditions[0].Type)
	assert.EqualValues(t, common.StatusTrue, postStatus.Conditions[0].Status)
	assert.EqualValues(t, "AllGood", postStatus.Conditions[0].Reason)
	assert.Equal(t, "everything okay", postStatus.Conditions[0].Message)
	assert.False(t, preUpdateTS.After(postStatus.Conditions[0].LastTransitionTime.Time))
}

func TestUpdateStatusCondition_UpdateCondition_AddToExisting(t *testing.T) {
	tenSecAgo := metav1.NewTime(time.Unix(time.Now().Add(-10*time.Second).Unix(), 0))
	status := &v1alpha1.CentralStatus{
		Conditions: []common.StackRoxCondition{
			{
				Type:               common.ConditionDeployed,
				Status:             common.StatusFalse,
				Reason:             common.ReasonReconcileError,
				Message:            "no reconcile",
				LastTransitionTime: tenSecAgo,
			},
		},
	}

	var uSt unstructured.Unstructured
	var err error
	uSt.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
	require.NoError(t, err)

	preUpdateTS := time.Unix(time.Now().Unix(), 0)
	updated := updateStatusCondition(&uSt, string(common.ConditionReleaseFailed), metav1.ConditionTrue, "ReleaseFailed", "the release failed")

	assert.True(t, updated)
	var postStatus v1alpha1.CentralStatus
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &postStatus))

	require.Len(t, postStatus.Conditions, 2)
	assert.Equal(t, postStatus.Conditions[0], status.Conditions[0])
	assert.Equal(t, common.ConditionReleaseFailed, postStatus.Conditions[1].Type)
	assert.EqualValues(t, common.StatusTrue, postStatus.Conditions[1].Status)
	assert.EqualValues(t, "ReleaseFailed", postStatus.Conditions[1].Reason)
	assert.Equal(t, "the release failed", postStatus.Conditions[1].Message)
	assert.False(t, preUpdateTS.After(postStatus.Conditions[1].LastTransitionTime.Time))
}
