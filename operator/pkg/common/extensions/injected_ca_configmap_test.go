package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

func makeTestController(kind string, uid types.UID) *unstructured.Unstructured {
	controller := &unstructured.Unstructured{}

	controller.SetKind(kind)
	controller.SetUID(uid)
	controller.SetAPIVersion("test")
	controller.SetNamespace("test")
	return controller
}

func TestConfigMapController(t *testing.T) {
	scanner := makeTestController("Scanner", "0000")
	sensor := makeTestController("Sensor", "0001")
	central1 := makeTestController("Central", "0002")
	central2 := makeTestController("Central", "0003")

	cm := makeConfigMap(scanner)
	assert.False(t, takeControl(cm, scanner))

	assert.Equal(t, getController(cm).UID, scanner.GetUID())
	assert.NotEqual(t, getController(cm).UID, sensor.GetUID())

	assert.True(t, takeControl(cm, sensor))
	assert.NotEqual(t, getController(cm).UID, scanner.GetUID())
	assert.Equal(t, getController(cm).UID, sensor.GetUID())

	assert.True(t, takeControl(cm, central1))
	assert.NotEqual(t, getController(cm).UID, sensor.GetUID())
	assert.Equal(t, getController(cm).UID, central1.GetUID())

	assert.True(t, takeControl(cm, central2))
	assert.NotEqual(t, getController(cm).UID, central1.GetUID())
	assert.Equal(t, getController(cm).UID, central2.GetUID())
}
