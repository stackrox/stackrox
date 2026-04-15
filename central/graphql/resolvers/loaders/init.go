package loaders

import (
	"reflect"

	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/v2/datastore"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	componentV2DataStore "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	imageV2DataStore "github.com/stackrox/rox/central/imagev2/datastore"
	imageDataStore "github.com/stackrox/rox/central/imagev2/datastore/mapper/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	nodeComponentDataStore "github.com/stackrox/rox/central/nodecomponent/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/generated/storage"
)

// Init registers all GraphQL type loaders.
// Called explicitly from central/app/app.go instead of package init().
func Init() {
	RegisterTypeFactory(reflect.TypeOf(storage.ClusterCVE{}), func() interface{} {
		return NewClusterCVELoader(clusterCVEDataStore.Singleton())
	})

	RegisterTypeFactory(componentV2LoaderType, func() interface{} {
		return NewComponentV2Loader(componentV2DataStore.Singleton())
	})

	RegisterTypeFactory(reflect.TypeOf(storage.Deployment{}), func() interface{} {
		return NewDeploymentLoader(deploymentDataStore.Singleton(), deploymentsView.Singleton())
	})

	RegisterTypeFactory(imageCveV2LoaderType, func() interface{} {
		return NewImageCVEV2Loader(imageCVEDataStore.Singleton())
	})

	RegisterTypeFactory(reflect.TypeOf(storage.Image{}), func() interface{} {
		return NewImageLoader(imageDataStore.Singleton(), imagesView.Singleton())
	})

	RegisterTypeFactory(reflect.TypeOf(storage.ImageV2{}), func() interface{} {
		return NewImageV2Loader(imageV2DataStore.Singleton(), imagesView.Singleton())
	})

	RegisterTypeFactory(reflect.TypeOf(storage.ListDeployment{}), func() interface{} {
		return NewListDeploymentLoader(deploymentDataStore.Singleton())
	})

	RegisterTypeFactory(reflect.TypeOf(storage.NamespaceMetadata{}), func() interface{} {
		return NewNamespaceLoader(namespaceDataStore.Singleton())
	})

	RegisterTypeFactory(reflect.TypeOf(storage.NodeComponent{}), func() interface{} {
		return NewNodeComponentLoader(nodeComponentDataStore.Singleton())
	})

	RegisterTypeFactory(reflect.TypeOf(storage.NodeCVE{}), func() interface{} {
		return NewNodeCVELoader(nodeCVEDataStore.Singleton())
	})

	RegisterTypeFactory(nodeLoaderType, func() interface{} {
		return NewNodeLoader(nodeDataStore.Singleton())
	})

	RegisterTypeFactory(reflect.TypeOf(storage.Policy{}), func() interface{} {
		return NewPolicyLoader(policyDataStore.Singleton())
	})
}
