package centralservices

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeutils "k8s.io/apimachinery/pkg/util/runtime"
)

const (
	defaultResourceTestValues = `
imagePullSecrets:
  allowNone: true
central:
  db:
    enabled: true
  persistence:
    none: true
`
)

type resourcesSuite struct {
	baseSuite
}

func TestResources(t *testing.T) {
	suite.Run(t, new(resourcesSuite))
}

var (
	scheme = runtime.NewScheme()
)

func init() {
	runtimeutils.Must(appsv1.AddToScheme(scheme))
	runtimeutils.Must(corev1.AddToScheme(scheme))
}

func (s *resourcesSuite) TestDefaultResources() {

	defaultValues := defaultResourceTestValues

	type tc struct {
		name              string
		workloadKind      string
		workloadName      string
		containerName     string
		isInitContainer   bool
		expectedResources corev1.ResourceRequirements
		values            []string
	}

	testCases := []tc{
		{
			name:          "default central resources",
			workloadKind:  "Deployment",
			workloadName:  "central",
			containerName: "central",
			values: []string{
				defaultValues,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4000m"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1500m"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
		},
		{
			name:          "default central-db resources",
			workloadKind:  "Deployment",
			workloadName:  "central-db",
			containerName: "central-db",
			values: []string{
				defaultValues,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("16Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
		},
		{
			name:          "default scanner resources",
			workloadKind:  "Deployment",
			workloadName:  "scanner",
			containerName: "scanner",
			values: []string{
				defaultValues,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1500Mi"),
				},
			},
		},
		{
			name:          "default scanner-db resources",
			workloadKind:  "Deployment",
			workloadName:  "scanner-db",
			containerName: "db",
			values: []string{
				defaultValues,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2000m"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
			},
		},
		{
			name:            "default init-db resources",
			workloadKind:    "Deployment",
			workloadName:    "scanner-db",
			containerName:   "init-db",
			isInitContainer: true,
			values: []string{
				defaultValues,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2000m"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
			},
		}, {
			name:          "overridden central resources",
			workloadKind:  "Deployment",
			workloadName:  "central",
			containerName: "central",
			values: []string{
				defaultValues,
				`
central:
  resources:
    limits:
      cpu: 100m
      memory: 100Mi
`,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		},
		{
			name:          "overridden central-db resources",
			workloadKind:  "Deployment",
			workloadName:  "central-db",
			containerName: "central-db",
			values: []string{
				defaultValues,
				`
central:
  db:
    resources:
      limits:
        cpu: 100m
        memory: 100Mi
`,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		},
		{
			name:          "overridden scanner resources",
			workloadKind:  "Deployment",
			workloadName:  "scanner",
			containerName: "scanner",
			values: []string{
				defaultValues,
				`
scanner:
  resources:
    limits:
      cpu: 100m
      memory: 100Mi
`,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		},
		{
			name:          "overridden scanner-db resources",
			workloadKind:  "Deployment",
			workloadName:  "scanner-db",
			containerName: "db",
			values: []string{
				defaultValues,
				`
scanner:
  dbResources:
    limits:
      cpu: 100m
      memory: 100Mi
`,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		},
		{
			name:            "overridden init-db resources",
			workloadKind:    "Deployment",
			workloadName:    "scanner-db",
			containerName:   "init-db",
			isInitContainer: true,
			values: []string{
				defaultValues,
				`
scanner:
  dbResources:
    limits:
      cpu: 100m
      memory: 100Mi
`,
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		},
	}

	for _, tc := range testCases {
		var tc = tc
		s.T().Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, m := s.LoadAndRender(tc.values...)
			require.NotEmpty(t, m)
			p := s.ParseObjects(m)
			objs := map[string]unstructured.Unstructured{}
			for _, obj := range p {
				objs[obj.GetKind()+"/"+obj.GetName()] = obj
			}

			var containers []corev1.Container
			if tc.workloadKind == "Deployment" {
				var deployment appsv1.Deployment
				require.NoError(t, mustFindDeployment(objs, tc.workloadKind+"/"+tc.workloadName, &deployment))
				if tc.isInitContainer {
					containers = deployment.Spec.Template.Spec.InitContainers
				} else {
					containers = deployment.Spec.Template.Spec.Containers
				}
			} else {
				t.Fatalf("unsupported workload kind %s", tc.workloadKind)
			}

			var container corev1.Container
			require.NoError(t, mustFindContainer(containers, tc.containerName, &container))
			require.Truef(t, assertResourceRequirementsEqual(t, tc.expectedResources, container.Resources), "expected %v, got %v", tc.expectedResources, container.Resources)

		})
	}

}

func assertResourceRequirementsEqual(t *testing.T, expected, actual corev1.ResourceRequirements) bool {
	return assertResourceListEqual(t, "limits", expected.Limits, actual.Limits) && assertResourceListEqual(t, "request", expected.Requests, actual.Requests)
}

func assertResourceListEqual(t *testing.T, typ string, expected, actual corev1.ResourceList) bool {
	// check if keys are equal
	for key := range expected {
		if _, ok := actual[key]; !ok {
			t.Errorf("expected key %s.%s not found in actual", typ, key)
			return false
		}
	}
	if len(expected) != len(actual) {
		t.Errorf("expected %d keys, got %d in %s", len(expected), len(actual), typ)
		return false
	}
	// check if values are equal
	for resourceName, expectedValue := range expected {
		actualValue := actual[resourceName]
		if !expectedValue.Equal(actualValue) {
			t.Errorf("expected value %v for key %s.%s, got %v", expectedValue, typ, resourceName, actualValue)
			return false
		}
	}
	return true
}

func mustFindDeployment(objs map[string]unstructured.Unstructured, name string, target *appsv1.Deployment) error {
	centralUnstructured, ok := objs[name]
	if !ok {
		return errors.Errorf("deployment %s not found", name)
	}
	var d appsv1.Deployment
	if err := scheme.Convert(&centralUnstructured, &d, nil); err != nil {
		return errors.Wrapf(err, "failed to convert %s to Deployment", name)
	}
	*target = d
	return nil
}

func mustFindContainer(containers []corev1.Container, name string, target *corev1.Container) error {
	for _, c := range containers {
		if c.Name == name {
			*target = c
			return nil
		}
	}
	return errors.Errorf("container %s not found", name)
}
