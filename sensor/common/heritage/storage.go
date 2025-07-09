package heritage

import (
	"encoding/json"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pastSensorDataToConfigMap(data ...*SensorMetadata) (*v1.ConfigMap, error) {
	if data == nil {
		return nil, nil
	}
	dataMap := make(map[string]string, len(data))
	byteEntry, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrapf(err, "marshalling data for %v", data)
	}
	dataMap[configMapKey] = string(byteEntry)

	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cmName,
			Annotations: map[string]string{
				annotationInfoKey: annotationInfoText,
			},
			Labels: map[string]string{
				// Tells the cluster upgrader to "own" this resource.
				// See `UpgradeResourceLabelKey` in `sensor/upgrader/common/consts.go`.
				// This is used in `sensor/openshift/delete-sensor.sh`.
				"auto-upgrade.stackrox.io/component": "sensor",
				// Tells the cluster upgrader to preserve this resource during upgrades.
				// See: `PreserveResourcesAnnotationKey` in `sensor/upgrader/common/consts.go`.
				"auto-upgrade.stackrox.io/preserve-resources": "true",
				"app.kubernetes.io/managed-by":                "sensor",
				"app.kubernetes.io/created-by":                "sensor",
				"app.kubernetes.io/name":                      "sensor",
			},
		},
		Data: dataMap,
	}, nil
}

func configMapToPastSensorData(cm *v1.ConfigMap) ([]*SensorMetadata, error) {
	if cm == nil {
		return nil, nil
	}
	data := make([]*SensorMetadata, 0, len(cm.Data))
	for key, jsonStr := range cm.Data {
		if key != configMapKey {
			continue
		}
		var entries []SensorMetadata
		if err := json.Unmarshal([]byte(jsonStr), &entries); err != nil {
			return nil, errors.Wrapf(err, "unmarshalling data %v", jsonStr)
		}
		for _, entry := range entries {
			data = append(data, &entry)
		}
	}
	return data, nil
}
