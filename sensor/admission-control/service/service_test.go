package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	policiesTesting "github.com/stackrox/rox/pkg/defaults/policies/testing"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/admission-control/manager"
	managerTesting "github.com/stackrox/rox/sensor/admission-control/manager/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
)

const (
	ExecIntoPodPolicyName = "Kubernetes Actions: Exec into Pod"
	LatestTagPolicyName   = "Latest tag"
)

func TestExecIntoPodNameEventPolicy(t *testing.T) {
	policy, err := policiesTesting.GetDefaultPolicy(t, ExecIntoPodPolicyName)
	require.NoError(t, err)

	mgr := managerTesting.NewTestManager(t,
		managerTesting.TestManagerOptions{Policy: policy},
	)

	mgr.Start()
	defer mgr.Stop()

	const deploymentID = "f3237faf-8350-4c39-b045-ff4c493ddb71"
	managerTesting.ProcessDeploymentEvent(t, mgr, &storage.Deployment{
		Id:        deploymentID,
		Name:      "sensor",
		Type:      "Deployment",
		Namespace: "stackrox",
	})
	managerTesting.ProcessPodEvent(t, mgr, &storage.Pod{
		Id:           "64a1d6ee-2425-5f19-990e-a2d8b18c1e4c",
		Name:         "sensor-74f6965874-qckz6",
		DeploymentId: deploymentID,
		Namespace:    "stackrox",
	})

	r := serviceTestRun{
		mgr:               mgr,
		reviewRequestPath: "testdata/review_requests/pod_exec_event_review.json",
		handlerFunc:       (*service).handleK8sEvents,
		assertionFunc: func(t *testing.T, resp *http.Response, alerts []*storage.Alert) {
			require.NotNil(t, alerts)
			require.Len(t, alerts, 1)
			assert.Equal(t, "Kubernetes Actions: Exec into Pod", alerts[0].GetPolicy().GetName())

			violations := alerts[0].GetViolations()
			require.Len(t, violations, 1)
			assert.Equal(t, "Kubernetes API received exec '/bin/sh' request into pod 'sensor-74f6965874-qckz6' container 'sensor'", violations[0].GetMessage())
			assert.Equal(t, "sensor", alerts[0].GetDeployment().GetName())

			review := readV1AdmissionReview(t, resp)
			assert.True(t, review.Response.Allowed)
		},
		t: t,
	}
	r.execute()
}

func TestLatestTagPolicyAdmissionReview(t *testing.T) {
	policy, err := policiesTesting.GetDefaultPolicy(t, LatestTagPolicyName)
	require.NoError(t, err)

	policy.EnforcementActions = []storage.EnforcementAction{
		storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
	}

	mgr := managerTesting.NewTestManager(t, managerTesting.TestManagerOptions{
		AdmissionControllerSettings: &sensor.AdmissionControlSettings{
			ClusterId: uuid.NewDummy().String(),
			ClusterConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					EnforceOnUpdates: true,
					Enabled:          true,
				},
			},
		},
		Policy: policy,
		ImageServiceResponse: &sensor.GetImageResponse{
			Image: &storage.Image{
				Id: "sha256:e66b2e83961df8f87a4a20c0365b1404d60cdd58798f4db5763332fe0ac235ea",
				Name: &storage.ImageName{
					Registry: "docker.io",
					Remote:   "library/nginx",
					Tag:      "latest",
					FullName: "docker.io/library/nginx:latest",
				},
			},
		},
	})

	mgr.Start()
	defer mgr.Stop()

	runv1 := serviceTestRun{
		mgr:               mgr,
		handlerFunc:       (*service).handleValidate,
		reviewRequestPath: "testdata/review_requests/latest_tag_admission_review_v1.json",
		assertionFunc: func(t *testing.T, resp *http.Response, alerts []*storage.Alert) {
			require.NotNil(t, alerts)
			require.Len(t, alerts, 1)
			assert.Equal(t, LatestTagPolicyName, alerts[0].GetPolicy().GetName())
			require.Len(t, alerts[0].GetViolations(), 1)
			assert.Equal(t, "Container 'nginx' has image with tag 'latest'", alerts[0].GetViolations()[0].Message)

			review := readV1AdmissionReview(t, resp)
			assert.Equal(t, admissionv1.SchemeGroupVersion.String(), review.APIVersion)
		},
		t: t,
	}

	runv1.execute()

	runv1beta1 := serviceTestRun{
		mgr: mgr,
		assertionFunc: func(t *testing.T, resp *http.Response, alerts []*storage.Alert) {
			const latestTagErrMessage = "Container 'nginx' has image with tag 'latest'"
			require.NotNil(t, alerts)
			require.Len(t, alerts, 1)
			assert.Equal(t, LatestTagPolicyName, alerts[0].GetPolicy().GetName())
			require.Len(t, alerts[0].GetViolations(), 1)
			assert.Equal(t, latestTagErrMessage, alerts[0].GetViolations()[0].Message)

			review := readV1beta1AdmissionReview(t, resp)
			assert.Contains(t, review.Response.Result.Message, latestTagErrMessage)
			assert.False(t, review.Response.Allowed)
			assert.Equal(t, admissionv1beta1.SchemeGroupVersion.String(), review.APIVersion)
		},
		handlerFunc:       (*service).handleValidate,
		reviewRequestPath: "testdata/review_requests/latest_tag_admission_review_v1beta1.json",
		t:                 t,
	}

	runv1beta1.execute()
}

type serviceTestRun struct {
	mgr               manager.Manager
	reviewRequestPath string
	assertionFunc     func(t *testing.T, resp *http.Response, alerts []*storage.Alert)
	// handlerFunc is the service handler func to be tested
	handlerFunc func(*service, http.ResponseWriter, *http.Request)
	t           *testing.T
}

// execute runs the review request through the handler and then
// runs alerts from manager through the assertion function.
func (r serviceTestRun) execute() {
	require.NotNil(r.t, r.mgr)
	require.NotNil(r.t, r.handlerFunc)
	require.True(r.t, r.mgr.IsReady(), "Manager is stopped or was not started")
	// Wait for any events delivered to manager prior to this call to be processed.
	syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(r.t, r.mgr.Sync(syncCtx))

	s := service{
		mgr: r.mgr,
	}

	requestBody, err := os.ReadFile(r.reviewRequestPath)
	require.NoError(r.t, err)

	req := httptest.NewRequest(http.MethodPost, "https://some-admission-url.stackrox:443", bytes.NewBuffer(requestBody))
	resp := httptest.NewRecorder()

	// Execute the review request
	r.handlerFunc(&s, resp, req)

	require.NotNil(r.t, resp)
	assert.Equal(r.t, http.StatusOK, resp.Code)

	select {
	case <-time.After(3 * time.Second):
		assert.Fail(r.t, "Did not receive any alerts before timeout expired, but expected some")
	case alerts := <-r.mgr.Alerts():
		r.assertionFunc(r.t, resp.Result(), alerts)
	}
}

func readV1beta1AdmissionReview(t *testing.T, resp *http.Response) admissionv1beta1.AdmissionReview {
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	review := admissionv1beta1.AdmissionReview{}
	err = json.Unmarshal(respBody, &review)
	require.NoError(t, err)
	return review
}

func readV1AdmissionReview(t *testing.T, resp *http.Response) admissionv1.AdmissionReview {
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	review := admissionv1.AdmissionReview{}
	err = json.Unmarshal(respBody, &review)
	require.NoError(t, err)
	return review
}
