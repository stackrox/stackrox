package datastore

import (
	"context"

	"github.com/stackrox/rox/central/teams/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	_ DataStore = (*dataStoreImpl)(nil)
)

type dataStoreImpl struct {
	store store.Store
}

func (d *dataStoreImpl) GetTeam(ctx context.Context, id string) (*storage.Team, bool, error) {
	return d.store.Get(ctx, id)
}

func (d *dataStoreImpl) ListTeams(ctx context.Context) ([]*storage.Team, error) {
	return d.store.GetAll(ctx)
}

func (d *dataStoreImpl) AddTeam(ctx context.Context, team *storage.Team) (*storage.Team, error) {
	team.Id = uuid.NewV4().String()
	return team, d.store.Upsert(ctx, team)
}

func (d *dataStoreImpl) GetTeamsByName(ctx context.Context, names ...string) ([]*storage.Team, error) {
	namesSet := set.NewStringSet(names...)
	var teams []*storage.Team
	err := d.store.Walk(ctx, func(obj *storage.Team) error {
		if namesSet.Contains(obj.GetName()) {
			teams = append(teams, obj)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return teams, nil
}
