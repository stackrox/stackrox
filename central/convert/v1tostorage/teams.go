package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Team converts v1.Team to storage.Team.
func Team(team *v1.Team) *storage.Team {
	return &storage.Team{
		Id:   team.GetId(),
		Name: team.GetName(),
	}
}

// Teams converts []v1.Team to []storage.Team.
func Teams(teams []*v1.Team) []*storage.Team {
	converted := make([]*storage.Team, 0, len(teams))
	for _, team := range teams {
		converted = append(converted, Team(team))
	}
	return converted
}
