package datagraph

import (
	"reflect"
	"testing"

	indexMocks "github.com/stackrox/rox/central/secret/index/mocks"
	storeMocks "github.com/stackrox/rox/central/secret/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestSecretDataGraph(t *testing.T) {
	suite.Run(t, new(SecretDataGraphTestSuite))
}

type SecretDataGraphTestSuite struct {
	suite.Suite

	mockStore   *storeMocks.Store
	mockIndexer *indexMocks.Indexer

	datagraph DataGraph
}

func (suite *SecretDataGraphTestSuite) SetupTest() {
	suite.mockStore = &storeMocks.Store{}
	suite.mockIndexer = &indexMocks.Indexer{}

	suite.datagraph = New(suite.mockStore, suite.mockIndexer)
}

func (suite *SecretDataGraphTestSuite) TestProcessCreateEvent() {
	embedded := &v1.EmbeddedSecret{
		Id:   "secretId",
		Name: "secretName",
		Path: "werethesecretis",
	}
	container := &v1.Container{
		Id: "containerId",
		Secrets: []*v1.EmbeddedSecret{
			embedded,
		},
	}
	deployment := &v1.Deployment{
		Id:          "deploymentId",
		Name:        "deploymentName",
		ClusterId:   "clusterId",
		ClusterName: "clusterName",
		Namespace:   "namespace",
		Containers: []*v1.Container{
			container,
		},
	}

	// Return relationships that match, so the upserted values should be the same.
	secret := toSecret(embedded)
	relationship := toRelationship(deployment, container, embedded)
	sar := &v1.SecretAndRelationship{
		Secret:       secret,
		Relationship: relationship,
	}
	suite.mockStore.On("GetRelationship", "secretId").Return(relationship, true, nil)

	suite.mockStore.On("UpsertSecret",
		mock.MatchedBy(func(s *v1.Secret) bool { return reflect.DeepEqual(s, secret) })).Return(nil)
	suite.mockStore.On("UpsertRelationship",
		mock.MatchedBy(func(r *v1.SecretRelationship) bool { return reflect.DeepEqual(r, relationship) })).Return(nil)

	suite.mockIndexer.On("SecretAndRelationship",
		mock.MatchedBy(func(s *v1.SecretAndRelationship) bool { return reflect.DeepEqual(s, sar) })).Return(nil)

	err := suite.datagraph.ProcessDeploymentEvent(v1.ResourceAction_CREATE_RESOURCE, deployment)
	suite.NoError(err)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockIndexer.AssertExpectations(suite.T())
}

func (suite *SecretDataGraphTestSuite) TestProcessRemoveEvent() {
	embedded := &v1.EmbeddedSecret{
		Id:   "secretId",
		Name: "secretName",
		Path: "werethesecretis",
	}
	container := &v1.Container{
		Id: "containerId",
		Secrets: []*v1.EmbeddedSecret{
			embedded,
		},
	}
	deployment := &v1.Deployment{
		Id:          "deploymentId",
		Name:        "deploymentName",
		ClusterId:   "clusterId",
		ClusterName: "clusterName",
		Namespace:   "namespace",
		Containers: []*v1.Container{
			container,
		},
	}

	// Return relationships that match the removed relationship.
	secret := toSecret(embedded)
	relationship := toRelationship(deployment, container, embedded)

	// The relationship that will be upserted will be void of relationships.
	emptyRelationship := toRelationship(deployment, container, embedded)
	emptyRelationship.ContainerRelationships = []*v1.SecretContainerRelationship{}
	emptyRelationship.DeploymentRelationships = []*v1.SecretDeploymentRelationship{}
	sar := &v1.SecretAndRelationship{
		Secret:       secret,
		Relationship: emptyRelationship,
	}
	suite.mockStore.On("GetRelationship", "secretId").Return(relationship, true, nil)

	suite.mockStore.On("UpsertSecret",
		mock.MatchedBy(func(s *v1.Secret) bool { return reflect.DeepEqual(s, secret) })).Return(nil)
	suite.mockStore.On("UpsertRelationship",
		mock.MatchedBy(func(r *v1.SecretRelationship) bool { return reflect.DeepEqual(r, emptyRelationship) })).Return(nil)

	suite.mockIndexer.On("SecretAndRelationship",
		mock.MatchedBy(func(s *v1.SecretAndRelationship) bool { return reflect.DeepEqual(s, sar) })).Return(nil)

	err := suite.datagraph.ProcessDeploymentEvent(v1.ResourceAction_REMOVE_RESOURCE, deployment)
	suite.NoError(err)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockIndexer.AssertExpectations(suite.T())
}

