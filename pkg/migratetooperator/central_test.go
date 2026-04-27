package migratetooperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type fakeSource struct {
	deployments map[string]*appsv1.Deployment
	daemonSets  map[string]*appsv1.DaemonSet
	services    map[string]*corev1.Service
	secrets     map[string]*corev1.Secret
	routes      map[string]bool
}

func (f *fakeSource) Deployment(name string) (*appsv1.Deployment, error) {
	return f.deployments[name], nil
}
func (f *fakeSource) DaemonSet(name string) (*appsv1.DaemonSet, error) {
	return f.daemonSets[name], nil
}
func (f *fakeSource) Service(name string) (*corev1.Service, error) { return f.services[name], nil }

func (f *fakeSource) Secret(name string) (*corev1.Secret, error) {
	if f.secrets != nil {
		return f.secrets[name], nil
	}
	return nil, nil
}

func (f *fakeSource) Route(name string) (*unstructured.Unstructured, error) {
	if f.routes != nil && f.routes[name] {
		return &unstructured.Unstructured{}, nil
	}
	return nil, nil
}

func (f *fakeSource) ValidatingWebhookConfiguration(_ string) (*admissionv1.ValidatingWebhookConfiguration, error) {
	return nil, nil
}

func defaultCentralSource() *fakeSource {
	return &fakeSource{
		deployments: map[string]*appsv1.Deployment{
			"central": {
				ObjectMeta: metav1.ObjectMeta{Name: "central"},
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: []corev1.Container{{
						Name:  "central",
						Image: "quay.io/rhacs-eng/main:latest",
						Env: []corev1.EnvVar{
							{Name: "ROX_ENABLE_OPENSHIFT_AUTH", Value: "true"},
							{Name: "ROX_ENABLE_SECURE_METRICS", Value: "true"},
						},
					}}},
				}},
			},
			"central-db": {
				ObjectMeta: metav1.ObjectMeta{Name: "central-db"},
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{{
							Name:         "disk",
							VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "central-db"}},
						}},
					},
				}},
			},
		},
	}
}

func TestCentral_PVCDefault(t *testing.T) {
	cr, _, err := TransformToCentral(defaultCentralSource())
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Central.DB.Persistence.PersistentVolumeClaim)
	assert.Equal(t, "central-db", *cr.Spec.Central.DB.Persistence.PersistentVolumeClaim.ClaimName)
	assert.Nil(t, cr.Spec.Central.DB.Persistence.HostPath)
}

func TestCentral_PVCCustomName(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central-db"].Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName = "my-pvc"
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	assert.Equal(t, "my-pvc", *cr.Spec.Central.DB.Persistence.PersistentVolumeClaim.ClaimName)
}

func TestCentral_HostPath(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central-db"].Spec.Template.Spec.Volumes = []corev1.Volume{{
		Name:         "disk",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data/stackrox"}},
	}}
	src.deployments["central-db"].Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": "w1"}
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Central.DB.Persistence.HostPath)
	assert.Equal(t, "/data/stackrox", *cr.Spec.Central.DB.Persistence.HostPath.Path)
	assert.Equal(t, "w1", cr.Spec.Central.DB.NodeSelector["kubernetes.io/hostname"])
}

func TestCentral_MonitoringDefault(t *testing.T) {
	cr, _, err := TransformToCentral(defaultCentralSource())
	require.NoError(t, err)
	assert.Nil(t, cr.Spec.Monitoring)
}

func TestCentral_MonitoringDisabled(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central"].Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: "ROX_ENABLE_OPENSHIFT_AUTH", Value: "true"},
	}
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Monitoring)
	assert.False(t, *cr.Spec.Monitoring.OpenShiftMonitoring.Enabled)
}

func TestCentral_K8sOmitsMonitoring(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central"].Spec.Template.Spec.Containers[0].Env = nil
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	assert.Nil(t, cr.Spec.Monitoring)
}

func TestCentral_ExposureLB(t *testing.T) {
	src := defaultCentralSource()
	src.services = map[string]*corev1.Service{
		"central-loadbalancer": {Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer}},
	}
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Central.Exposure)
	assert.True(t, *cr.Spec.Central.Exposure.LoadBalancer.Enabled)
}

