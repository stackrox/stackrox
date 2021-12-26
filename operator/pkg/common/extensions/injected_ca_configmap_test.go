package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

func makeTestController(name string, uid types.UID) *unstructured.Unstructured {
	controller := &unstructured.Unstructured{}

	controller.SetName(name)
	controller.SetUID(uid)
	controller.SetAPIVersion("test")
	controller.SetNamespace("test")
	return controller
}

func TestConfigMapController(t *testing.T) {
	controller1 := makeTestController("test", "0000")
	controller2 := makeTestController("test", "0001")
	controller3 := makeTestController("test1", "0000")

	cm := makeConfigMap(controller1)
	assert.True(t, isAlreadyControlled(cm, controller1))
	assert.False(t, isAlreadyControlled(cm, controller2))
	assert.True(t, isAlreadyControlled(cm, controller3))

	addController(cm, controller2)
	assert.True(t, isAlreadyControlled(cm, controller1))
	assert.True(t, isAlreadyControlled(cm, controller2))
}
