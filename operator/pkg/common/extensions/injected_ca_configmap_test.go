package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func makeTestController(name string, uid types.UID) *unstructured.Unstructured {
	controller := &unstructured.Unstructured{}
	controller.SetKind("Kind")
	controller.SetName(name)
	controller.SetUID(uid)
	controller.SetAPIVersion("test")
	controller.SetNamespace("test")
	return controller
}

func Test_makeConfigMap(t *testing.T) {
	configMap := makeConfigMap(&types.NamespacedName{Name: "name", Namespace: "test"})
	obj := makeTestController("controller", "0000")
	assert.Nil(t, controllerutil.SetControllerReference(obj, configMap, nil))
	assert.True(t, metav1.IsControlledBy(configMap, obj))
	assert.Equal(t, "true", configMap.Labels["config.openshift.io/inject-trusted-cabundle"])
}
