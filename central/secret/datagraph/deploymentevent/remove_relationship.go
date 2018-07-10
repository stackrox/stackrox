package index

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// To remove a relationship, we need search the SecretRelationship in storage. Since the relationships
// are kept sorted, we can use the sort.Search function to find matching relationships.
func removeRelationship(storage store.Store, relationshipToRemove *v1.SecretRelationship) (*v1.SecretRelationship, error) {
	relationshipToStore, exists, err := storage.GetRelationship(relationshipToRemove.GetId())
	if err != nil {
		return nil, err
	}

	if exists {
		removeRelationships(relationshipToStore, relationshipToRemove)
	} else {
		relationshipToStore = relationshipToRemove
	}
	return relationshipToStore, nil
}

func removeRelationships(removeFrom, toRemove *v1.SecretRelationship) {
	removeContainerRelationships(removeFrom, toRemove)
	removeDeploymentRelationships(removeFrom, toRemove)
}

func removeContainerRelationships(removeFrom, toRemove *v1.SecretRelationship) {
	for _, relationship := range toRemove.ContainerRelationships {
		// Search for the container relationship we want to remove.
		currentLength := len(removeFrom.ContainerRelationships)
		pos := sort.Search(currentLength, func(n int) bool {
			return relationship.GetId() == removeFrom.ContainerRelationships[n].GetId()
		})
		if pos == currentLength {
			// Relationship isn't present, so nothing to do.
			return
		}

		// Build a new slice without the container relationship we want to remove.
		var newSlice []*v1.SecretContainerRelationship
		if pos > 0 {
			newSlice = removeFrom.ContainerRelationships[0:pos]
		}
		if pos < currentLength-1 {
			newSlice = append(newSlice, removeFrom.ContainerRelationships[pos+1:currentLength]...)
		}
		removeFrom.ContainerRelationships = newSlice
	}
}

func removeDeploymentRelationships(removeFrom, toRemove *v1.SecretRelationship) {
	for _, relationship := range toRemove.DeploymentRelationships {
		// Search for the deployment relationship we want to remove.
		currentLength := len(removeFrom.DeploymentRelationships)
		pos := sort.Search(currentLength, func(n int) bool {
			return relationship.GetId() == removeFrom.DeploymentRelationships[n].GetId()
		})
		if pos == currentLength {
			// Relationship isn't present, so nothing to do.
			return
		}

		// Build a new slice without the deployment relationship we want to remove.
		var newSlice []*v1.SecretDeploymentRelationship
		if pos > 0 {
			newSlice = removeFrom.DeploymentRelationships[0:pos]
		}
		if pos < currentLength-1 {
			newSlice = append(newSlice, removeFrom.DeploymentRelationships[pos+1:currentLength]...)
		}
		removeFrom.DeploymentRelationships = newSlice
	}
}
