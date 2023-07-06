package metadatagetter

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func namespaceWithAnnotation(annotationKey, annotationValue string) *storage.NamespaceMetadata {
	ns := &storage.NamespaceMetadata{
		Id:          fixtureconsts.Namespace1,
		Name:        "name",
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

func namespaceWithLabels() *storage.NamespaceMetadata {
	return &storage.NamespaceMetadata{
		Id:          fixtureconsts.Namespace1,
		Name:        "name",
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "cluster-name",
		Labels: map[string]string{
			"x":                           "y",
			"abc":                         "xyz",
			"kubernetes.io/metadata.name": "name",
		},
	}
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
		namespace     []*storage.NamespaceMetadata
		annotationKey string
		alert         *storage.Alert
		expectedValue string
	}{
		{
			name:          "Get from deployment if it exists in both",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "deployValue",
		},
		{
			name:          "Get from deployment if it exists but not in namespace",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "deployValue",
		},
		{
			name:          "Get from deployment label if it exists",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentLabel("annotKey", "labelVal"),
			expectedValue: "labelVal",
		},
		{
			name:          "Get from namespace when not in deployment and exists in namespace",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "nsValue",
		},
		{
			name:          "Get from namespace for resource alert if it exists in namespace",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         fixtures.GetResourceAlert(),
			expectedValue: "nsValue",
		},
		{
			name:          "Get default when not in deployment or namespace",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("", "")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "Get default when no cluster id available to lookup namespace",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithNoClusterID,
			expectedValue: "default",
		},
		{
			name:          "Get default when no namespace name available to lookup namespace",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "annotKey",
			alert:         alertWithNoNamespace,
			expectedValue: "default",
		},
		{
			name:          "Get default when nil namespace found",
			namespace:     []*storage.NamespaceMetadata{nil},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "Get default when no namespaces found",
			namespace:     []*storage.NamespaceMetadata{},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name: "Get default when multiple namespaces found",
			namespace: []*storage.NamespaceMetadata{
				namespaceWithAnnotation("annotKey", "nsValue"),
				namespaceWithAnnotation("", ""),
			},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: "default",
		},
		{
			name:          "Get default when key is incorrect",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("altKey", "altNsValue")},
			annotationKey: "annotKey",
			alert:         alertWithDeploymentAnnotation("altKey", "altDeployValue"),
			expectedValue: "default",
		},
		{
			name:          "Get default if key is not provided",
			namespace:     []*storage.NamespaceMetadata{namespaceWithAnnotation("annotKey", "nsValue")},
			annotationKey: "",
			alert:         alertWithDeploymentAnnotation("annotKey", "deployValue"),
			expectedValue: "default",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
			metadataGetter := newTestMetadataGetter(t, nsStore)

			nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(c.namespace, nil).AnyTimes()
			value := metadataGetter.GetAnnotationValue(context.Background(), c.alert, c.annotationKey, "default")

			assert.Equal(t, c.expectedValue, value)
		})
	}
}