func TestCentral_ExposureRoute(t *testing.T) {
	src := defaultCentralSource()
	src.routes = map[string]bool{"central": true}
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Central.Exposure)
	assert.True(t, *cr.Spec.Central.Exposure.Route.Enabled)
}

func TestCentral_ExposureNone(t *testing.T) {
	cr, _, err := TransformToCentral(defaultCentralSource())
	require.NoError(t, err)
	assert.Nil(t, cr.Spec.Central.Exposure)
}

func TestCentral_DefaultTLSSecret(t *testing.T) {
	src := defaultCentralSource()
	src.secrets = map[string]*corev1.Secret{
		"central-default-tls-cert": {ObjectMeta: metav1.ObjectMeta{Name: "central-default-tls-cert"}},
	}
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Central.DefaultTLSSecret)
	assert.Equal(t, "central-default-tls-cert", cr.Spec.Central.DefaultTLSSecret.Name)
}

func TestCentral_TelemetryDisabled(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central"].Spec.Template.Spec.Containers[0].Env = append(
		src.deployments["central"].Spec.Template.Spec.Containers[0].Env,
		corev1.EnvVar{Name: "ROX_TELEMETRY_STORAGE_KEY_V1", Value: "DISABLED"},
	)
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Central.Telemetry)
	assert.False(t, *cr.Spec.Central.Telemetry.Enabled)
}

func TestCentral_Offline(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central"].Spec.Template.Spec.Containers[0].Env = append(
		src.deployments["central"].Spec.Template.Spec.Containers[0].Env,
		corev1.EnvVar{Name: "ROX_OFFLINE_MODE", Value: "true"},
	)
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Egress)
}

func TestCentral_CustomImages(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central"].Spec.Template.Spec.Containers[0].Image = "my-reg.example.com/main:v1"
	_, warnings, err := TransformToCentral(src)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "RELATED_IMAGE")
}

func TestCentral_PlaintextEndpoints(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central"].Spec.Template.Spec.Containers[0].Env = append(
		src.deployments["central"].Spec.Template.Spec.Containers[0].Env,
		corev1.EnvVar{Name: "ROX_PLAINTEXT_ENDPOINTS", Value: "8080"},
	)
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Customize)
	require.Len(t, cr.Spec.Customize.EnvVars, 1)
	assert.Equal(t, "8080", cr.Spec.Customize.EnvVars[0].Value)
}

func TestCentral_DeclarativeConfig(t *testing.T) {
	src := defaultCentralSource()
	src.deployments["central"].Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{Name: "my-cm", MountPath: "/run/stackrox.io/declarative-configuration/my-cm"},
	}
	src.deployments["central"].Spec.Template.Spec.Volumes = append(
		src.deployments["central"].Spec.Template.Spec.Volumes,
		corev1.Volume{Name: "my-cm", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "my-cm"}}}},
	)
	cr, _, err := TransformToCentral(src)
	require.NoError(t, err)
	require.NotNil(t, cr.Spec.Central.DeclarativeConfiguration)
	require.Len(t, cr.Spec.Central.DeclarativeConfiguration.ConfigMaps, 1)
	assert.Equal(t, "my-cm", cr.Spec.Central.DeclarativeConfiguration.ConfigMaps[0].Name)
}

func TestCentral_MissingCentralDB(t *testing.T) {
	src := defaultCentralSource()
	delete(src.deployments, "central-db")
	_, _, err := TransformToCentral(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCentral_MissingCentral(t *testing.T) {
	src := defaultCentralSource()
	delete(src.deployments, "central")
	_, _, err := TransformToCentral(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCentral_Metadata(t *testing.T) {
	cr, _, err := TransformToCentral(defaultCentralSource())
	require.NoError(t, err)
	assert.Equal(t, "platform.stackrox.io/v1alpha1", cr.APIVersion)
	assert.Equal(t, "Central", cr.Kind)
	assert.Equal(t, "stackrox-central-services", cr.Name)
}
