package migratetooperator

import (
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func generateCR(config *detectedConfig) *platform.Central {
	cr := &platform.Central{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "Central",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "stackrox-central-services",
		},
	}

	switch config.Storage.Type {
	case storagePVC:
		cr.Spec.Central = &platform.CentralComponentSpec{
			DB: &platform.CentralDBSpec{
				Persistence: &platform.DBPersistence{
					PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
						ClaimName: pointers.String(config.Storage.PVCName),
					},
				},
			},
		}
	case storageHostPath:
		db := &platform.CentralDBSpec{
			Persistence: &platform.DBPersistence{
				HostPath: &platform.HostPathSpec{
					Path: pointers.String(config.Storage.HostPath),
				},
			},
		}
		if len(config.Storage.NodeSelector) > 0 {
			db.NodeSelector = config.Storage.NodeSelector
		}
		cr.Spec.Central = &platform.CentralComponentSpec{DB: db}
	}

	return cr
}

// marshalCR serializes only the spec portion of the Central CR, avoiding
// empty status fields that the operator types include.
func marshalCR(cr *platform.Central) ([]byte, error) {
	obj := struct {
		APIVersion string               `json:"apiVersion"`
		Kind       string               `json:"kind"`
		Metadata   map[string]string    `json:"metadata"`
		Spec       platform.CentralSpec `json:"spec"`
	}{
		APIVersion: cr.APIVersion,
		Kind:       cr.Kind,
		Metadata:   map[string]string{"name": cr.Name},
		Spec:       cr.Spec,
	}
	return yaml.Marshal(obj)
}