func TestGetNamespaceLabels(t *testing.T) {
	alertWithNoClusterID := fixtures.GetResourceAlert()
	alertWithNoClusterID.GetResource().ClusterId = ""

	alertWithNoNamespace := fixtures.GetResourceAlert()
	alertWithNoNamespace.GetResource().Namespace = ""

	cases := []struct {
		name          string
		namespace     []*storage.NamespaceMetadata
		alert         *storage.Alert
		expectedValue map[string]string
	}{
		{
			name:      "Get namespace labels from deployment alerts",
			namespace: []*storage.NamespaceMetadata{namespaceWithLabels()},
			alert:     fixtures.GetAlert(),
			expectedValue: map[string]string{
				"x":                           "y",
				"abc":                         "xyz",
				"kubernetes.io/metadata.name": "name",
			},
		},
		{
			name:      "Get namespace labels from resource alerts",
			namespace: []*storage.NamespaceMetadata{namespaceWithLabels()},
			alert:     fixtures.GetResourceAlert(),
			expectedValue: map[string]string{
				"x":                           "y",
				"abc":                         "xyz",
				"kubernetes.io/metadata.name": "name",
			},
		},
		{
			name:          "Get empty for image alerts",
			namespace:     []*storage.NamespaceMetadata{namespaceWithLabels()},
			alert:         fixtures.GetImageAlert(),
			expectedValue: map[string]string{},
		},
		{
			name:          "Get empty when no cluster id available to lookup namespace",
			namespace:     []*storage.NamespaceMetadata{namespaceWithLabels()},
			alert:         alertWithNoClusterID,
			expectedValue: map[string]string{},
		},
		{
			name:          "Get empty when no namespace name available to lookup namespace",
			namespace:     []*storage.NamespaceMetadata{namespaceWithLabels()},
			alert:         alertWithNoNamespace,
			expectedValue: map[string]string{},
		},
		{
			name:          "Get empty when nil namespace found",
			namespace:     []*storage.NamespaceMetadata{nil},
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: map[string]string{},
		},
		{
			name:          "Get empty when no namespaces found",
			namespace:     []*storage.NamespaceMetadata{},
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: map[string]string{},
		},
		{
			name:          "Get empty when multiple namespaces found",
			namespace:     []*storage.NamespaceMetadata{namespaceWithLabels(), namespaceWithLabels()},
			alert:         alertWithDeploymentAnnotation("", ""),
			expectedValue: map[string]string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
			metadataGetter := newTestMetadataGetter(t, nsStore)

			nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(c.namespace, nil).AnyTimes()
			value := metadataGetter.GetNamespaceLabels(context.Background(), c.alert)

			assert.Equal(t, c.expectedValue, value)
		})
	}
}

func TestGetAnnotationValueCorrectlyQueriesForNamespace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	metadataGetter := newTestMetadataGetter(t, nsStore)

	alert := fixtures.GetAlert()
	ns := namespaceWithAnnotation("somekey", "somevalue")

	expectedQuery := search.NewQueryBuilder().AddExactMatches(search.Namespace, alert.GetDeployment().GetNamespace()).AddExactMatches(search.ClusterID, alert.GetDeployment().GetClusterId()).ProtoQuery()

	nsStore.EXPECT().SearchNamespaces(gomock.Any(), expectedQuery).Return([]*storage.NamespaceMetadata{ns}, nil)
	value := metadataGetter.GetAnnotationValue(context.Background(), alert, "somekey", "default")

	assert.Equal(t, "somevalue", value)
}

func TestGetNamespaceLabelsCorrectlyQueriesForNamespace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	metadataGetter := newTestMetadataGetter(t, nsStore)

	alert := fixtures.GetAlert()
	ns := namespaceWithLabels()

	expectedQuery := search.NewQueryBuilder().AddExactMatches(search.Namespace, alert.GetDeployment().GetNamespace()).AddExactMatches(search.ClusterID, alert.GetDeployment().GetClusterId()).ProtoQuery()

	nsStore.EXPECT().SearchNamespaces(gomock.Any(), expectedQuery).Return([]*storage.NamespaceMetadata{ns}, nil)
	value := metadataGetter.GetNamespaceLabels(context.Background(), alert)

	assert.Equal(t, ns.GetLabels(), value)
}

func TestGetAnnotationValueReturnsDefaultIfStoreReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	metadataGetter := newTestMetadataGetter(t, nsStore)

	alert := fixtures.GetAlert()

	nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(nil, errors.New(fixtureconsts.Cluster1))
	value := metadataGetter.GetAnnotationValue(context.Background(), alert, "somekey", "default")

	assert.Equal(t, "default", value)
}

func TestGetNamespaceLabelsReturnsEmptyIfStoreReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	metadataGetter := newTestMetadataGetter(t, nsStore)

	alert := fixtures.GetAlert()

	nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(nil, errors.New(fixtureconsts.Cluster1))
	value := metadataGetter.GetNamespaceLabels(context.Background(), alert)

	assert.Empty(t, value)
}
