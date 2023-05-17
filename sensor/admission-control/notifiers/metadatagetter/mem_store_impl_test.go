package metadatagetter

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/sensor/admission-control/resources/deployments"
	"github.com/stackrox/rox/sensor/admission-control/resources/namespaces"
	"github.com/stackrox/rox/sensor/admission-control/resources/pods"
	"github.com/stretchr/testify/assert"
)

func namespaceWithAnnotation(annotationKey, annotationValue string) *storage.NamespaceMetadata {
	ns := &storage.NamespaceMetadata{
		Id:          fixtureconsts.Namespace1,
		Name:        "stackrox",
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "cluster-name",
	}

	if annotationKey != "" {
		ns.Annotations = map[string]string{
			annotationKey: annotationValue,
		}
	}

	return ns
}

func alertWithDeploymentAnnotation(annotationKey, annotationValue string) *storage.Alert {
	alert := fixtures.GetAlert()
	if annotationKey != "" {
		alert.GetDeployment().Annotations = map[string]string{
			annotationKey: annotationValue,
		}
	}

	return alert
}

func alertWithDeploymentLabel(labelKey, labelValue string) *storage.Alert {
	alert := fixtures.GetAlert()
	if labelKey != "" {
		alert.GetDeployment().Labels = map[string]string{
			labelKey: labelValue,
		}
	}

	return alert
}

func TestGetAnnotationValue(t *testing.T) {
	alertWithNoClusterID := fixtures.GetResourceAlert()
	alertWithNoClusterID.GetResource().ClusterId = ""

	alertWithNoNamespace := fixtures.GetResourceAlert()
	alertWithNoNamespace.GetResource().Namespace = ""

	cases := []struct {
		name          string
		nsMetadata    *storage.NamespaceMetadata
		annotationKey string
		alert         *storage.Alert
		expectedValue string
	}{
		{
			name:          "Get from deployment if it exists in both",
			nsMetadata:    namespaceWithAnnotation("annotKey", "nsValue"),
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "deployValue",
		},
		{
			name:          "Get from deployment if it exists but not in namespace",
			nsMetadata:    nil,
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "deployValue",
		},
		{
			name:          "Get from deployment label if it exists",
			nsMetadata:    nil,
			annotationKey: "annotKey",
			alert:         alertWithDeploymentLabel("annotKey", "labelVal"),
			expectedValue: "labelVal",
		},
		{
			name:          "Get from namespace when not in deployment and exists in namespace",
			nsMetadata:    namespaceWithAnnotation("annotKey", "nsValue"),
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "nsValue",
		},
		{
			name:          "Get from namespace for resource alert if it exists in namespace",
			nsMetadata:    namespaceWithAnnotation("annotKey", "nsValue"),
			annotationKey: "annotKey",
			alert:         fixtures.GetResourceAlert(),
			expectedValue: "nsValue",
		},
		{
			name:          "Get default when not in deployment or namespace",
			nsMetadata:    namespaceWithAnnotation("", ""),
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "Get default when no namespace name available to lookup namespace",
			nsMetadata:    namespaceWithAnnotation("annotKey", "nsValue"),
			annotationKey: "annotKey",
			alert:         alertWithNoNamespace,
			expectedValue: "default",
		},
		{
			name:          "Get default when nil namespace found",
			nsMetadata:    nil,
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "Get default when key is incorrect",
			nsMetadata:    namespaceWithAnnotation("altKey", "altNsValue"),
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("altKey", "altDeployValue"),
			expectedValue: "default",
		},
		{
			name:          "Get default if key is not provided",
			nsMetadata:    namespaceWithAnnotation("annotKey", "nsValue"),
			annotationKey: "",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "default",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			nsStore := namespaces.NewNamespaceStore(deployments.Singleton(), pods.Singleton())
			nsStore.AddNamespace(c.nsMetadata)
			metadataGetter := newMetadataGetter(nsStore)

			value := metadataGetter.GetAnnotationValue(context.Background(), c.alert, c.annotationKey, "default")

			assert.Equal(t, c.expectedValue, value)
		})
	}
}

func TestGetAnnotationValueCorrectlyLooksForNamespace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	namespace := namespaceWithAnnotation("somekey", "somevalue")
	nsStore := namespaces.NewNamespaceStore(deployments.Singleton(), pods.Singleton())
	nsStore.AddNamespace(namespace)
	metadataGetter := newMetadataGetter(nsStore)

	alert := fixtures.GetAlert()
	value := metadataGetter.GetAnnotationValue(context.Background(), alert, "somekey", "default")

	assert.Equal(t, "somevalue", value)
}

func TestGetAnnotationValueReturnsDefaultIfNoStoreReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	nsStore := namespaces.NewNamespaceStore(deployments.Singleton(), pods.Singleton())
	metadataGetter := newMetadataGetter(nsStore)

	alert := fixtures.GetAlert()

	value := metadataGetter.GetAnnotationValue(context.Background(), alert, "somekey", "default")

	assert.Equal(t, "default", value)
}
