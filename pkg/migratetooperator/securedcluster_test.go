package migratetooperator

import (
	"os"
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type scFakeSource struct {
	fakeSource
	clusterName        string
	clusterNamePresent bool
	clusterNameInData  bool
	webhooks           []admissionv1.ValidatingWebhook
}

func (f *scFakeSource) Secret(name string) (*corev1.Secret, error) {
	if name != "helm-effective-cluster-name" || (!f.clusterNamePresent && f.clusterName == "") {
		return nil, nil
	}
	s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if f.clusterNameInData {
		s.Data = map[string][]byte{"cluster-name": []byte(f.clusterName)}
	} else {
		s.StringData = map[string]string{"cluster-name": f.clusterName}
	}
	return s, nil
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
		clusterName:        "my-cluster",
		clusterNamePresent: true,
		webhooks: []admissionv1.ValidatingWebhook{
			{Name: "check.stackrox.io", FailurePolicy: &ignore},
			{Name: "policyeval.stackrox.io", FailurePolicy: &ignore},
		},
	}
}

func TestSC_Default(t *testing.T) {
	cr, warnings, err := TransformToSecuredCluster(defaultSCSource())
	require.NoError(t, err)
	assert.Empty(t, warnings)

	golden, err := os.ReadFile("testdata/securedcluster_default.yaml")
	require.NoError(t, err)

	var expected platform.SecuredCluster
	require.NoError(t, yaml.Unmarshal(golden, &expected))

	assert.Equal(t, expected, *cr)
}

func TestSC_CustomCentralEndpoint(t *testing.T) {
	src := defaultSCSource()
	src.deployments["sensor"].Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: "ROX_CENTRAL_ENDPOINT", Value: "my-central.example.com:443"},
	}
	cr, _, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.CentralEndpoint)
	assert.Equal(t, "my-central.example.com:443", *cr.Spec.CentralEndpoint)
}

func TestSC_EnforcementDisabled(t *testing.T) {
	src := defaultSCSource()
	src.webhooks = []admissionv1.ValidatingWebhook{{Name: "check.stackrox.io"}}
	cr, _, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.AdmissionControl)
	require.NotNil(t, cr.Spec.AdmissionControl.Enforcement)
	assert.Equal(t, platform.PolicyEnforcementDisabled, *cr.Spec.AdmissionControl.Enforcement)
}

func TestSC_NoVWCDoesNotSetEnforcement(t *testing.T) {
	src := defaultSCSource()
	src.webhooks = nil
	cr, _, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	assert.Nil(t, cr.Spec.AdmissionControl)
}

func TestSC_FailurePolicyFail(t *testing.T) {
	src := defaultSCSource()
	fail := admissionv1.Fail
	src.webhooks = []admissionv1.ValidatingWebhook{
		{Name: "check.stackrox.io", FailurePolicy: &fail},
		{Name: "policyeval.stackrox.io", FailurePolicy: &fail},
	}
	cr, _, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.AdmissionControl)
	require.NotNil(t, cr.Spec.AdmissionControl.FailurePolicy)
	assert.Equal(t, platform.FailurePolicyFail, *cr.Spec.AdmissionControl.FailurePolicy)
}

func TestSC_CollectionNone(t *testing.T) {
	src := defaultSCSource()
	src.daemonSets["collector"].Spec.Template.Spec.Containers = []corev1.Container{{Name: "compliance"}}
	cr, _, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.PerNode)
	require.NotNil(t, cr.Spec.PerNode.Collector)
	assert.Equal(t, platform.CollectionNone, *cr.Spec.PerNode.Collector.Collection)
}

func TestSC_TolerationsDisabled(t *testing.T) {
	src := defaultSCSource()
	src.daemonSets["collector"].Spec.Template.Spec.Tolerations = nil
	cr, _, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.PerNode)
	assert.Equal(t, platform.TaintAvoid, *cr.Spec.PerNode.TaintToleration)
}

func TestSC_CustomImages(t *testing.T) {
	src := defaultSCSource()
	src.deployments["sensor"].Spec.Template.Spec.Containers[0].Image = "my-reg.example.com/main:latest"
	_, warnings, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "RELATED_IMAGE")
}

func TestSC_MissingClusterName(t *testing.T) {
	src := defaultSCSource()
	src.clusterName = ""
	src.clusterNamePresent = false
	_, _, err := TransformToSecuredCluster(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSC_EmptyClusterNameInSecret(t *testing.T) {
	src := defaultSCSource()
	src.clusterName = ""
	src.clusterNamePresent = true
	_, _, err := TransformToSecuredCluster(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name is empty")
}

func TestSC_ClusterNameFromData(t *testing.T) {
	src := defaultSCSource()
	src.clusterNameInData = true
	cr, _, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	assert.Equal(t, "my-cluster", *cr.Spec.ClusterName)
}
