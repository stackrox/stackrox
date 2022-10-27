package kubernetes

import (
	"testing"

	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	numGoroutines = 100
	numIterations = 1000
)

var (
	untrimmedAnnotations = map[string]string{
		"kubectl.kubernetes.io/last-applied-configuration": "last-config",
		"deployment.kubernetes.io/revision":                "this-revision",
		"someKey":                                          "short value",
		"someOtherKey":                                     "this is a really long annotation value that should be trimmed by the function we are testing. Since we are cutting of at a length of 256 and at the word boundary, our expectation for this test is that this annotation gets cut off after the following veryverylongword and this text should not appear any more",
	}

	expectedTrimmedAnnotations = map[string]string{
		"someKey":      "short value",
		"someOtherKey": "this is a really long annotation value that should be trimmed by the function we are testing. Since we are cutting of at a length of 256 and at the word boundary, our expectation for this test is that this annotation gets cut off after the following...",
	}
)

func TestTrimAnnotations(t *testing.T) {
	obj := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: maputil.ShallowClone(untrimmedAnnotations),
		},
	}

	TrimAnnotations(&obj)
	assert.Equal(t, expectedTrimmedAnnotations, obj.GetAnnotations())
}

// Tested with -race -count 20
func TestTrimAnnotationsRace(t *testing.T) {
	obj := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: maputil.ShallowClone(untrimmedAnnotations),
		},
	}

	TrimAnnotations(&obj)
	assert.Equal(t, expectedTrimmedAnnotations, obj.GetAnnotations())

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for i := 0; i < numIterations; i++ {
				TrimAnnotations(&obj)
				assert.Equal(t, expectedTrimmedAnnotations, obj.GetAnnotations())
			}
		}()
	}
}
