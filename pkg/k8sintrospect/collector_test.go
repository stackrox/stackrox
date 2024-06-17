package k8sintrospect

import (
	"testing"

	compv1alpha1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateFileName(t *testing.T) {
	testCases := map[string]struct {
		obj      k8sutil.Object
		expected string
	}{
		"test with deployment should use kubernetes app label": {
			obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sensor",
					Namespace: "stackrox",
					Labels: map[string]string{
						"app.kubernetes.io/name": "sensor",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "Deployment",
				},
			},
			expected: "stackrox/sensor/deployment-sensor.yaml",
		},
		"test with deployment should use simple app label": {
			obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sensor",
					Namespace: "another-namespace",
					Labels: map[string]string{
						"app": "sensor",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "Deployment",
				},
			},
			expected: "another-namespace/sensor/deployment-sensor.yaml",
		},
		"test with compliance operator objects": {
			obj: &compv1alpha1.Profile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ocp-cis",
					Namespace: "openshift-compliance",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Profile",
					APIVersion: complianceoperator.Profile.GroupVersionKind().GroupVersion().String(),
				},
			},
			expected: "openshift-compliance/Profile/profile-ocp-cis.yaml",
		},
		"test with _ungrouped object": {
			obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-deployment",
					Namespace: "openshift-compliance",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "Deployment",
				},
			},
			expected: "openshift-compliance/_ungrouped/deployment-some-deployment.yaml",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			fileName := generateFileName(testCase.obj, ".yaml")
			assert.Equal(t, testCase.expected, fileName)
		})
	}
}
