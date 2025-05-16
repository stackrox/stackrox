package reflectutils

import (
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type SomeStruct struct {
	I int
	S SubStruct
}

type SubStruct struct {
	S string
	P *SubStruct
}

func TestDeepMergeStructs(t *testing.T) {
	for name, testCase := range map[string]struct {
		a        interface{}
		b        interface{}
		expected interface{}
	}{
		"both empty": {
			a:        SomeStruct{},
			b:        SomeStruct{},
			expected: SomeStruct{},
		},
		"a empty": {
			a:        SomeStruct{},
			b:        SomeStruct{I: 1, S: SubStruct{S: "test"}},
			expected: SomeStruct{I: 1, S: SubStruct{S: "test"}},
		},
		"b empty": {
			a:        SomeStruct{I: 1, S: SubStruct{S: "test"}},
			b:        SomeStruct{},
			expected: SomeStruct{I: 1, S: SubStruct{S: "test"}},
		},
		"both non-empty": {
			a:        SomeStruct{I: 1},
			b:        SomeStruct{S: SubStruct{S: "test"}},
			expected: SomeStruct{I: 1, S: SubStruct{S: "test"}},
		},
		"preserves indirection": {
			a:        &SomeStruct{I: 1},
			b:        &SomeStruct{S: SubStruct{S: "test"}},
			expected: &SomeStruct{I: 1, S: SubStruct{S: "test"}},
		},
		"nested overwrite with b": {
			a:        SomeStruct{S: SubStruct{S: "from a"}},
			b:        SomeStruct{S: SubStruct{S: "from b"}},
			expected: SomeStruct{S: SubStruct{S: "from b"}},
		},
		"nil pointer only in a": {
			a:        SomeStruct{S: SubStruct{S: "from a", P: &SubStruct{S: "inner"}}},
			b:        SomeStruct{S: SubStruct{S: "from b"}},
			expected: SomeStruct{S: SubStruct{S: "from b", P: &SubStruct{S: "inner"}}},
		},
		"nil pointer only in b": {
			a:        SomeStruct{S: SubStruct{S: "from a"}},
			b:        SomeStruct{S: SubStruct{S: "from b", P: &SubStruct{S: "inner"}}},
			expected: SomeStruct{S: SubStruct{S: "from b", P: &SubStruct{S: "inner"}}},
		},
		"scanner V4 use-case": {
			a: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
			b: platform.CentralSpec{
				Central: &platform.CentralComponentSpec{
					AdminPasswordSecret: &platform.LocalSecretReference{
						Name: "foo",
					},
				},
				ScannerV4: &platform.ScannerV4Spec{
					Indexer: &platform.ScannerV4Component{
						DeploymentSpec: platform.DeploymentSpec{
							Resources: &corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("90"),
									corev1.ResourceMemory: resource.MustParse("100"),
								},
							},
						},
					},
				},
			},
			expected: platform.CentralSpec{
				Central: &platform.CentralComponentSpec{
					AdminPasswordSecret: &platform.LocalSecretReference{
						Name: "foo",
					},
				},
				ScannerV4: &platform.ScannerV4Spec{
					Indexer: &platform.ScannerV4Component{
						DeploymentSpec: platform.DeploymentSpec{
							Resources: &corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("90"),
									corev1.ResourceMemory: resource.MustParse("100"),
								},
							},
						},
					},
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			merged := DeepMergeStructs(testCase.a, testCase.b)
			assert.Equal(t, testCase.expected, merged)
		})
	}
}
