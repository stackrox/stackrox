package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
)

func TestDeployments(t *testing.T) {
	t.Parallel()

	deployments := []*v1.Deployment{
		{
			Id:        "fooID",
			Name:      "foo",
			Version:   "100",
			Type:      "Replicated",
			UpdatedAt: ptypes.TimestampNow(),
		},
		{
			Id:        "barID",
			Name:      "bar",
			Version:   "400",
			Type:      "Global",
			UpdatedAt: ptypes.TimestampNow(),
		},
	}

	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := New(persistent)

	// Test Add
	for _, d := range deployments {
		assert.NoError(t, storage.AddDeployment(d))
	}

	// Verify insertion multiple times does not deadlock and causes an error
	for _, d := range deployments {
		assert.Error(t, storage.AddDeployment(d))
	}

	for _, d := range deployments {
		got, exists, err := storage.GetDeployment(d.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, d)
	}

	// Test Update
	for _, d := range deployments {
		d.UpdatedAt = ptypes.TimestampNow()
		d.Version += "0"
	}

	for _, d := range deployments {
		assert.NoError(t, storage.UpdateDeployment(d))
	}

	for _, d := range deployments {
		got, exists, err := storage.GetDeployment(d.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, d)
	}

	// Test Remove
	for _, d := range deployments {
		assert.NoError(t, storage.RemoveDeployment(d.GetId()))
	}

	for _, d := range deployments {
		_, exists, err := storage.GetDeployment(d.GetId())
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}
