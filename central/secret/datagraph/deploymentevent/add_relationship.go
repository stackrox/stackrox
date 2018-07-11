package index

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// To add a relationship, we need to update relationships currently present in storage, as well
// as add any new relationships. Since types of relationships are stored as slices, we need to
// search the current slice for where to update or add any new relationships. To make this work,
// all relationship types stored in a SecretRelationship need to be kept sorted.
func addRelationship(storage store.Store, relationshipToAdd *v1.SecretRelationship) (*v1.SecretRelationship, error) {
	relationshipToStore, exists, err := storage.GetRelationship(relationshipToAdd.GetId())
	if err != nil {
		return nil, err
	}

	if exists {
		stapleNewRelationships(relationshipToStore, relationshipToAdd)
	} else {
		relationshipToStore = relationshipToAdd
	}
	return relationshipToStore, nil
}

func stapleNewRelationships(stapleTo, stapleFrom *v1.SecretRelationship) {
	stapleContainerRelationships(stapleTo, stapleFrom)
	stapleDeploymentRelationships(stapleTo, stapleFrom)
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
	var newSlice []*v1.SecretContainerRelationship
	if pos != 0 {
		newSlice = stapleTo.ContainerRelationships[0:pos]
	}
	newSlice = append(newSlice, relationship)
	if pos != currentLength {
		newSlice = append(newSlice, stapleTo.ContainerRelationships[pos:currentLength]...)
	}
	stapleTo.ContainerRelationships = newSlice
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
	var newSlice []*v1.SecretDeploymentRelationship
	if pos != 0 {
		newSlice = stapleTo.DeploymentRelationships[0:pos]
	}
	newSlice = append(newSlice, relationship)
	if pos != currentLength {
		newSlice = append(newSlice, stapleTo.DeploymentRelationships[pos:currentLength]...)
	}
	stapleTo.DeploymentRelationships = newSlice
}
