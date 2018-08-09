package datagraph

import (
	"sort"

	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// serviceImpl provides APIs for alerts.
type datagraphImpl struct {
	storage store.Store
	indexer index.Indexer
}

// ProcessDeploymentEvent updates the secrets service with a new deployment event.
func (s *datagraphImpl) ProcessDeploymentEvent(action v1.ResourceAction, deployment *v1.Deployment) error {
	for _, container := range deployment.GetContainers() {
		for _, embeddedSecret := range container.GetSecrets() {
			// Merge relationship
			relationship := toRelationship(deployment, container, embeddedSecret)
			oldRelationship, exists, err := s.storage.GetRelationship(relationship.GetId())
			if err != nil {
				return err
			}
			if action == v1.ResourceAction_REMOVE_RESOURCE {
				if exists {
					removeRelationships(oldRelationship, relationship)
				} else {
					oldRelationship = new(v1.SecretRelationship)
				}
			} else {
				if exists {
					stapleRelationships(oldRelationship, relationship)
				} else {
					oldRelationship = relationship
				}
			}
			if err = s.storage.UpsertRelationship(oldRelationship); err != nil {
				return err
			}

			// Store new secret information.
			secret := toSecret(embeddedSecret)
			if err = s.storage.UpsertSecret(secret); err != nil {
				return err
			}

			// Index the secret and relationship together.
			if err = s.indexer.SecretAndRelationship(&v1.SecretAndRelationship{
				Secret:       secret,
				Relationship: oldRelationship,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func toRelationship(deployment *v1.Deployment, container *v1.Container, embedded *v1.EmbeddedSecret) *v1.SecretRelationship {
	// Build relationships.
	relationship := new(v1.SecretRelationship)
	relationship.Id = embedded.GetId()

	clusterRelationship := new(v1.SecretClusterRelationship)
	clusterRelationship.Id = deployment.GetClusterId()
	clusterRelationship.Name = deployment.GetClusterName()
	relationship.ClusterRelationship = clusterRelationship

	namespaceRelationship := new(v1.SecretNamespaceRelationship)
	namespaceRelationship.Namespace = deployment.GetNamespace()
	relationship.NamespaceRelationship = namespaceRelationship

	containerRelationship := new(v1.SecretContainerRelationship)
	containerRelationship.Id = container.GetId()
	containerRelationship.Path = embedded.GetPath()
	relationship.ContainerRelationships = append(relationship.ContainerRelationships, containerRelationship)

	deploymentRelationship := new(v1.SecretDeploymentRelationship)
	deploymentRelationship.Id = deployment.GetId()
	deploymentRelationship.Name = deployment.GetName()
	relationship.DeploymentRelationships = append(relationship.DeploymentRelationships, deploymentRelationship)

	return relationship
}

func toSecret(embedded *v1.EmbeddedSecret) *v1.Secret {
	// Build secret.
	secret := new(v1.Secret)
	secret.Id = embedded.GetId()
	secret.Name = embedded.GetName()

	return secret
}

func stapleRelationships(stapleTo, stapleFrom *v1.SecretRelationship) {
	stapleContainerRelationships(stapleTo, stapleFrom)
	stapleDeploymentRelationships(stapleTo, stapleFrom)
}

func removeRelationships(removeFrom, toRemove *v1.SecretRelationship) {
	removeContainerRelationships(removeFrom, toRemove)
	removeDeploymentRelationships(removeFrom, toRemove)
}

func stapleContainerRelationships(stapleTo, stapleFrom *v1.SecretRelationship) {
	for _, relationship := range stapleFrom.ContainerRelationships {
		stapleContainerRelationship(stapleTo, relationship)
	}
}

func stapleContainerRelationship(stapleTo *v1.SecretRelationship, relationship *v1.SecretContainerRelationship) {
	// Check if the relationship already exists, and replace it if so.
	currentLength := len(stapleTo.ContainerRelationships)

	// Search for where the relationships should be inserted since it is not present.
	pos := sort.Search(currentLength, func(n int) bool {
		return relationship.GetId() <= stapleTo.ContainerRelationships[n].GetId()
	})

	// Container id already present, so just overwrite.
	if pos != currentLength && relationship.GetId() == stapleTo.ContainerRelationships[pos].GetId() {
		stapleTo.ContainerRelationships[pos] = relationship
		return
	}

	// Replace slice with a new slice that has the new relationship inserted.
	stapleTo.ContainerRelationships = append(stapleTo.ContainerRelationships, nil)
	if pos != currentLength {
		copy(stapleTo.ContainerRelationships[pos+1:], stapleTo.ContainerRelationships[pos:])
	}
	stapleTo.ContainerRelationships[pos] = relationship
}

func stapleDeploymentRelationships(stapleTo, stapleFrom *v1.SecretRelationship) {
	for _, relationship := range stapleFrom.DeploymentRelationships {
		stapleDeploymentRelationship(stapleTo, relationship)
	}
}

func stapleDeploymentRelationship(stapleTo *v1.SecretRelationship, relationship *v1.SecretDeploymentRelationship) {
	// Check if the relationship already exists, and replace it if so.
	currentLength := len(stapleTo.DeploymentRelationships)

	// Search for where the relationships should be inserted since it is not present.
	pos := sort.Search(currentLength, func(n int) bool {
		return relationship.GetId() <= stapleTo.DeploymentRelationships[n].GetId()
	})

	// Deployment id already present, so just overwrite.
	if pos != currentLength && relationship.GetId() == stapleTo.DeploymentRelationships[pos].GetId() {
		stapleTo.DeploymentRelationships[pos] = relationship
		return
	}

	// Replace slice with a new slice that has the new relationship inserted.
	stapleTo.DeploymentRelationships = append(stapleTo.DeploymentRelationships, nil)
	if pos != currentLength {
		copy(stapleTo.DeploymentRelationships[pos+1:], stapleTo.DeploymentRelationships[pos:])
	}
	stapleTo.DeploymentRelationships[pos] = relationship
}

func removeContainerRelationships(removeFrom, toRemove *v1.SecretRelationship) {
	for _, relationship := range toRemove.ContainerRelationships {
		removeContainerRelationship(removeFrom, relationship)
	}
}

func removeContainerRelationship(removeFrom *v1.SecretRelationship, relationship *v1.SecretContainerRelationship) {
	// Search for the container relationship we want to remove.
	currentLength := len(removeFrom.ContainerRelationships)
	// Search for where the relationships should be inserted since it is not present.
	pos := sort.Search(currentLength, func(n int) bool {
		return relationship.GetId() <= removeFrom.ContainerRelationships[n].GetId()
	})

	// Container id already present, so just overwrite.
	if pos == currentLength || relationship.GetId() != removeFrom.ContainerRelationships[pos].GetId() {
		return
	}

	// Build a new slice without the container relationship we want to remove.
	removeFrom.ContainerRelationships = append(removeFrom.ContainerRelationships[:pos], removeFrom.ContainerRelationships[pos+1:]...)
}

func removeDeploymentRelationships(removeFrom, toRemove *v1.SecretRelationship) {
	for _, relationship := range toRemove.DeploymentRelationships {
		removeDeploymentRelationship(removeFrom, relationship)
	}
}

func removeDeploymentRelationship(removeFrom *v1.SecretRelationship, relationship *v1.SecretDeploymentRelationship) {
	// Search for the container relationship we want to remove.
	currentLength := len(removeFrom.DeploymentRelationships)
	// Search for where the relationships should be inserted since it is not present.
	pos := sort.Search(currentLength, func(n int) bool {
		return relationship.GetId() <= removeFrom.DeploymentRelationships[n].GetId()
	})

	// Deployment id already present, so just overwrite.
	if pos == currentLength || relationship.GetId() != removeFrom.DeploymentRelationships[pos].GetId() {
		return
	}

	// Build a new slice without the container relationship we want to remove.
	removeFrom.DeploymentRelationships = append(removeFrom.DeploymentRelationships[:pos], removeFrom.DeploymentRelationships[pos+1:]...)

}
