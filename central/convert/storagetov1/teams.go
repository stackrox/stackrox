package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Team converts storage.Team to v1.Team.
func Team(team *storage.Team) *v1.Team {
	return &v1.Team{
		Id:   team.GetId(),
		Name: team.GetName(),
	}
}

// Teams converts []storage.Team to []v1.Team.
func Teams(teams []*storage.Team) []*v1.Team {
	converted := make([]*v1.Team, 0, len(teams))
	for _, team := range teams {
		converted = append(converted, Team(team))
	}
	return converted
}
