package testing

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/sensor/admission-control/manager"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// ProcessPodEvent adds a pod to the admission controllers pod storage. In a production system the given pod is running inside a cluster. The
// admission controller run its policy evaluations on them.
func ProcessPodEvent(t *testing.T, mgr manager.Manager, pod *storage.Pod) {
	if t == nil {
		panic("This function must be called from a test.")
	}
	require.True(t, mgr.IsReady())

	mgr.ResourceUpdatesC() <- &sensor.AdmCtrlUpdateResourceRequest{
		Resource: &sensor.AdmCtrlUpdateResourceRequest_Pod{Pod: pod},
		Action:   central.ResourceAction_CREATE_RESOURCE,
	}
}

// ProcessDeploymentEvent adds deployment to the admission controller deployment storage. In a production system
// the given deployment is running inside a cluster. The admission controller runs its policy evaluations on them.
func ProcessDeploymentEvent(t *testing.T, mgr manager.Manager, deployment *storage.Deployment) {
	if t == nil {
		panic("This function must be called from a test.")
	}
	require.True(t, mgr.IsReady())

	mgr.ResourceUpdatesC() <- &sensor.AdmCtrlUpdateResourceRequest{
		Resource: &sensor.AdmCtrlUpdateResourceRequest_Deployment{Deployment: deployment},
		Action:   central.ResourceAction_CREATE_RESOURCE,
	}
}

func addPolicyToSettings(t *testing.T, settings *sensor.AdmissionControlSettings, policy *storage.Policy) {
	if t == nil {
		panic("This function must be called from a test.")
	}

	var deploytimePolicies, runtimePolicies []*storage.Policy
	if policies.AppliesAtDeployTime(policy) {
		deploytimePolicies = append(deploytimePolicies, policy)
	}
	if policies.AppliesAtRunTime(policy) {
		runtimePolicies = append(runtimePolicies, policy)
	}

	settings.RuntimePolicies = &storage.PolicyList{Policies: runtimePolicies}
	settings.EnforcedDeployTimePolicies = &storage.PolicyList{Policies: deploytimePolicies}
}

// TestManagerOptions define the options for the testing manager.
type TestManagerOptions struct {
	Policy                      *storage.Policy
	ImageServiceResponse        *sensor.GetImageResponse
	GetDeploymentForPodResponse *storage.Deployment
	AdmissionControllerSettings *sensor.AdmissionControlSettings
}

// NewTestManager creates a new manager which is used for testing
func NewTestManager(t *testing.T, opts TestManagerOptions) manager.Manager {
	if t == nil {
		panic("NewTestManager is only allowed to be called in tests")
	}

	settings := &sensor.AdmissionControlSettings{}
	if opts.AdmissionControllerSettings != nil {
		settings = opts.AdmissionControllerSettings
	}

	imageServiceClient := testingImageServiceClient{GetImageResponse: opts.ImageServiceResponse, t: t}
	deploymentServiceClient := testingDeploymentServiceClient{GetDeploymentForPodResponse: opts.GetDeploymentForPodResponse, t: t}

	mgr := manager.NewManager("stackrox", 20*size.MB, imageServiceClient, deploymentServiceClient)

	addPolicyToSettings(t, settings, opts.Policy)
	mgr.ProcessNewSettings(settings)

	return mgr
}

var _ sensor.DeploymentServiceClient = (*testingDeploymentServiceClient)(nil)

type testingDeploymentServiceClient struct {
	GetDeploymentForPodResponse *storage.Deployment
	t                           *testing.T
}

func (t testingDeploymentServiceClient) GetDeploymentForPod(_ context.Context, _ *sensor.GetDeploymentForPodRequest, _ ...grpc.CallOption) (*storage.Deployment, error) {
	if t.GetDeploymentForPodResponse == nil {
		require.Fail(t.t, "unexpected call to GetDeploymentForPod")
	}
	return t.GetDeploymentForPodResponse, nil
}

var _ sensor.ImageServiceClient = (*testingImageServiceClient)(nil)

type testingImageServiceClient struct {
	GetImageResponse *sensor.GetImageResponse
	t                *testing.T
}

func (t testingImageServiceClient) GetImage(_ context.Context, _ *sensor.GetImageRequest, _ ...grpc.CallOption) (*sensor.GetImageResponse, error) {
	if t.GetImageResponse == nil {
		require.Fail(t.t, "unexpected call to GetImage")
	}
	return t.GetImageResponse, nil
}
