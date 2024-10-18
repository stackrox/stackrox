package vuln

import (
	"context"
	"errors"
	"testing"

	"github.com/quay/claircore"
	"github.com/stackrox/rox/scanner/datastore/postgres/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDistManager(t *testing.T) {
	ctx := context.Background()
	store := mocks.NewMockMatcherStore(gomock.NewController(t))
	m := &distManager{
		store: store,
	}

	dists := []claircore.Distribution{
		{
			DID:       "rhel",
			VersionID: "8",
			Version:   "8",
		},
		{
			DID:       "rhel",
			VersionID: "9",
			Version:   "9",
		},
		{
			DID:       "ubuntu",
			VersionID: "22.04",
			Version:   "22.04 (Jammy)",
		},
		{
			DID:       "debian",
			VersionID: "10",
			Version:   "10 (buster)",
		},
		{
			DID:       "alpine",
			VersionID: "",
			Version:   "3.17",
		},
		{
			DID:       "alpine",
			VersionID: "",
			Version:   "3.18",
		},
	}

	// Get nothing.
	assert.Nil(t, m.get())

	// Successful fetch.
	store.EXPECT().Distributions(gomock.Any()).Return(dists, nil)
	err := m.update(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, dists, m.get())

	// Unsuccessful should return same dists as before.
	store.EXPECT().Distributions(gomock.Any()).Return(nil, errors.New("error"))
	err = m.update(ctx)
	assert.Error(t, err)
	assert.ElementsMatch(t, dists, m.get())
}