func (suite *SecretDataGraphTestSuite) TestCreateRelationship() {
	deployment := &v1.Deployment{
		Id:          "deploymentId",
		Name:        "deploymentName",
		ClusterId:   "clusterId",
		ClusterName: "clusterName",
		Namespace:   "namespace",
	}
	container := &v1.Container{
		Id: "containerId",
	}
	embedded := &v1.EmbeddedSecret{
		Id:   "secretId",
		Name: "secretName",
		Path: "werethesecretis",
	}

	relationship := toRelationship(deployment, container, embedded)

	// Build relationships.
	suite.Equal(embedded.GetId(), relationship.GetId())

	suite.Equal(deployment.GetClusterId(), relationship.GetClusterRelationship().GetId())
	suite.Equal(deployment.GetClusterName(), relationship.GetClusterRelationship().GetName())

	suite.Equal(deployment.GetNamespace(), relationship.GetNamespaceRelationship().GetNamespace())

	suite.Equal(1, len(relationship.GetContainerRelationships()))
	suite.Equal(container.GetId(), relationship.GetContainerRelationships()[0].GetId())
	suite.Equal(embedded.GetPath(), relationship.GetContainerRelationships()[0].GetPath())

	suite.Equal(1, len(relationship.GetDeploymentRelationships()))
	suite.Equal(deployment.GetId(), relationship.GetDeploymentRelationships()[0].GetId())
	suite.Equal(deployment.GetName(), relationship.GetDeploymentRelationships()[0].GetName())
}

func (suite *SecretDataGraphTestSuite) TestCreateSecret() {
	embedded := &v1.EmbeddedSecret{
		Id:   "secretId",
		Name: "secretName",
		Path: "werethesecretis",
	}

	secret := toSecret(embedded)

	// Build relationships.
	suite.Equal(embedded.GetId(), secret.GetId())
	suite.Equal(embedded.GetName(), secret.GetName())
}

func (suite *SecretDataGraphTestSuite) TestStapleToRelationshipAtFront() {
	stapleTo := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	stapleFrom := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
		},
	}

	expectedRelationship := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	stapleRelationships(stapleTo, stapleFrom)
	suite.Equal(expectedRelationship, stapleTo)
}

func (suite *SecretDataGraphTestSuite) TestStapleToRelationshipInMiddle() {
	stapleTo := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId0",
				Path: "containerPath",
			},
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId0",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	stapleFrom := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
		},
	}

	expectedRelationship := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId0",
				Path: "containerPath",
			},
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId0",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	stapleRelationships(stapleTo, stapleFrom)
	suite.Equal(expectedRelationship, stapleTo)
}

func (suite *SecretDataGraphTestSuite) TestStapleToRelationshipAtEnd() {
	stapleTo := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId0",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId0",
				Name: "deploymentName",
			},
		},
	}

	stapleFrom := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
		},
	}

	expectedRelationship := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId0",
				Path: "containerPath",
			},
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId0",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
		},
	}

	stapleRelationships(stapleTo, stapleFrom)
	suite.Equal(expectedRelationship, stapleTo)
}

func (suite *SecretDataGraphTestSuite) TestRemoveFromRelationshipAtFront() {
	removeFrom := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	toRemove := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
		},
	}

	expectedRelationship := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	removeRelationships(removeFrom, toRemove)
	suite.Equal(expectedRelationship, removeFrom)
}

func (suite *SecretDataGraphTestSuite) TestRemoveFromRelationshipInMiddle() {
	removeFrom := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId0",
				Path: "containerPath",
			},
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId0",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	toRemove := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
		},
	}

	expectedRelationship := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId0",
				Path: "containerPath",
			},
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId0",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	removeRelationships(removeFrom, toRemove)
	suite.Equal(expectedRelationship, removeFrom)
}

func (suite *SecretDataGraphTestSuite) TestRemoveFromRelationshipAtEnd() {
	removeFrom := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	toRemove := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId2",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId2",
				Name: "deploymentName",
			},
		},
	}

	expectedRelationship := &v1.SecretRelationship{
		Id: "id1",
		ClusterRelationship: &v1.SecretClusterRelationship{
			Id:   "clusterID",
			Name: "clusterName",
		},
		NamespaceRelationship: &v1.SecretNamespaceRelationship{
			Namespace: "namespace",
		},
		ContainerRelationships: []*v1.SecretContainerRelationship{
			{
				Id:   "containerId1",
				Path: "containerPath",
			},
		},
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "deploymentId1",
				Name: "deploymentName",
			},
		},
	}

	removeRelationships(removeFrom, toRemove)
	suite.Equal(expectedRelationship, removeFrom)
}
