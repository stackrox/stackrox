package migratetooperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type scFakeSource struct {
	fakeSource
	clusterName string
	webhooks    []admissionv1.ValidatingWebhook
}

func (f *scFakeSource) Secret(name string) (*corev1.Secret, error) {
	if name == "helm-effective-cluster-name" && f.clusterName != "" {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			StringData: map[string]string{"cluster-name": f.clusterName},
		}, nil
	}
	return nil, nil
}

func (f *scFakeSource) ValidatingWebhookConfiguration(_ string) (*admissionv1.ValidatingWebhookConfiguration, error) {
	if f.webhooks == nil {
		return nil, nil
	}
	return &admissionv1.ValidatingWebhookConfiguration{Webhooks: f.webhooks}, nil
}

func defaultSCSource() *scFakeSource {
	ignore := admissionv1.Ignore
	return &scFakeSource{
		fakeSource: fakeSource{
			deployments: map[string]*appsv1.Deployment{
				"sensor": {
					ObjectMeta: metav1.ObjectMeta{Name: "sensor"},
					Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{
							Name:  "sensor",
							Image: "quay.io/rhacs-eng/main:latest",
							Env:   []corev1.EnvVar{{Name: "ROX_CENTRAL_ENDPOINT", Value: "central.stackrox:443"}},
						}}},
					}},
				},
			},
			daemonSets: map[string]*appsv1.DaemonSet{
				"collector": {
					ObjectMeta: metav1.ObjectMeta{Name: "collector"},
					Spec: appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers:  []corev1.Container{{Name: "collector"}, {Name: "compliance"}},
							Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
						},
					}},
				},
			},
		},
		clusterName: "my-cluster",
		webhooks: []admissionv1.ValidatingWebhook{
			{Name: "check.stackrox.io", FailurePolicy: &ignore},
			{Name: "policyeval.stackrox.io", FailurePolicy: &ignore},
		},
	}
}

func TestSCDetect_Default(t *testing.T) {
	src := defaultSCSource()
	config, err := detectSecuredCluster(src)
	require.NoError(t, err)
	assert.Equal(t, "my-cluster", config.clusterName)
	assert.Equal(t, "central.stackrox:443", config.centralEndpoint)
	assert.False(t, config.enforcementDisabled)
	assert.False(t, config.failurePolicyFail)
	assert.False(t, config.collectionNone)
	assert.False(t, config.tolerationsDisabled)
	assert.False(t, config.customImages)
}

func TestSCDetect_CustomCentralEndpoint(t *testing.T) {
	src := defaultSCSource()
	src.deployments["sensor"].Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: "ROX_CENTRAL_ENDPOINT", Value: "my-central.example.com:443"},
	}
	config, err := detectSecuredCluster(src)
	require.NoError(t, err)
	assert.Equal(t, "my-central.example.com:443", config.centralEndpoint)
}

func TestSCDetect_EnforcementDisabled(t *testing.T) {
	src := defaultSCSource()
	src.webhooks = []admissionv1.ValidatingWebhook{{Name: "check.stackrox.io"}}
	config, err := detectSecuredCluster(src)
	require.NoError(t, err)
	assert.True(t, config.enforcementDisabled)
}

func TestSCDetect_FailurePolicyFail(t *testing.T) {
	src := defaultSCSource()
	fail := admissionv1.Fail
	src.webhooks = []admissionv1.ValidatingWebhook{
		{Name: "check.stackrox.io", FailurePolicy: &fail},
		{Name: "policyeval.stackrox.io", FailurePolicy: &fail},
	}
	config, err := detectSecuredCluster(src)
	require.NoError(t, err)
	assert.True(t, config.failurePolicyFail)
}

func TestSCDetect_CollectionNone(t *testing.T) {
	src := defaultSCSource()
	src.daemonSets["collector"].Spec.Template.Spec.Containers = []corev1.Container{{Name: "compliance"}}
	config, err := detectSecuredCluster(src)
	require.NoError(t, err)
	assert.True(t, config.collectionNone)
}

func TestSCDetect_TolerationsDisabled(t *testing.T) {
	src := defaultSCSource()
	src.daemonSets["collector"].Spec.Template.Spec.Tolerations = nil
	config, err := detectSecuredCluster(src)
	require.NoError(t, err)
	assert.True(t, config.tolerationsDisabled)
}

func TestSCDetect_CustomImages(t *testing.T) {
	src := defaultSCSource()
	src.deployments["sensor"].Spec.Template.Spec.Containers[0].Image = "my-registry.example.com/main:latest"
	config, err := detectSecuredCluster(src)
	require.NoError(t, err)
	assert.True(t, config.customImages)
}

func TestSCDetect_MissingClusterName(t *testing.T) {
	src := defaultSCSource()
	src.clusterName = ""
	_, err := detectSecuredCluster(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
