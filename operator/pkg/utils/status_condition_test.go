package utils

import (
	"testing"
	"time"

	platform "github.com/stackrox/stackrox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestUpdateStatusCondition_AddCondition(t *testing.T) {
	var status platform.CentralStatus

	var uSt unstructured.Unstructured
	var err error
	uSt.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
	require.NoError(t, err)

	preUpdateTS := time.Unix(time.Now().Unix(), 0) // status condition time only exists at second granularity
	updated := updateStatusCondition(&uSt, string(platform.ConditionDeployed), metav1.ConditionFalse, string(platform.ReasonReconcileError), "reconcile error")

	assert.True(t, updated)
	var postStatus platform.CentralStatus
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &postStatus))

	require.Len(t, postStatus.Conditions, 1)
	assert.Equal(t, platform.ConditionDeployed, postStatus.Conditions[0].Type)
	assert.EqualValues(t, platform.StatusFalse, postStatus.Conditions[0].Status)
	assert.EqualValues(t, platform.ReasonReconcileError, postStatus.Conditions[0].Reason)
	assert.Equal(t, "reconcile error", postStatus.Conditions[0].Message)
	assert.False(t, preUpdateTS.After(postStatus.Conditions[0].LastTransitionTime.Time))
}

func TestUpdateStatusCondition_UpdateCondition_NoStatusChange(t *testing.T) {
	tenSecAgo := metav1.NewTime(time.Unix(time.Now().Add(-10*time.Second).Unix(), 0))
	status := &platform.CentralStatus{
		Conditions: []platform.StackRoxCondition{
			{
				Type:               platform.ConditionDeployed,
				Status:             platform.StatusFalse,
				Reason:             platform.ReasonReconcileError,
				Message:            "no reconcile",
				LastTransitionTime: tenSecAgo,
			},
		},
	}

	var uSt unstructured.Unstructured
	var err error
	uSt.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
	require.NoError(t, err)

	updated := updateStatusCondition(&uSt, string(platform.ConditionDeployed), metav1.ConditionFalse, string(platform.ReasonInstallError), "install error")

	assert.True(t, updated)
	var postStatus platform.CentralStatus
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &postStatus))

	require.Len(t, postStatus.Conditions, 1)
	assert.Equal(t, platform.ConditionDeployed, postStatus.Conditions[0].Type)
	assert.EqualValues(t, platform.StatusFalse, postStatus.Conditions[0].Status)
	assert.EqualValues(t, platform.ReasonInstallError, postStatus.Conditions[0].Reason)
	assert.Equal(t, "install error", postStatus.Conditions[0].Message)
	assert.Equal(t, tenSecAgo, postStatus.Conditions[0].LastTransitionTime)
}

func TestUpdateStatusCondition_UpdateCondition_WithStatusChange(t *testing.T) {
	tenSecAgo := metav1.NewTime(time.Unix(time.Now().Add(-10*time.Second).Unix(), 0))
	status := &platform.CentralStatus{
		Conditions: []platform.StackRoxCondition{
			{
				Type:               platform.ConditionDeployed,
				Status:             platform.StatusFalse,
				Reason:             platform.ReasonReconcileError,
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
	updated := updateStatusCondition(&uSt, string(platform.ConditionDeployed), metav1.ConditionTrue, "AllGood", "everything okay")

	assert.True(t, updated)
	var postStatus platform.CentralStatus
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &postStatus))

	require.Len(t, postStatus.Conditions, 1)
	assert.Equal(t, platform.ConditionDeployed, postStatus.Conditions[0].Type)
	assert.EqualValues(t, platform.StatusTrue, postStatus.Conditions[0].Status)
	assert.EqualValues(t, "AllGood", postStatus.Conditions[0].Reason)
	assert.Equal(t, "everything okay", postStatus.Conditions[0].Message)
	assert.False(t, preUpdateTS.After(postStatus.Conditions[0].LastTransitionTime.Time))
}

func TestUpdateStatusCondition_UpdateCondition_AddToExisting(t *testing.T) {
	tenSecAgo := metav1.NewTime(time.Unix(time.Now().Add(-10*time.Second).Unix(), 0))
	status := &platform.CentralStatus{
		Conditions: []platform.StackRoxCondition{
			{
				Type:               platform.ConditionDeployed,
				Status:             platform.StatusFalse,
				Reason:             platform.ReasonReconcileError,
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
	updated := updateStatusCondition(&uSt, string(platform.ConditionReleaseFailed), metav1.ConditionTrue, "ReleaseFailed", "the release failed")

	assert.True(t, updated)
	var postStatus platform.CentralStatus
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &postStatus))

	require.Len(t, postStatus.Conditions, 2)
	assert.Equal(t, postStatus.Conditions[0], status.Conditions[0])
	assert.Equal(t, platform.ConditionReleaseFailed, postStatus.Conditions[1].Type)
	assert.EqualValues(t, platform.StatusTrue, postStatus.Conditions[1].Status)
	assert.EqualValues(t, "ReleaseFailed", postStatus.Conditions[1].Reason)
	assert.Equal(t, "the release failed", postStatus.Conditions[1].Message)
	assert.False(t, preUpdateTS.After(postStatus.Conditions[1].LastTransitionTime.Time))
}
