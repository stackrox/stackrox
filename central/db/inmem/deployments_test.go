package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
)

func TestDeployments(t *testing.T) {
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

func TestGetDeployments(t *testing.T) {
	d0 := &v1.Deployment{
		Id:        "fooID",
		Name:      "foo",
		Version:   "100",
		Type:      "Replicated",
		Replicas:  10,
		UpdatedAt: ptypes.TimestampNow(),
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Sha: "04a094fe844e055828cb2d64ead6bd3eb4257e7c7b5d1e2af0da89fa20472cf4",
					},
				},
			},
		},
	}
	d1 := &v1.Deployment{
		Id:        "barID",
		Name:      "bar",
		Version:   "400",
		Type:      "Global",
		UpdatedAt: ptypes.TimestampNow(),
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Sha: "5b1e27e74327764cee6db966f5b624fbfbb6ce280754b575ff78cd940a43196f",
					},
				},
			},
		},
	}
	d2 := &v1.Deployment{
		Id:        "farID",
		Name:      "far",
		Version:   "333",
		Type:      "Replicated",
		Replicas:  1,
		UpdatedAt: ptypes.TimestampNow(),
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Sha: "25baa3ba19031d81309549af43f75c45aaaab318f34f5e4d5380a9fea304dddb",
					},
				},
			},
		},
	}
	storedDeployments := []*v1.Deployment{d0, d1, d2}

	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := New(persistent)

	for _, d := range storedDeployments {
		assert.NoError(t, storage.AddDeployment(d))
	}

	// Get all
	deployments, err := storage.GetDeployments(&v1.GetDeploymentsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, []*v1.Deployment{d1, d2, d0}, deployments)

	// Filter by name
	deployments, err = storage.GetDeployments(&v1.GetDeploymentsRequest{
		Name: []string{"foo", "bar"},
	})
	assert.Nil(t, err)
	assert.Equal(t, []*v1.Deployment{d1, d0}, deployments)

	// Filter by type
	deployments, err = storage.GetDeployments(&v1.GetDeploymentsRequest{
		Type: []string{"Global"},
	})
	assert.Nil(t, err)
	assert.Equal(t, []*v1.Deployment{d1}, deployments)

	// Filter by image sha
	deployments, err = storage.GetDeployments(&v1.GetDeploymentsRequest{
		ImageSha: []string{"25baa3ba19031d81309549af43f75c45aaaab318f34f5e4d5380a9fea304dddb",
			"not a sha"},
	})
	assert.Nil(t, err)
	assert.Equal(t, []*v1.Deployment{d2}, deployments)
}
