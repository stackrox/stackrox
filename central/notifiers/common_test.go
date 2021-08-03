package notifiers

import (
	"context"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
)

func namespaceWithAnnotation(annotationKey, annotationValue string) *storage.NamespaceMetadata {
	ns := &storage.NamespaceMetadata{
		Id:          "id",
		Name:        "name",
		ClusterId:   "cluster-id",
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
	envIsolator := envisolator.NewEnvIsolator(t)

	alertWithNoClusterID := fixtures.GetResourceAlert()
	alertWithNoClusterID.GetResource().ClusterId = ""

	alertWithNoNamespace := fixtures.GetResourceAlert()
	alertWithNoNamespace.GetResource().Namespace = ""

	cases := []struct {
		name          string
		feature       bool
		namespace     []*storage.NamespaceMetadata
		annotationKey string
		alert         *storage.Alert
		expectedValue string
	}{
		{
			name:          "[Feature on] Get from deployment if it exists in both",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "deployValue",
		},
		{
			name:          "[Feature on] Get from deployment if it exists but not in namespace",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "deployValue",
		},
		{
			name:          "[Feature on] Get from deployment label if it exists",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentLabel("annotKey", "labelVal"),
			expectedValue: "labelVal",
		},
		{
			name:          "[Feature on] Get from namespace when not in deployment and exists in namespace",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "nsValue",
		},
		{
			name:          "[Feature on] Get from namespace for resource alert if it exists in namespace",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         fixtures.GetResourceAlert(),
			expectedValue: "nsValue",
		},
		{
			name:          "[Feature on] Get default when not in deployment or namespace",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "[Feature on] Get default when no cluster id available to lookup namespace",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithNoClusterID,
			expectedValue: "default",
		},
		{
			name:          "[Feature on] Get default when no namespace name available to lookup namespace",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithNoNamespace,
			expectedValue: "default",
		},
		{
			name:          "[Feature on] Get default when nil namespace found",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{nil},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "[Feature on] Get default when no namespaces found",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:    "[Feature on] Get default when multiple namespaces found",
			feature: true,
			namespace: []*storage.NamespaceMetadata{
				namespaceWithAnnotation("annotKey", "nsValue"),
				namespaceWithAnnotation("", ""),
			},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "[Feature on] Get default when key is incorrect",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("altKey", "altNsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("altKey", "altDeployValue"),
			expectedValue: "default",
		},
		{
			name:          "[Feature on] Get default if key is not provided",
			feature:       true,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "default",
		},
		// Tests with feature off. Will be removed once released
		{
			name:          "[Feature off] Get from deployment if it exists in both",
			feature:       false,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "deployValue",
		},
		{
			name:          "[Feature off] Get from deployment if it exists but not in namespace",
			feature:       false,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "deployValue",
		},
		{
			name:          "[Feature off] Get from deployment label if it exists",
			feature:       false,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentLabel("annotKey", "labelVal"),
			expectedValue: "labelVal",
		},
		{
			name:          "[Feature off] Get default when not in deployment and even if it exists in namespace",
			feature:       false,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "[Feature off] Get default for resource alert",
			feature:       false,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         fixtures.GetResourceAlert(),
			expectedValue: "default",
		},
		{
			name:          "[Feature off] Get default when not in deployment or namespace",
			feature:       false,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "[Feature off] Get default when key is incorrect",
			feature:       false,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("altKey", "altNsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("altKey", "altDeployValue"),
			expectedValue: "default",
		},
		{
			name:          "[Feature off] Get default if key is not provided",
			feature:       false,
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "default",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			envIsolator.Setenv(features.NamespaceAnnotationsForNotifiers.EnvVar(), strconv.FormatBool(c.feature))
			if c.feature && !features.NamespaceAnnotationsForNotifiers.Enabled() { // skip tests with feature on for release tests
				t.Skipf("%s feature flag not enabled, skipping...", features.NamespaceAnnotationsForNotifiers.Name())
			}

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			nsStore := namespaceMocks.NewMockDataStore(mockCtrl)

			nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(c.namespace, nil).AnyTimes()
			value := GetAnnotationValue(context.Background(), c.alert, c.annotationKey, "default", nsStore)

			assert.Equal(t, c.expectedValue, value)
		})
	}
}

func TestGetAnnotationValueCorrectlyQueriesForNamespace(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)
	envIsolator.Setenv(features.NamespaceAnnotationsForNotifiers.EnvVar(), "true")
	if !features.NamespaceAnnotationsForNotifiers.Enabled() {
		t.Skipf("%s feature flag not enabled, skipping...", features.NamespaceAnnotationsForNotifiers.Name())
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)

	alert := fixtures.GetAlert()
	ns := namespaceWithAnnotation("somekey", "somevalue")

	expectedQuery := search.NewQueryBuilder().AddExactMatches(search.Namespace, alert.GetDeployment().GetNamespace()).AddExactMatches(search.ClusterID, alert.GetDeployment().GetClusterId()).ProtoQuery()

	nsStore.EXPECT().SearchNamespaces(gomock.Any(), expectedQuery).Return([]*storage.NamespaceMetadata{ns}, nil)
	value := GetAnnotationValue(context.Background(), alert, "somekey", "default", nsStore)

	assert.Equal(t, "somevalue", value)
}

func TestGetAnnotationValueReturnsDefaultIfNoStoreReturnsError(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)
	envIsolator.Setenv(features.NamespaceAnnotationsForNotifiers.EnvVar(), "true")
	if !features.NamespaceAnnotationsForNotifiers.Enabled() {
		t.Skipf("%s feature flag not enabled, skipping...", features.NamespaceAnnotationsForNotifiers.Name())
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)

	alert := fixtures.GetAlert()

	nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(nil, errors.New("testttt"))
	value := GetAnnotationValue(context.Background(), alert, "somekey", "default", nsStore)

	assert.Equal(t, "default", value)
}
