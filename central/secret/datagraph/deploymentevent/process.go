package index

import (
	"bitbucket.org/stack-rox/apollo/central/secret/index"
	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/boltdb/bolt"
)

// ProcessDeploymentEvent reads all of a deployment events deployed secrets, adding any new secrets to the storage, as
// we as adding or removing any new relationships, depending on the action of the event.
func ProcessDeploymentEvent(db *bolt.DB, gIndex bleve.Index, event *v1.DeploymentEvent) error {
	deployment := event.GetDeployment()
	deploymentID := deployment.GetId()
	deploymentName := deployment.GetName()
	clusterID := deployment.GetClusterId()
	clusterName := deployment.GetClusterName()
	namespace := deployment.GetNamespace()

	storage := store.New(db)
	for _, container := range deployment.GetContainers() {
		for _, embeddedSecret := range container.GetSecrets() {
			// Build relationships.
			relationship := new(v1.SecretRelationship)
			relationship.Id = embeddedSecret.GetId()

			clusterRelationship := new(v1.SecretClusterRelationship)
			clusterRelationship.Id = clusterID
			clusterRelationship.Name = clusterName
			relationship.ClusterRelationship = clusterRelationship

			namespaceRelationship := new(v1.SecretNamespaceRelationship)
			namespaceRelationship.Namespace = namespace
			relationship.NamespaceRelationship = namespaceRelationship

			containerRelationship := new(v1.SecretContainerRelationship)
			containerRelationship.Id = container.GetId()
			containerRelationship.Path = embeddedSecret.GetPath()
			relationship.ContainerRelationships = append(relationship.ContainerRelationships, containerRelationship)

			deploymentRelationship := new(v1.SecretDeploymentRelationship)
			deploymentRelationship.Id = deploymentID
			deploymentRelationship.Name = deploymentName
			relationship.DeploymentRelationships = append(relationship.DeploymentRelationships, deploymentRelationship)

			// Merge relationship
			var err error
			if event.Action == v1.ResourceAction_REMOVE_RESOURCE {
				relationship, err = removeRelationship(storage, relationship)
			} else {
				relationship, err = addRelationship(storage, relationship)
			}
			if err != nil {
				return err
			}

			// Build secret.
			secret := new(v1.Secret)
			secret.Id = embeddedSecret.GetId()
			secret.Name = embeddedSecret.GetName()

			// Store new secret and relationship, then index them.
			if err = storage.UpsertSecret(secret); err != nil {
				return err
			}
			if err = storage.UpsertRelationship(relationship); err != nil {
				return err
			}

			if err = index.SecretAndRelationship(gIndex, &v1.SecretAndRelationship{
				Secret:       secret,
				Relationship: relationship,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
